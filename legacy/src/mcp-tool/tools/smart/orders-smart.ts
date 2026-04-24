import { z } from 'zod';
import { McpTool, MiddlewareContext } from '../../types';
import { clientFromCtx, fetchAll, ok, err, parseDate } from './helpers';

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function nowIso(): string {
  return new Date().toISOString();
}

function priorPeriod(from: string, to: string): { from: string; to: string } {
  const msFrom = new Date(from).getTime();
  const msTo = new Date(to).getTime();
  const length = msTo - msFrom;
  return {
    from: new Date(msFrom - length).toISOString(),
    to: new Date(msTo - length).toISOString(),
  };
}

function pct(next: number, prev: number): number | null {
  if (prev === 0) return null;
  return Math.round(((next - prev) / prev) * 10000) / 100;
}

function median(sorted: number[]): number | null {
  if (!sorted.length) return null;
  const mid = Math.floor(sorted.length / 2);
  return sorted.length % 2 === 0 ? (sorted[mid - 1] + sorted[mid]) / 2 : sorted[mid];
}

function p90(sorted: number[]): number | null {
  if (!sorted.length) return null;
  const idx = Math.floor(sorted.length * 0.9);
  return sorted[Math.min(idx, sorted.length - 1)];
}

function hoursBetween(a: string | null | undefined, b: string | null | undefined): number | null {
  if (!a || !b) return null;
  const diff = new Date(b).getTime() - new Date(a).getTime();
  return diff < 0 ? null : Math.round((diff / 3_600_000) * 100) / 100;
}

// ---------------------------------------------------------------------------
// Tool 1: hrv_orders_summary
// ---------------------------------------------------------------------------

async function ordersSummaryHandler(ctx: MiddlewareContext) {
  try {
    const { date_from, date_to, compare_prior: doCompare } = ctx.params as {
      date_from?: string;
      date_to?: string;
      compare_prior?: boolean;
    };

    const from = parseDate(date_from, 30);
    const to = date_to ?? nowIso();
    let totalApiCalls = 0;

    const FIELDS =
      'id,total_price,financial_status,cancelled_status,cancel_reason,source_name,gateway_code,fulfillment_status,created_at,total_discounts,discount_codes,location_id';

    const client = clientFromCtx(ctx);

    const { items: orders, apiCalls } = await fetchAll<any>(
      client,
      '/com/orders.json',
      'orders',
      { created_at_min: from, created_at_max: to, status: 'any' },
      { fields: FIELDS }
    );
    totalApiCalls += apiCalls;

    const metrics = computeOrderSummaryMetrics(orders);

    let comparison: Record<string, any> | undefined;
    if (doCompare !== false) {
      const prior = priorPeriod(from, to);
      const { items: priorOrders, apiCalls: priorCalls } = await fetchAll<any>(
        client,
        '/com/orders.json',
        'orders',
        { created_at_min: prior.from, created_at_max: prior.to, status: 'any' },
        { fields: FIELDS }
      );
      totalApiCalls += priorCalls;
      const priorMetrics = computeOrderSummaryMetrics(priorOrders);

      comparison = {
        period: prior,
        total_orders_change_pct: pct(metrics.total_orders, priorMetrics.total_orders),
        total_revenue_change_pct: pct(metrics.total_revenue, priorMetrics.total_revenue),
        aov_change_pct: pct(metrics.aov, priorMetrics.aov),
        prior_metrics: priorMetrics,
      };
    }

    return ok({
      period: { from, to },
      ...metrics,
      ...(comparison ? { comparison } : {}),
      _meta: { api_calls_used: totalApiCalls, generated_at: nowIso() },
    });
  } catch (e: any) {
    return err(`hrv_orders_summary failed: ${e?.message ?? e}`);
  }
}

