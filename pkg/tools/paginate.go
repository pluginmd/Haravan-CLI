package tools

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/pluginmd/haravan-cli/internal/client"
)

// Paginate walks a Haravan list endpoint fully and returns an envelope
// shaped like the server's own single-page response, with a _pagination
// meta block appended. It is a thin bridge from Handler code into the
// internal/client helper.
//
//	{
//	  "<resource_key>": [ ... all items ... ],
//	  "_pagination": { "pages_fetched": N, "total_fetched": M, "truncated": bool }
//	}
func Paginate(ctx context.Context, deps *Deps, path string, query url.Values, resourceKey string) (*Result, error) {
	p, err := client.PaginateAll(ctx, deps.Client, path, query, resourceKey)
	if err != nil {
		return nil, err
	}
	envelope := map[string]any{
		p.ResourceKey: p.Items,
		"_pagination": map[string]any{
			"pages_fetched": p.Pages,
			"total_fetched": len(p.Items),
			"truncated":     p.Truncated,
		},
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}
	return NewRaw(raw), nil
}
