package smart

import (
	"context"
	"encoding/json"
	"math"
	"net/url"
	"sort"
	"time"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var readCustomers = []string{"com.read_customers"}

// --- quintile scoring helpers (RFM) -------------------------------------

// quintileScore returns a 1..5 score where ascending=true gives higher
// scores to larger values (used for frequency, monetary), and
// ascending=false flips it (used for recency where smaller=better).
//
// The algorithm mirrors the legacy TS: for each value, count values strictly
// below it, divide into 5 buckets.
func quintileScores(values []float64, ascending bool) map[float64]int {
	if len(values) == 0 {
		return map[float64]int{}
	}
	sorted := sortFloats(values)
	n := float64(len(sorted))
	out := map[float64]int{}
	for _, v := range values {
		rank := 0
		for _, x := range sorted {
			if x < v {
				rank++
			}
		}
		q := int(math.Floor(float64(rank)/n*5)) + 1
		if q > 5 {
			q = 5
		}
		score := q
		if !ascending {
			score = 6 - q
		}
		if existing, ok := out[v]; !ok || score > existing {
			out[v] = score
		}
	}
	return out
}

func assignScore(m map[float64]int, v float64) int {
	if s, ok := m[v]; ok {
		return s
	}
	return 1
}

type segment string

const (
	segChampions         segment = "Champions"
	segLoyal             segment = "Loyal"
	segPotentialLoyalist segment = "Potential_Loyalists"
	segNew               segment = "New"
	segAtRisk            segment = "At_Risk"
	segHibernating       segment = "Hibernating"
	segLost              segment = "Lost"
	segOthers            segment = "Others"
)

func classifySegment(r, f, m int) segment {
	switch {
	case r >= 4 && f >= 4 && m >= 4:
		return segChampions
	case r >= 3 && f >= 4 && m >= 3:
		return segLoyal
	case r >= 4 && f <= 3 && m <= 3:
		return segPotentialLoyalist
	case r >= 4 && f == 1:
		return segNew
	case r <= 2 && f >= 3 && m >= 3:
		return segAtRisk
	case r <= 2 && f <= 2:
		return segHibernating
	case r == 1 && f == 1:
		return segLost
	default:
		return segOthers
	}
}

var actionSuggestions = map[segment]string{
	segChampions:         "Reward loyalty; ask for reviews; make them brand advocates.",
	segLoyal:             "Upsell higher-value products; offer loyalty programme.",
	segPotentialLoyalist: "Offer membership or loyalty programme to deepen engagement.",
	segNew:               "Onboard well; provide early support and product education.",
	segAtRisk:            "Send win-back campaigns; share what's new; offer discount.",
	segHibernating:       "Reactivate with a compelling offer; survey for feedback.",
	segLost:              "Try to revive with major incentive; otherwise accept churn.",
	segOthers:            "Analyse individually; no clear pattern detected.",
}

var allSegments = []segment{
	segChampions, segLoyal, segPotentialLoyalist, segNew,
	segAtRisk, segHibernating, segLost, segOthers,
}

// --- tool ---------------------------------------------------------------

func init() {
	tools.Register(&tools.Tool{
		Name:  "hrv_customer_segments",
		Short: "RFM analysis: segment customers into Champions/Loyal/At_Risk/etc.",
		Long: `RFM analysis using quintile scoring. Classifies every customer into
Champions, Loyal, Potential_Loyalists, New, At_Risk, Hibernating, Lost,
or Others, and returns counts + revenue metrics + an action suggestion
per segment.`,
		Category: tools.CatSmart,
		Scopes:   readCustomers,
		Flags: []tools.Flag{
			{Name: "min_orders", Type: tools.FlagInt, Description: "Minimum order count required to include a customer (default 0)"},
		},
		Handler: customerSegmentsHandler,
	})
}

func customerSegmentsHandler(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
	minOrders := in.Int("min_orders")

	fetched, err := fetchAll(ctx, deps.Client, "/com/customers.json", "customers", url.Values{},
		fetchOpts{Fields: "id,email,first_name,last_name,orders_count,total_spent,last_order_date,created_at,tags"})
	if err != nil {
		return nil, err
	}
	if len(fetched.Items) == 0 {
		return tools.NewJSON(map[string]any{
			"segments":        []any{},
			"total_customers": 0,
			"_meta":           map[string]any{"api_calls": fetched.APICalls},
		})
	}

	type record struct {
		Recency   float64
		Frequency float64
		Monetary  float64
	}
	records := make([]record, 0, len(fetched.Items))
	nowMs := time.Now().UnixMilli()

	for _, raw := range fetched.Items {
		var c struct {
			OrdersCount   int             `json:"orders_count"`
			TotalSpent    json.RawMessage `json:"total_spent"`
			LastOrderDate string          `json:"last_order_date"`
			CreatedAt     string          `json:"created_at"`
		}
		_ = json.Unmarshal(raw, &c)
		if c.OrdersCount < minOrders {
			continue
		}
		ts := c.LastOrderDate
		if ts == "" {
			ts = c.CreatedAt
		}
		var recency float64
		if t, err := time.Parse(time.RFC3339, ts); err == nil {
			recency = math.Floor(float64(nowMs-t.UnixMilli()) / 86_400_000.0)
		}
		records = append(records, record{
			Recency:   recency,
			Frequency: float64(c.OrdersCount),
			Monetary:  parseFloatOr(c.TotalSpent, 0),
		})
	}

	rs := make([]float64, len(records))
	fs := make([]float64, len(records))
	ms := make([]float64, len(records))
	for i, r := range records {
		rs[i], fs[i], ms[i] = r.Recency, r.Frequency, r.Monetary
	}
	recencyMap := quintileScores(rs, false)
	freqMap := quintileScores(fs, true)
	monMap := quintileScores(ms, true)

	type bucket struct{ Count, Orders int; Revenue float64 }
	buckets := map[segment]*bucket{}
	for _, s := range allSegments {
		buckets[s] = &bucket{}
	}

	for _, rec := range records {
		r := assignScore(recencyMap, rec.Recency)
		f := assignScore(freqMap, rec.Frequency)
		m := assignScore(monMap, rec.Monetary)
		seg := classifySegment(r, f, m)
		b := buckets[seg]
		b.Count++
		b.Revenue += rec.Monetary
		b.Orders += int(rec.Frequency)
	}

	total := len(records)
	type segOut struct {
		Name             string  `json:"name"`
		Count            int     `json:"count"`
		Pct              float64 `json:"pct"`
		TotalRevenue     float64 `json:"total_revenue"`
		AvgOrderValue    float64 `json:"avg_order_value"`
		ActionSuggestion string  `json:"action_suggestion"`
	}
	segments := make([]segOut, 0, len(allSegments))
	for _, s := range allSegments {
		b := buckets[s]
		if b.Count == 0 {
			continue
		}
		p := 0.0
		if total > 0 {
			p = roundToDecimals(float64(b.Count)/float64(total)*100, 1)
		}
		aov := 0.0
		if b.Orders > 0 {
			aov = roundToDecimals(b.Revenue/float64(b.Orders), 2)
		}
		segments = append(segments, segOut{
			Name:             string(s),
			Count:            b.Count,
			Pct:              p,
			TotalRevenue:     roundToDecimals(b.Revenue, 2),
			AvgOrderValue:    aov,
			ActionSuggestion: actionSuggestions[s],
		})
	}
	sort.SliceStable(segments, func(i, j int) bool {
		// preserve declared order
		return segmentIndex(segments[i].Name) < segmentIndex(segments[j].Name)
	})

	return tools.NewJSON(map[string]any{
		"segments":        segments,
		"total_customers": total,
		"_meta": map[string]any{
			"api_calls":         fetched.APICalls,
			"min_orders_filter": minOrders,
			"generated_at":      nowISO(),
		},
	})
}

func segmentIndex(name string) int {
	for i, s := range allSegments {
		if string(s) == name {
			return i
		}
	}
	return len(allSegments)
}

func roundToDecimals(v float64, decimals int) float64 {
	p := math.Pow10(decimals)
	return math.Round(v*p) / p
}
