package smart

import (
	"context"
	"encoding/json"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var readOrders = []string{"com.read_orders"}

func init() {
	registerOrdersSummary()
	registerTopProducts()
	registerOrderCycleTime()
}

// ---------- hrv_orders_summary -----------------------------------------

func registerOrdersSummary() {
	tools.Register(&tools.Tool{
		Name:     "hrv_orders_summary",
		Short:    "Aggregate revenue/AOV/status breakdown with optional prior-period comparison",
		Category: tools.CatSmart,
		Scopes:   readOrders,
		Flags: []tools.Flag{
			{Name: "date_from", Type: tools.FlagDateTimeISO, Description: "Start date ISO 8601 (default: 30 days ago)"},
			{Name: "date_to", Type: tools.FlagDateTimeISO, Description: "End date ISO 8601 (default: now)"},
			{Name: "compare_prior", Type: tools.FlagBool, Description: "Fetch prior equal-length window and add comparison (default true)"},
		},
		Handler: ordersSummaryHandler,
	})
}

const orderSummaryFields = "id,total_price,financial_status,cancelled_status,cancel_reason,source_name,gateway_code,fulfillment_status,created_at,total_discounts,discount_codes,location_id"

func ordersSummaryHandler(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
	from := parseDate(in.String("date_from"), 30)
	to := in.StringOr("date_to", nowISO())
	doCompare := true
	if in.Has("compare_prior") {
		doCompare = in.Bool("compare_prior")
	}

	q := url.Values{}
	q.Set("created_at_min", from)
	q.Set("created_at_max", to)
	q.Set("status", "any")
	fetched, err := fetchAll(ctx, deps.Client, "/com/orders.json", "orders", q, fetchOpts{Fields: orderSummaryFields})
	if err != nil {
		return nil, err
	}
	metrics := orderSummaryMetrics(fetched.Items)
	totalCalls := fetched.APICalls

	out := map[string]any{
		"period": map[string]string{"from": from, "to": to},
	}
	for k, v := range metrics {
		out[k] = v
	}

	if doCompare {
		priorFrom, priorTo := priorPeriod(from, to)
		pq := url.Values{}
		pq.Set("created_at_min", priorFrom)
		pq.Set("created_at_max", priorTo)
		pq.Set("status", "any")
		prior, err := fetchAll(ctx, deps.Client, "/com/orders.json", "orders", pq, fetchOpts{Fields: orderSummaryFields})
		if err != nil {
			return nil, err
		}
		totalCalls += prior.APICalls
		priorMetrics := orderSummaryMetrics(prior.Items)
		out["comparison"] = map[string]any{
			"period":                  map[string]string{"from": priorFrom, "to": priorTo},
			"total_orders_change_pct": pct(float64(metrics["total_orders"].(int)), float64(priorMetrics["total_orders"].(int))),
			"total_revenue_change_pct": pct(metrics["total_revenue"].(float64), priorMetrics["total_revenue"].(float64)),
			"aov_change_pct":           pct(float64(metrics["aov"].(int)), float64(priorMetrics["aov"].(int))),
			"prior_metrics":            priorMetrics,
		}
	}
	out["_meta"] = map[string]any{
		"api_calls_used": totalCalls,
		"generated_at":   nowISO(),
	}
	return tools.NewJSON(out)
}

// orderSummaryMetrics computes the per-period block that both the primary and
// prior-period responses share. It returns ints where TS would have used
// numbers — we keep revenue as float64 until the final rounding to match the
// legacy behaviour (Math.round on total_revenue and aov).
func orderSummaryMetrics(orders []json.RawMessage) map[string]any {
	var totalRevenue float64
	byStatus := map[string]int{"paid": 0, "pending": 0, "refunded": 0, "cancelled": 0}
	bySource := map[string]int{"web": 0, "pos": 0, "iphone": 0, "android": 0, "other": 0}
	cancelReasons := map[string]int{}
	discountCount := 0
	var discountValue float64

	for _, raw := range orders {
		var o struct {
			TotalPrice       json.RawMessage `json:"total_price"`
			FinancialStatus  string          `json:"financial_status"`
			CancelledStatus  string          `json:"cancelled_status"`
			CancelReason     string          `json:"cancel_reason"`
			SourceName       string          `json:"source_name"`
			TotalDiscounts   json.RawMessage `json:"total_discounts"`
		}
		_ = json.Unmarshal(raw, &o)
		price := parseFloatOr(o.TotalPrice, 0)
		fs := strings.ToLower(o.FinancialStatus)
		cs := strings.ToLower(o.CancelledStatus)
		src := strings.ToLower(o.SourceName)
		disc := parseFloatOr(o.TotalDiscounts, 0)

		if fs == "paid" || fs == "partially_paid" {
			totalRevenue += price
		}

		switch {
		case cs == "cancelled":
			byStatus["cancelled"]++
			if o.CancelReason != "" {
				cancelReasons[o.CancelReason]++
			}
		case fs == "refunded" || fs == "partially_refunded":
			byStatus["refunded"]++
		case fs == "paid" || fs == "partially_paid":
			byStatus["paid"]++
		default:
			byStatus["pending"]++
		}

		switch src {
		case "web", "":
			bySource["web"]++
		case "pos":
			bySource["pos"]++
		case "iphone":
			bySource["iphone"]++
		case "android":
			bySource["android"]++
		default:
			bySource["other"]++
		}

		if disc > 0 {
			discountCount++
			discountValue += disc
		}
	}

	total := len(orders)
	paidCount := byStatus["paid"] + byStatus["refunded"]
	aov := 0
	if paidCount > 0 {
		aov = int(totalRevenue / float64(paidCount))
	}

	return map[string]any{
		"total_orders":     total,
		"total_revenue":    roundFloat(totalRevenue),
		"aov":              aov,
		"orders_by_status": byStatus,
		"orders_by_source": bySource,
		"cancel_reasons":   cancelReasons,
		"discount_usage": map[string]any{
			"orders_with_discount": discountCount,
			"total_discount_value": roundFloat(discountValue),
		},
	}
}

// ---------- hrv_top_products -------------------------------------------

func registerTopProducts() {
	tools.Register(&tools.Tool{
		Name:     "hrv_top_products",
		Short:    "Top N products by revenue with variant-level breakdown",
		Category: tools.CatSmart,
		Scopes:   readOrders,
		Flags: []tools.Flag{
			{Name: "date_from", Type: tools.FlagDateTimeISO, Description: "Start date ISO 8601 (default 30 days ago)"},
			{Name: "date_to", Type: tools.FlagDateTimeISO, Description: "End date ISO 8601 (default now)"},
			{Name: "top_n", Type: tools.FlagInt, Description: "Number of top products (default 10)"},
		},
		Handler: topProductsHandler,
	})
}

func topProductsHandler(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
	from := parseDate(in.String("date_from"), 30)
	to := in.StringOr("date_to", nowISO())
	n := in.Int("top_n")
	if n == 0 {
		n = 10
	}

	q := url.Values{}
	q.Set("created_at_min", from)
	q.Set("created_at_max", to)
	q.Set("status", "any")
	fetched, err := fetchAll(ctx, deps.Client, "/com/orders.json", "orders", q, fetchOpts{Fields: "id,line_items,created_at"})
	if err != nil {
		return nil, err
	}

	type variantAgg struct {
		ID, Qty  int64
		Title    string
		Revenue  float64
	}
	type productAgg struct {
		ID       int64
		Title    string
		Qty      int64
		Revenue  float64
		Variants map[int64]*variantAgg
	}
	products := map[int64]*productAgg{}

	for _, raw := range fetched.Items {
		var o struct {
			LineItems []struct {
				ProductID    int64           `json:"product_id"`
				VariantID    int64           `json:"variant_id"`
				Title        string          `json:"title"`
				VariantTitle string          `json:"variant_title"`
				Quantity     int64           `json:"quantity"`
				Price        json.RawMessage `json:"price"`
			} `json:"line_items"`
		}
		_ = json.Unmarshal(raw, &o)
		for _, li := range o.LineItems {
			if li.ProductID == 0 {
				continue
			}
			rev := parseFloatOr(li.Price, 0) * float64(li.Quantity)
			p, ok := products[li.ProductID]
			if !ok {
				p = &productAgg{ID: li.ProductID, Title: li.Title, Variants: map[int64]*variantAgg{}}
				products[li.ProductID] = p
			}
			p.Qty += li.Quantity
			p.Revenue += rev
			if li.VariantID != 0 {
				v, ok := p.Variants[li.VariantID]
				if !ok {
					v = &variantAgg{ID: li.VariantID, Title: li.VariantTitle}
					p.Variants[li.VariantID] = v
				}
				v.Qty += li.Quantity
				v.Revenue += rev
			}
		}
	}

	list := make([]*productAgg, 0, len(products))
	for _, p := range products {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Revenue > list[j].Revenue })
	if len(list) > n {
		list = list[:n]
	}

	type variantOut struct {
		VariantID     int64   `json:"variant_id"`
		Title         string  `json:"title"`
		TotalQuantity int64   `json:"total_quantity"`
		TotalRevenue  float64 `json:"total_revenue"`
	}
	type productOut struct {
		ProductID        int64        `json:"product_id"`
		Title            string       `json:"title"`
		TotalQuantity    int64        `json:"total_quantity"`
		TotalRevenue     float64      `json:"total_revenue"`
		VariantBreakdown []variantOut `json:"variant_breakdown"`
	}

	outList := make([]productOut, 0, len(list))
	for _, p := range list {
		vs := make([]*variantAgg, 0, len(p.Variants))
		for _, v := range p.Variants {
			vs = append(vs, v)
		}
		sort.Slice(vs, func(i, j int) bool { return vs[i].Revenue > vs[j].Revenue })
		if len(vs) > 3 {
			vs = vs[:3]
		}
		vouts := make([]variantOut, 0, len(vs))
		for _, v := range vs {
			vouts = append(vouts, variantOut{
				VariantID:     v.ID,
				Title:         v.Title,
				TotalQuantity: v.Qty,
				TotalRevenue:  roundFloat(v.Revenue),
			})
		}
		outList = append(outList, productOut{
			ProductID:        p.ID,
			Title:            p.Title,
			TotalQuantity:    p.Qty,
			TotalRevenue:     roundFloat(p.Revenue),
			VariantBreakdown: vouts,
		})
	}

	return tools.NewJSON(map[string]any{
		"period":   map[string]string{"from": from, "to": to},
		"top_n":    n,
		"products": outList,
		"_meta": map[string]any{
			"api_calls_used": fetched.APICalls,
			"generated_at":   nowISO(),
		},
	})
}