function computeOrderSummaryMetrics(orders: any[]) {
  let total_revenue = 0;
  const orders_by_status: Record<string, number> = {
    paid: 0,
    pending: 0,
    refunded: 0,
    cancelled: 0,
  };
  const orders_by_source: Record<string, number> = {
    web: 0,
    pos: 0,
    iphone: 0,
    android: 0,
    other: 0,
  };
  const cancel_reasons: Record<string, number> = {};
  let discount_order_count = 0;
  let total_discount_value = 0;

  for (const o of orders) {
    const price = parseFloat(o.total_price ?? '0') || 0;
    const fs = (o.financial_status ?? '').toLowerCase();
    const cs = (o.cancelled_status ?? '').toLowerCase();
    const src = (o.source_name ?? '').toLowerCase();
    const discountVal = parseFloat(o.total_discounts ?? '0') || 0;

    if (fs === 'paid' || fs === 'partially_paid') {
      total_revenue += price;
    }

    if (cs === 'cancelled') {
      orders_by_status.cancelled = (orders_by_status.cancelled ?? 0) + 1;
      if (o.cancel_reason) {
        const r = o.cancel_reason as string;
        cancel_reasons[r] = (cancel_reasons[r] ?? 0) + 1;
      }
    } else if (fs === 'refunded' || fs === 'partially_refunded') {
      orders_by_status.refunded = (orders_by_status.refunded ?? 0) + 1;
    } else if (fs === 'paid' || fs === 'partially_paid') {
      orders_by_status.paid = (orders_by_status.paid ?? 0) + 1;
    } else {
      orders_by_status.pending = (orders_by_status.pending ?? 0) + 1;
    }

    if (src === 'web' || src === '') {
      orders_by_source.web++;
    } else if (src === 'pos') {
      orders_by_source.pos++;
    } else if (src === 'iphone') {
      orders_by_source.iphone++;
    } else if (src === 'android') {
      orders_by_source.android++;
    } else {
      orders_by_source.other++;
    }

    if (discountVal > 0) {
      discount_order_count++;
      total_discount_value += discountVal;
    }
  }

  const total_orders = orders.length;
  const paid_count = orders_by_status.paid + orders_by_status.refunded;
  const aov = paid_count > 0 ? Math.round(total_revenue / paid_count) : 0;

  return {
    total_orders,
    total_revenue: Math.round(total_revenue),
    aov,
    orders_by_status,
    orders_by_source,
    cancel_reasons,
    discount_usage: {
      orders_with_discount: discount_order_count,
      total_discount_value: Math.round(total_discount_value),
    },
  };
}

// ---------------------------------------------------------------------------
// Tool 2: hrv_top_products
// ---------------------------------------------------------------------------

async function topProductsHandler(ctx: MiddlewareContext) {
  try {
    const { date_from, date_to, top_n } = ctx.params as {
      date_from?: string;
      date_to?: string;
      top_n?: number;
    };

    const from = parseDate(date_from, 30);
    const to = date_to ?? nowIso();
    const n = top_n ?? 10;

    const client = clientFromCtx(ctx);
    const { items: orders, apiCalls } = await fetchAll<any>(
      client,
      '/com/orders.json',
      'orders',
      { created_at_min: from, created_at_max: to, status: 'any' },
      { fields: 'id,line_items,created_at' }
    );

    const productMap = new Map<number, {
      product_id: number;
      title: string;
      total_quantity: number;
      total_revenue: number;
      variants: Map<number, { variant_id: number; title: string; qty: number; rev: number }>;
    }>();

    for (const order of orders) {
      const lineItems: any[] = order.line_items ?? [];
      for (const item of lineItems) {
        const pid = item.product_id as number;
        if (!pid) continue;
        const qty = item.quantity ?? 0;
        const rev = (parseFloat(item.price ?? '0') || 0) * qty;
        const vid = item.variant_id as number;

        if (!productMap.has(pid)) {
          productMap.set(pid, {
            product_id: pid,
            title: item.title ?? '',
            total_quantity: 0,
            total_revenue: 0,
            variants: new Map(),
          });
        }
        const p = productMap.get(pid)!;
        p.total_quantity += qty;
        p.total_revenue += rev;

        if (vid) {
          if (!p.variants.has(vid)) {
            p.variants.set(vid, { variant_id: vid, title: item.variant_title ?? '', qty: 0, rev: 0 });
          }
          const v = p.variants.get(vid)!;
          v.qty += qty;
          v.rev += rev;
        }
      }
    }

    const products = Array.from(productMap.values())
      .sort((a, b) => b.total_revenue - a.total_revenue)
      .slice(0, n)
      .map((p) => {
        const topVariants = Array.from(p.variants.values())
          .sort((a, b) => b.rev - a.rev)
          .slice(0, 3)
          .map((v) => ({
            variant_id: v.variant_id,
            title: v.title,
            total_quantity: v.qty,
            total_revenue: Math.round(v.rev),
          }));
        return {
          product_id: p.product_id,
          title: p.title,
          total_quantity: p.total_quantity,
          total_revenue: Math.round(p.total_revenue),
          variant_breakdown: topVariants,
        };
      });

    return ok({
      period: { from, to },
      top_n: n,
      products,
      _meta: { api_calls_used: apiCalls, generated_at: nowIso() },
    });
  } catch (e: any) {
    return err(`hrv_top_products failed: ${e?.message ?? e}`);
  }
}

// ---------------------------------------------------------------------------
// Tool 3: hrv_order_cycle_time
// ---------------------------------------------------------------------------

