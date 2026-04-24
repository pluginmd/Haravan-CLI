package smart

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"time"

	"github.com/pluginmd/haravan-cli/internal/client"
	"github.com/pluginmd/haravan-cli/internal/logger"
)

// The smart tools re-implement the pagination + throttling loop because they
// need fine-grained control (custom fields, max-page caps, inline throttling)
// that the generic paginator in internal/client deliberately hides.

const smartPageSize = 50

// fetchAll pages through a list endpoint, accumulating items from resourceKey.
// opts.Fields restricts the returned columns to save API quota; opts.MaxPages
// caps the walk (0 = default safety cap).
type fetchOpts struct {
	Fields   string
	MaxPages int
}

type fetched struct {
	Items    []json.RawMessage
	APICalls int
}

func fetchAll(ctx context.Context, c *client.Client, path, resourceKey string, params url.Values, opts fetchOpts) (fetched, error) {
	maxPages := opts.MaxPages
	if maxPages == 0 {
		maxPages = 100
	}
	if params == nil {
		params = url.Values{}
	}
	params.Set("limit", strconv.Itoa(smartPageSize))
	if opts.Fields != "" {
		params.Set("fields", opts.Fields)
	}

	var out fetched
	for page := 1; page <= maxPages; page++ {
		params.Set("page", strconv.Itoa(page))
		resp, err := c.Get(ctx, path, params)
		if err != nil {
			return out, err
		}
		out.APICalls++

		top := map[string]json.RawMessage{}
		if err := json.Unmarshal(resp.Body, &top); err != nil {
			return out, fmt.Errorf("decode %s: %w", path, err)
		}
		arr, ok := top[resourceKey]
		if !ok {
			break
		}
		var items []json.RawMessage
		if err := json.Unmarshal(arr, &items); err != nil {
			return out, fmt.Errorf("decode %s[%s]: %w", path, resourceKey, err)
		}
		out.Items = append(out.Items, items...)
		logger.Debugf("smart.fetchAll %s page=%d got=%d total=%d", path, page, len(items), len(out.Items))

		if len(items) < smartPageSize {
			break
		}
		// Cheap throttle to respect the leaky bucket. The client's own
		// bucket handles hard-limit waits; this just paces us.
		throttle(ctx, resp.RateLimitUsed)
	}
	return out, nil
}

func throttle(ctx context.Context, used float64) {
	dur := 250 * time.Millisecond
	if used > 60 {
		dur = time.Second
	}
	t := time.NewTimer(dur)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
	}
}

// ---------- date / number helpers ---------------------------------------

// parseDate returns input if non-empty, otherwise now-<daysAgo> as an ISO-8601 UTC timestamp.
func parseDate(input string, daysAgo int) string {
	if input != "" {
		return input
	}
	return time.Now().UTC().AddDate(0, 0, -daysAgo).Format(time.RFC3339)
}

func nowISO() string { return time.Now().UTC().Format(time.RFC3339) }

// priorPeriod returns the equal-length window immediately preceding [from, to].
func priorPeriod(from, to string) (string, string) {
	f, _ := time.Parse(time.RFC3339, from)
	t, _ := time.Parse(time.RFC3339, to)
	length := t.Sub(f)
	return f.Add(-length).UTC().Format(time.RFC3339), t.Add(-length).UTC().Format(time.RFC3339)
}

// pct returns (next/prev - 1) * 100 rounded to 2 decimals. Returns nil if prev is 0.
func pct(next, prev float64) *float64 {
	if prev == 0 {
		return nil
	}
	v := math.Round((next-prev)/prev*10000) / 100
	return &v
}

func median(sorted []float64) *float64 {
	n := len(sorted)
	if n == 0 {
		return nil
	}
	mid := n / 2
	var v float64
	if n%2 == 0 {
		v = (sorted[mid-1] + sorted[mid]) / 2
	} else {
		v = sorted[mid]
	}
	return &v
}

func p90(sorted []float64) *float64 {
	n := len(sorted)
	if n == 0 {
		return nil
	}
	idx := int(math.Floor(float64(n) * 0.9))
	if idx >= n {
		idx = n - 1
	}
	v := sorted[idx]
	return &v
}

// hoursBetween returns (b-a) in hours, rounded to 2 decimals. Returns nil
// if either timestamp is empty or a is after b.
func hoursBetween(a, b string) *float64 {
	if a == "" || b == "" {
		return nil
	}
	ta, err1 := time.Parse(time.RFC3339, a)
	tb, err2 := time.Parse(time.RFC3339, b)
	if err1 != nil || err2 != nil {
		return nil
	}
	diff := tb.Sub(ta)
	if diff < 0 {
		return nil
	}
	v := math.Round(diff.Hours()*100) / 100
	return &v
}

// parseFloatOr parses a JSON-encoded number or string-wrapped number.
func parseFloatOr(raw json.RawMessage, fallback float64) float64 {
	if len(raw) == 0 {
		return fallback
	}
	// Try as number first.
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return n
	}
	// Then as string.
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		if s == "" {
			return fallback
		}
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			return f
		}
	}
	return fallback
}

// sortFloats returns a sorted copy (ascending).
func sortFloats(vs []float64) []float64 {
	out := append([]float64(nil), vs...)
	sort.Float64s(out)
	return out
}
