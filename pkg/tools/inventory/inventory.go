package inventory

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var (
	readInventory  = []string{"com.read_inventories"}
	writeInventory = []string{"com.write_inventories"}
)

func init() {
	registerAdjustmentsList()
	registerAdjustmentsCount()
	registerAdjustmentsGet()
	registerAdjustOrSet()
	registerLocations()
}

func registerAdjustmentsList() {
	tools.Register(&tools.Tool{
		Name:     "haravan_inventory_adjustments_list",
		Short:    "List inventory adjustments",
		Long:     "List inventory adjustments with pagination and date filters.",
		Category: tools.CatInventory,
		Scopes:   readInventory,
		Flags: []tools.Flag{
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
			{Name: "since_id", Type: tools.FlagInt64, Description: "Results after this ID"},
			{Name: "created_at_min", Type: tools.FlagDateTimeISO, Description: "Created after (ISO 8601)"},
			{Name: "created_at_max", Type: tools.FlagDateTimeISO, Description: "Created before (ISO 8601)"},
			{Name: "fetch_all", Type: tools.FlagBool, Description: "Auto-paginate to fetch every page"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "page", "limit", "since_id", "created_at_min", "created_at_max")
			if in.Bool("fetch_all") {
				return tools.Paginate(ctx, deps, "/com/inventories/adjustments.json", q, "adjustments")
			}
			resp, err := deps.Client.Get(ctx, "/com/inventories/adjustments.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerAdjustmentsCount() {
	tools.Register(&tools.Tool{
		Name:     "haravan_inventory_adjustments_count",
		Short:    "Count inventory adjustments",
		Category: tools.CatInventory,
		Scopes:   readInventory,
		Flags: []tools.Flag{
			{Name: "created_at_min", Type: tools.FlagDateTimeISO, Description: "Created after (ISO 8601)"},
			{Name: "created_at_max", Type: tools.FlagDateTimeISO, Description: "Created before (ISO 8601)"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "created_at_min", "created_at_max")
			resp, err := deps.Client.Get(ctx, "/com/inventories/adjustments/count.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerAdjustmentsGet() {
	tools.Register(&tools.Tool{
		Name:     "haravan_inventory_adjustments_get",
		Short:    "Get a single inventory adjustment by ID",
		Category: tools.CatInventory,
		Scopes:   readInventory,
		Flags: []tools.Flag{
			{Name: "adjustment_id", Type: tools.FlagInt64, Description: "Adjustment ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/inventories/adjustments/%d.json", in.Int64("adjustment_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerAdjustOrSet() {
	tools.Register(&tools.Tool{
		Name:     "haravan_inventory_adjust_or_set",
		Short:    "Create inventory adjustment (adjust=delta, set=replace)",
		Long: `Create an inventory adjustment. type=adjust adds or subtracts quantities,
type=set replaces them. Max 200 line items per request.
Pass the line_items and reason via --body.`,
		Category: tools.CatInventory,
		Scopes:   writeInventory,
		Flags: []tools.Flag{
			{Name: "location_id", Type: tools.FlagInt64, Description: "Location/warehouse ID", Required: true},
			{Name: "type", Type: tools.FlagString, Description: "adjust|set (default adjust)",
				Enum: []string{"adjust", "set"}},
			{Name: "reason", Type: tools.FlagString, Description: "Adjustment reason",
				Enum: []string{"newproduct", "returned", "productionofgoods", "damaged", "shrinkage", "promotion", "transfer"}},
			{Name: "note", Type: tools.FlagString, Description: "Adjustment note"},
			{Name: "tags", Type: tools.FlagString, Description: "Comma-separated tags"},
			{Name: "body", Type: tools.FlagJSON, Description: `Line items, e.g. {"line_items":[{"product_id":1,"product_variant_id":2,"quantity":5}]}`, Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			// Compose the full payload from flags + body line_items.
			var lineItems any
			var bodyMap map[string]any
			if raw := in.JSON("body"); len(raw) > 0 {
				tmp := map[string]any{}
				if err := json.Unmarshal(raw, &tmp); err != nil {
					return nil, fmt.Errorf("body: %w", err)
				}
				if li, ok := tmp["line_items"]; ok {
					lineItems = li
				} else {
					lineItems = tmp
				}
				bodyMap = tmp
			}
			payload := map[string]any{
				"location_id": in.Int64("location_id"),
			}
			if t := in.StringOr("type", "adjust"); t != "" {
				payload["type"] = t
			}
			if v := in.String("reason"); v != "" {
				payload["reason"] = v
			}
			if v := in.String("note"); v != "" {
				payload["note"] = v
			}
			if v := in.String("tags"); v != "" {
				payload["tags"] = v
			}
			if lineItems != nil {
				payload["line_items"] = lineItems
			}
			// Preserve any extra keys the user passed via body.
			for k, v := range bodyMap {
				if k == "line_items" {
					continue
				}
				if _, exists := payload[k]; !exists {
					payload[k] = v
				}
			}
			resp, err := deps.Client.Post(ctx, "/com/inventories/adjustorset.json", payload)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerLocations() {
	tools.Register(&tools.Tool{
		Name:     "haravan_inventory_locations",
		Short:    "Get inventory levels by location / variant / product",
		Category: tools.CatInventory,
		Scopes:   readInventory,
		Flags: []tools.Flag{
			{Name: "location_id", Type: tools.FlagInt64, Description: "Filter by location ID"},
			{Name: "variant_id", Type: tools.FlagInt64, Description: "Filter by variant ID"},
			{Name: "product_id", Type: tools.FlagInt64, Description: "Filter by product ID"},
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "location_id", "variant_id", "product_id", "page", "limit")
			resp, err := deps.Client.Get(ctx, "/com/inventories/locations.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}