async function orderCycleTimeHandler(ctx: MiddlewareContext) {
  try {
    const { date_from, date_to } = ctx.params as { date_from?: string; date_to?: string };

    const from = parseDate(date_from, 30);
    const to = date_to ?? nowIso();

    const client = clientFromCtx(ctx);
    const { items: orders, apiCalls } = await fetchAll<any>(
      client,
      '/com/orders.json',
      'orders',
      { created_at_min: from, created_at_max: to, status: 'any' },
      { fields: 'id,created_at,confirmed_at,closed_at,financial_status,fulfillment_status' }
    );

    const timesToConfirm: number[] = [];
    const timesToClose: number[] = [];
    let stuck_unconfirmed = 0;
    let paid_not_fulfilled = 0;

    const nowMs = Date.now();
    const h48 = 48 * 3_600_000;
    const h24 = 24 * 3_600_000;

    for (const o of orders) {
      const createdMs = o.created_at ? new Date(o.created_at).getTime() : null;

      const confirmHours = hoursBetween(o.created_at, o.confirmed_at);
      if (confirmHours !== null) timesToConfirm.push(confirmHours);

      const closeHours = hoursBetween(o.created_at, o.closed_at);
      if (closeHours !== null) timesToClose.push(closeHours);

      if (createdMs && nowMs - createdMs > h48 && !o.confirmed_at) {
        stuck_unconfirmed++;
      }

      const fs = (o.financial_status ?? '').toLowerCase();
      const ful = (o.fulfillment_status ?? '').toLowerCase();
      if (
        (fs === 'paid' || fs === 'partially_paid') &&
        ful !== 'fulfilled' &&
        createdMs &&
        nowMs - createdMs > h24
      ) {
        paid_not_fulfilled++;
      }
    }

    timesToConfirm.sort((a, b) => a - b);
    timesToClose.sort((a, b) => a - b);

    return ok({
      period: { from, to },
      total_orders: orders.length,
      time_to_confirm_hours: {
        median: median(timesToConfirm),
        p90: p90(timesToConfirm),
        sample_size: timesToConfirm.length,
      },
      time_to_close_hours: {
        median: median(timesToClose),
        p90: p90(timesToClose),
        sample_size: timesToClose.length,
      },
      stuck_orders: {
        unconfirmed_gt_48h: stuck_unconfirmed,
        paid_not_fulfilled_gt_24h: paid_not_fulfilled,
      },
      _meta: { api_calls_used: apiCalls, generated_at: nowIso() },
    });
  } catch (e: any) {
    return err(`hrv_order_cycle_time failed: ${e?.message ?? e}`);
  }
}

// ---------------------------------------------------------------------------
// Tool definitions
// ---------------------------------------------------------------------------

export const orderSmartTools: McpTool[] = [
  {
    name: 'hrv_orders_summary',
    project: 'smart',
    description:
      'Aggregate orders summary for a date range: total orders, revenue, AOV, breakdown by status/source/cancel reason, discount usage. Optionally compare to the prior equal-length period.',
    schema: z.object({
      date_from: z.string().optional().describe('Start date ISO 8601 (default: 30 days ago)'),
      date_to: z.string().optional().describe('End date ISO 8601 (default: now)'),
      compare_prior: z.boolean().optional().describe('Fetch prior period and compute change percentages (default: true)'),
    }),
    httpMethod: 'GET',
    path: '/smart',
    scopes: ['com.read_orders'],
    customHandler: ordersSummaryHandler,
  },
  {
    name: 'hrv_top_products',
    project: 'smart',
    description:
      'Return top N products by revenue for a date range. Aggregates all order line items and includes variant-level breakdown (top 3 variants per product).',
    schema: z.object({
      date_from: z.string().optional().describe('Start date ISO 8601 (default: 30 days ago)'),
      date_to: z.string().optional().describe('End date ISO 8601 (default: now)'),
      top_n: z.number().optional().describe('Number of top products to return (default: 10)'),
    }),
    httpMethod: 'GET',
    path: '/smart',
    scopes: ['com.read_orders'],
    customHandler: topProductsHandler,
  },
  {
    name: 'hrv_order_cycle_time',
    project: 'smart',
    description:
      'Compute order processing cycle times (median & p90 hours to confirm and to close). Identifies stuck orders: unconfirmed >48h and paid-but-unfulfilled >24h.',
    schema: z.object({
      date_from: z.string().optional().describe('Start date ISO 8601 (default: 30 days ago)'),
      date_to: z.string().optional().describe('End date ISO 8601 (default: now)'),
    }),
    httpMethod: 'GET',
    path: '/smart',
    scopes: ['com.read_orders'],
    customHandler: orderCycleTimeHandler,
  },
];