// ---------- hrv_order_cycle_time ---------------------------------------

func registerOrderCycleTime() {
	tools.Register(&tools.Tool{
		Name:     "hrv_order_cycle_time",
		Short:    "Median/p90 time-to-confirm and time-to-close, plus stuck-order counts",
		Category: tools.CatSmart,
		Scopes:   readOrders,
		Flags: []tools.Flag{
			{Name: "date_from", Type: tools.FlagDateTimeISO, Description: "Start date ISO 8601 (default 30 days ago)"},
			{Name: "date_to", Type: tools.FlagDateTimeISO, Description: "End date ISO 8601 (default now)"},
		},
		Handler: orderCycleTimeHandler,
	})
}

func orderCycleTimeHandler(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
	from := parseDate(in.String("date_from"), 30)
	to := in.StringOr("date_to", nowISO())

	q := url.Values{}
	q.Set("created_at_min", from)
	q.Set("created_at_max", to)
	q.Set("status", "any")
	fetched, err := fetchAll(ctx, deps.Client, "/com/orders.json", "orders", q,
		fetchOpts{Fields: "id,created_at,confirmed_at,closed_at,financial_status,fulfillment_status"})
	if err != nil {
		return nil, err
	}

	var confirmHours, closeHours []float64
	stuckUnconfirmed := 0
	paidNotFulfilled := 0
	now := time.Now().UTC()
	h48 := 48 * time.Hour
	h24 := 24 * time.Hour

	for _, raw := range fetched.Items {
		var o struct {
			CreatedAt         string `json:"created_at"`
			ConfirmedAt       string `json:"confirmed_at"`
			ClosedAt          string `json:"closed_at"`
			FinancialStatus   string `json:"financial_status"`
			FulfillmentStatus string `json:"fulfillment_status"`
		}
		_ = json.Unmarshal(raw, &o)

		if v := hoursBetween(o.CreatedAt, o.ConfirmedAt); v != nil {
			confirmHours = append(confirmHours, *v)
		}
		if v := hoursBetween(o.CreatedAt, o.ClosedAt); v != nil {
			closeHours = append(closeHours, *v)
		}
		if t, err := time.Parse(time.RFC3339, o.CreatedAt); err == nil {
			age := now.Sub(t)
			if age > h48 && o.ConfirmedAt == "" {
				stuckUnconfirmed++
			}
			fs := strings.ToLower(o.FinancialStatus)
			ful := strings.ToLower(o.FulfillmentStatus)
			if (fs == "paid" || fs == "partially_paid") && ful != "fulfilled" && age > h24 {
				paidNotFulfilled++
			}
		}
	}
	confirmSorted := sortFloats(confirmHours)
	closeSorted := sortFloats(closeHours)

	return tools.NewJSON(map[string]any{
		"period":       map[string]string{"from": from, "to": to},
		"total_orders": len(fetched.Items),
		"time_to_confirm_hours": map[string]any{
			"median":      median(confirmSorted),
			"p90":         p90(confirmSorted),
			"sample_size": len(confirmSorted),
		},
		"time_to_close_hours": map[string]any{
			"median":      median(closeSorted),
			"p90":         p90(closeSorted),
			"sample_size": len(closeSorted),
		},
		"stuck_orders": map[string]any{
			"unconfirmed_gt_48h":        stuckUnconfirmed,
			"paid_not_fulfilled_gt_24h": paidNotFulfilled,
		},
		"_meta": map[string]any{
			"api_calls_used": fetched.APICalls,
			"generated_at":   nowISO(),
		},
	})
}

// roundFloat rounds to the nearest integer (matches Math.round) while keeping
// a float64 type so JSON callers see a number, not an int.
func roundFloat(v float64) float64 {
	return float64(int64(v + 0.5))
}
