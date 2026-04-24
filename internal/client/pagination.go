package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"

	"github.com/pluginmd/haravan-cli/internal/logger"
)

// Default safety caps for auto-pagination.
const (
	DefaultPageLimit = 50
	MaxPages         = 20
	MaxRecords       = 5000
)

// Paginated is the envelope returned by PaginateAll. It mirrors the TS legacy
// shape so downstream formatters stay consistent.
type Paginated struct {
	ResourceKey string            `json:"-"`
	Items       []json.RawMessage `json:"-"`
	Pages       int               `json:"pages_fetched"`
	Truncated   bool              `json:"truncated"`
}

// PaginateAll walks a Haravan list endpoint page by page until the server
// returns a short page, the record cap is hit, or MaxPages is reached.
//
// It requires:
//   - path: the REST path (e.g. "/com/orders.json")
//   - params: base query string; page/limit will be injected
//   - resourceKey: the JSON field containing the array (e.g. "orders").
//     If empty, PaginateAll auto-detects the first top-level array field.
//
// The items are returned as raw JSON messages so the caller decides whether
// to decode into a typed slice or re-emit as JSON.
func PaginateAll(ctx context.Context, c *Client, path string, params url.Values, resourceKey string) (*Paginated, error) {
	if params == nil {
		params = url.Values{}
	}
	limit := DefaultPageLimit
	if v := params.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	params.Set("limit", strconv.Itoa(limit))

	startPage := 1
	if v := params.Get("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			startPage = n
		}
	}

	out := &Paginated{}

	for i := 0; i < MaxPages; i++ {
		pageNum := startPage + i
		params.Set("page", strconv.Itoa(pageNum))
		resp, err := c.Get(ctx, path, params)
		if err != nil {
			return nil, fmt.Errorf("pagination page %d: %w", pageNum, err)
		}

		top := map[string]json.RawMessage{}
		if err := json.Unmarshal(resp.Body, &top); err != nil {
			return nil, fmt.Errorf("pagination: decode top-level object: %w", err)
		}

		if resourceKey == "" {
			resourceKey = firstArrayKey(top)
			if resourceKey == "" {
				return nil, fmt.Errorf("pagination: response has no array field")
			}
		}
		out.ResourceKey = resourceKey

		var items []json.RawMessage
		if err := json.Unmarshal(top[resourceKey], &items); err != nil {
			return nil, fmt.Errorf("pagination: decode array %q: %w", resourceKey, err)
		}

		out.Items = append(out.Items, items...)
		out.Pages = i + 1

		logger.Debugf("pagination: page %d, %d items (total %d)", pageNum, len(items), len(out.Items))

		if len(items) < limit {
			break
		}
		if len(out.Items) >= MaxRecords {
			out.Truncated = true
			logger.Warnf("pagination: hit MaxRecords=%d cap", MaxRecords)
			break
		}
	}
	return out, nil
}

func firstArrayKey(m map[string]json.RawMessage) string {
	for k, v := range m {
		if len(v) > 0 && v[0] == '[' {
			return k
		}
	}
	return ""
}
