package products

import (
	"context"
	"fmt"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var (
	readProducts  = []string{"com.read_products"}
	writeProducts = []string{"com.write_products"}
)

// listFilterFlags covers the filter surface shared by products_list and products_count.
func listFilterFlags() []tools.Flag {
	return []tools.Flag{
		{Name: "collection_id", Type: tools.FlagInt64, Description: "Filter by collection"},
		{Name: "product_type", Type: tools.FlagString, Description: "Filter by product type"},
		{Name: "vendor", Type: tools.FlagString, Description: "Filter by vendor"},
		{Name: "handle", Type: tools.FlagString, Description: "Filter by handle (URL slug)"},
		{Name: "created_at_min", Type: tools.FlagDateTimeISO, Description: "Created after (ISO 8601)"},
		{Name: "created_at_max", Type: tools.FlagDateTimeISO, Description: "Created before (ISO 8601)"},
		{Name: "updated_at_min", Type: tools.FlagDateTimeISO, Description: "Updated after (ISO 8601)"},
		{Name: "updated_at_max", Type: tools.FlagDateTimeISO, Description: "Updated before (ISO 8601)"},
		{Name: "published_at_min", Type: tools.FlagDateTimeISO, Description: "Published after (ISO 8601)"},
		{Name: "published_at_max", Type: tools.FlagDateTimeISO, Description: "Published before (ISO 8601)"},
		{Name: "published_status", Type: tools.FlagString, Description: "Published status",
			Enum: []string{"published", "unpublished", "any"}},
	}
}

func init() {
	registerList()
	registerCount()
	registerGet()
	registerCreate()
	registerUpdate()
	registerDelete()
	registerVariants()
}

func registerList() {
	flags := append(listFilterFlags(),
		tools.Flag{Name: "page", Type: tools.FlagInt, Description: "Page number"},
		tools.Flag{Name: "limit", Type: tools.FlagInt, Description: "Results per page (max 250)"},
		tools.Flag{Name: "since_id", Type: tools.FlagInt64, Description: "Results after this ID"},
		tools.Flag{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		tools.Flag{Name: "fetch_all", Type: tools.FlagBool, Description: "Auto-paginate until all results are returned"},
	)
	tools.Register(&tools.Tool{
		Name:     "haravan_products_list",
		Short:    "List products",
		Long:     "List all products with pagination and filtering (collection, type, vendor, handle, publish state, …).",
		Category: tools.CatProducts,
		Scopes:   readProducts,
		Flags:    flags,
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "page", "limit", "since_id", "fields",
				"collection_id", "product_type", "vendor", "handle",
				"created_at_min", "created_at_max", "updated_at_min", "updated_at_max",
				"published_at_min", "published_at_max", "published_status")
			if in.Bool("fetch_all") {
				return tools.Paginate(ctx, deps, "/com/products.json", q, "products")
			}
			resp, err := deps.Client.Get(ctx, "/com/products.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerCount() {
	tools.Register(&tools.Tool{
		Name:     "haravan_products_count",
		Short:    "Get total product count with optional filters",
		Category: tools.CatProducts,
		Scopes:   readProducts,
		Flags:    listFilterFlags(),
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "collection_id", "product_type", "vendor",
				"created_at_min", "created_at_max", "updated_at_min", "updated_at_max",
				"published_at_min", "published_at_max", "published_status")
			resp, err := deps.Client.Get(ctx, "/com/products/count.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerGet() {
	tools.Register(&tools.Tool{
		Name:     "haravan_products_get",
		Short:    "Get a single product by ID",
		Long:     "Get a product by ID. Returns full details including variants, images, and options.",
		Category: tools.CatProducts,
		Scopes:   readProducts,
		Flags: []tools.Flag{
			{Name: "product_id", Type: tools.FlagInt64, Description: "Product ID", Required: true},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/products/%d.json", in.Int64("product_id"))
			resp, err := deps.Client.Get(ctx, path, tools.Query(in, "fields"))
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerCreate() {
	tools.Register(&tools.Tool{
		Name:     "haravan_products_create",
		Short:    "Create a new product",
		Long:     `Create a product. Pass the full payload via --body (e.g. {"product":{"title":"T","variants":[...]}}). Title is required.`,
		Category: tools.CatProducts,
		Scopes:   writeProducts,
		Flags: []tools.Flag{
			{Name: "body", Type: tools.FlagJSON, Description: "Haravan product payload (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "product")
			if err != nil {
				return nil, err
			}
			resp, err := deps.Client.Post(ctx, "/com/products.json", body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerUpdate() {
	tools.Register(&tools.Tool{
		Name:     "haravan_products_update",
		Short:    "Update an existing product",
		Category: tools.CatProducts,
		Scopes:   writeProducts,
		Flags: []tools.Flag{
			{Name: "product_id", Type: tools.FlagInt64, Description: "Product ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: "Haravan product patch (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "product")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/products/%d.json", in.Int64("product_id"))
			resp, err := deps.Client.Put(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerDelete() {
	tools.Register(&tools.Tool{
		Name:     "haravan_products_delete",
		Short:    "Delete a product by ID",
		Category: tools.CatProducts,
		Scopes:   writeProducts,
		Flags: []tools.Flag{
			{Name: "product_id", Type: tools.FlagInt64, Description: "Product ID to delete", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/products/%d.json", in.Int64("product_id"))
			resp, err := deps.Client.Delete(ctx, path)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerVariants() {
	tools.Register(&tools.Tool{
		Name:     "haravan_variants_list",
		Short:    "List variants of a product",
		Category: tools.CatProducts,
		Scopes:   readProducts,
		Flags: []tools.Flag{
			{Name: "product_id", Type: tools.FlagInt64, Description: "Product ID", Required: true},
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/products/%d/variants.json", in.Int64("product_id"))
			resp, err := deps.Client.Get(ctx, path, tools.Query(in, "page", "limit", "fields"))
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_variants_count",
		Short:    "Count variants of a product",
		Category: tools.CatProducts,
		Scopes:   readProducts,
		Flags: []tools.Flag{
			{Name: "product_id", Type: tools.FlagInt64, Description: "Product ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/products/%d/variants/count.json", in.Int64("product_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_variants_get",
		Short:    "Get a single variant by ID",
		Long:     "Fetches /com/variants/{variant_id}.json — no product_id required.",
		Category: tools.CatProducts,
		Scopes:   readProducts,
		Flags: []tools.Flag{
			{Name: "variant_id", Type: tools.FlagInt64, Description: "Variant ID", Required: true},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/variants/%d.json", in.Int64("variant_id"))
			resp, err := deps.Client.Get(ctx, path, tools.Query(in, "fields"))
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_variants_create",
		Short:    "Create a variant for a product",
		Category: tools.CatProducts,
		Scopes:   writeProducts,
		Flags: []tools.Flag{
			{Name: "product_id", Type: tools.FlagInt64, Description: "Product ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: `Variant payload, e.g. {"variant":{"price":99000,"sku":"X"}}`, Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "variant")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/products/%d/variants.json", in.Int64("product_id"))
			resp, err := deps.Client.Post(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_variants_update",
		Short:    "Update a variant",
		Long:     "Updates /com/variants/{variant_id}.json. The legacy TS API requires only the variant_id, not the parent product.",
		Category: tools.CatProducts,
		Scopes:   writeProducts,
		Flags: []tools.Flag{
			{Name: "variant_id", Type: tools.FlagInt64, Description: "Variant ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: "Variant patch (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "variant")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/variants/%d.json", in.Int64("variant_id"))
			resp, err := deps.Client.Put(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}
