import { z } from 'zod';
import { McpTool, MiddlewareContext } from '../../types';
import { clientFromCtx, fetchAll, ok, err } from './helpers';

// ---------------------------------------------------------------------------
// Quintile scoring helper (1–5)
// ---------------------------------------------------------------------------
function quintileScores(values: number[], ascending = true): Map<number, number> {
  const sorted = [...values].sort((a, b) => a - b);
  const n = sorted.length;
  const scores = new Map<number, number>();

  values.forEach((v) => {
    const rank = sorted.filter((x) => x < v).length;
    let q = Math.floor((rank / n) * 5) + 1;
    if (q > 5) q = 5;
    const finalScore = ascending ? q : 6 - q;
    const existing = scores.get(v);
    if (existing === undefined || finalScore > existing) {
      scores.set(v, finalScore);
    }
  });

  return scores;
}

function assignScore(scoreMap: Map<number, number>, value: number): number {
  return scoreMap.get(value) ?? 1;
}

// ---------------------------------------------------------------------------
// Segment classification
// ---------------------------------------------------------------------------
type Segment =
  | 'Champions'
  | 'Loyal'
  | 'Potential_Loyalists'
  | 'New'
  | 'At_Risk'
  | 'Hibernating'
  | 'Lost'
  | 'Others';

function classifySegment(r: number, f: number, m: number): Segment {
  if (r >= 4 && f >= 4 && m >= 4) return 'Champions';
  if (r >= 3 && f >= 4 && m >= 3) return 'Loyal';
  if (r >= 4 && f <= 3 && m <= 3) return 'Potential_Loyalists';
  if (r >= 4 && f === 1) return 'New';
  if (r <= 2 && f >= 3 && m >= 3) return 'At_Risk';
  if (r <= 2 && f <= 2) return 'Hibernating';
  if (r === 1 && f === 1) return 'Lost';
  return 'Others';
}

const ACTION_SUGGESTIONS: Record<Segment, string> = {
  Champions: 'Reward loyalty; ask for reviews; make them brand advocates.',
  Loyal: 'Upsell higher-value products; offer loyalty programme.',
  Potential_Loyalists: 'Offer membership or loyalty programme to deepen engagement.',
  New: 'Onboard well; provide early support and product education.',
  At_Risk: 'Send win-back campaigns; share what\'s new; offer discount.',
  Hibernating: 'Reactivate with a compelling offer; survey for feedback.',
  Lost: 'Try to revive with major incentive; otherwise accept churn.',
  Others: 'Analyse individually; no clear pattern detected.',
};

// ---------------------------------------------------------------------------
// hrv_customer_segments (RFM Analysis)
// ---------------------------------------------------------------------------
const customerSegmentsHandler = async (ctx: MiddlewareContext) => {
  try {
    const { min_orders = 0 } = ctx.params as { min_orders?: number };
    const client = clientFromCtx(ctx);
    const now = Date.now();

    const { items: customers, apiCalls } = await fetchAll<any>(
      client,
      '/com/customers.json',
      'customers',
      {},
      { fields: 'id,email,first_name,last_name,orders_count,total_spent,last_order_date,created_at,tags' }
    );

    if (!customers.length) {
      return ok({ segments: [], total_customers: 0, _meta: { api_calls: apiCalls } });
    }

    const filtered = customers.filter((c: any) => (c.orders_count ?? 0) >= min_orders);

    const records = filtered.map((c: any) => {
      const lastOrderDate = c.last_order_date
        ? new Date(c.last_order_date).getTime()
        : new Date(c.created_at).getTime();
      const recencyDays = Math.floor((now - lastOrderDate) / 86_400_000);
      const frequency: number = c.orders_count ?? 0;
      const monetary: number = parseFloat(c.total_spent ?? '0') || 0;
      return { customer: c, recencyDays, frequency, monetary };
    });

    const recencyMap = quintileScores(records.map((r) => r.recencyDays), false);
    const freqMap = quintileScores(records.map((r) => r.frequency), true);
    const monMap = quintileScores(records.map((r) => r.monetary), true);

    type Scored = { r: number; f: number; m: number; monetary: number; frequency: number; segment: Segment };

    const scored: Scored[] = records.map((rec) => {
      const r = assignScore(recencyMap, rec.recencyDays);
      const f = assignScore(freqMap, rec.frequency);
      const m = assignScore(monMap, rec.monetary);
      const segment = classifySegment(r, f, m);
      return { r, f, m, monetary: rec.monetary, frequency: rec.frequency, segment };
    });

    const segmentMap = new Map<Segment, { count: number; total_revenue: number; total_orders: number }>();
    const allSegments: Segment[] = [
      'Champions', 'Loyal', 'Potential_Loyalists', 'New', 'At_Risk', 'Hibernating', 'Lost', 'Others',
    ];
    allSegments.forEach((s) => segmentMap.set(s, { count: 0, total_revenue: 0, total_orders: 0 }));

    scored.forEach((s) => {
      const seg = segmentMap.get(s.segment)!;
      seg.count += 1;
      seg.total_revenue += s.monetary;
      seg.total_orders += s.frequency;
    });

    const total_customers = filtered.length;

    const segments = allSegments
      .map((name) => {
        const { count, total_revenue, total_orders } = segmentMap.get(name)!;
        const pctVal = total_customers > 0 ? parseFloat(((count / total_customers) * 100).toFixed(1)) : 0;
        const avg_order_value = total_orders > 0 ? parseFloat((total_revenue / total_orders).toFixed(2)) : 0;
        return {
          name,
          count,
          pct: pctVal,
          total_revenue: parseFloat(total_revenue.toFixed(2)),
          avg_order_value,
          action_suggestion: ACTION_SUGGESTIONS[name],
        };
      })
      .filter((s) => s.count > 0);

    return ok({
      segments,
      total_customers,
      _meta: {
        api_calls: apiCalls,
        min_orders_filter: min_orders,
        generated_at: new Date().toISOString(),
      },
    });
  } catch (e: any) {
    return err(`hrv_customer_segments failed: ${e.message}`);
  }
};

// ---------------------------------------------------------------------------
// Export
// ---------------------------------------------------------------------------
export const customerSmartTools: McpTool[] = [
  {
    name: 'hrv_customer_segments',
    project: 'smart',
    description:
      'RFM analysis: segments all customers into Champions, Loyal, Potential_Loyalists, New, At_Risk, Hibernating, Lost using quintile scoring. Returns segment breakdown with revenue metrics and action suggestions.',
    schema: z.object({
      min_orders: z.number().optional().describe('Minimum order count to include a customer (default: 0)'),
    }),
    httpMethod: 'GET',
    path: '/smart',
    scopes: ['com.read_customers'],
    customHandler: customerSegmentsHandler,
  },
];
