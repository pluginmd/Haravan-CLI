package shop

import (
	"context"
	"fmt"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

// readShopScopes is the advisory scope attached to every read-only shop tool.
var readShopScopes = []string{"com.read_shop"}

func init() {
	tools.Register(&tools.Tool{
		Name:     "haravan_shop_get",
		Short:    "Get shop information",
		Long:     "Get shop information: name, domain, email, currency, timezone, plan, address, checkout settings.",
		Category: tools.CatShop,
		Scopes:   readShopScopes,
		Handler: func(ctx context.Context, deps *tools.Deps, _ tools.Input) (*tools.Result, error) {
			resp, err := deps.Client.Get(ctx, "/com/shop.json", nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_locations_list",
		Short:    "List all locations/warehouses",
		Category: tools.CatShop,
		Scopes:   readShopScopes,
		Handler: func(ctx context.Context, deps *tools.Deps, _ tools.Input) (*tools.Result, error) {
			resp, err := deps.Client.Get(ctx, "/com/locations.json", nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_locations_get",
		Short:    "Get a single location by ID",
		Category: tools.CatShop,
		Scopes:   readShopScopes,
		Flags: []tools.Flag{
			{Name: "location_id", Type: tools.FlagInt64, Description: "Location ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/locations/%d.json", in.Int64("location_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_users_list",
		Short:    "List all shop staff users (Haravan Plus only)",
		Category: tools.CatShop,
		Scopes:   readShopScopes,
		Handler: func(ctx context.Context, deps *tools.Deps, _ tools.Input) (*tools.Result, error) {
			resp, err := deps.Client.Get(ctx, "/com/users.json", nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_users_get",
		Short:    "Get a single user by ID (Haravan Plus only)",
		Category: tools.CatShop,
		Scopes:   readShopScopes,
		Flags: []tools.Flag{
			{Name: "user_id", Type: tools.FlagInt64, Description: "User ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/users/%d.json", in.Int64("user_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_shipping_rates_get",
		Short:    "Calculate shipping rates for a destination address",
		Long:     "Get available shipping rates for a destination address. Pass address fields via the --address-* flags; at least one is typically required by the endpoint.",
		Category: tools.CatShop,
		Scopes:   []string{"com.read_orders"},
		Flags: []tools.Flag{
			{Name: "address1", Type: tools.FlagString, Description: "Street address"},
			{Name: "city", Type: tools.FlagString, Description: "City"},
			{Name: "province", Type: tools.FlagString, Description: "Province"},
			{Name: "country", Type: tools.FlagString, Description: "Country"},
			{Name: "zip", Type: tools.FlagString, Description: "ZIP / postal code"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "address1", "city", "province", "country", "zip")
			// Haravan expects bracketed query params: shipping_address[city]=...
			bracketed := make(map[string][]string, len(q))
			for k, v := range q {
				bracketed["shipping_address["+k+"]"] = v
			}
			resp, err := deps.Client.Get(ctx, "/com/shipping_rates.json", bracketed)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}
