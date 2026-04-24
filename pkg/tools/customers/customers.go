package customers

import (
	"context"
	"fmt"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var (
	readCustomers  = []string{"com.read_customers"}
	writeCustomers = []string{"com.write_customers"}
)

func init() {
	registerList()
	registerSearch()
	registerCount()
	registerGet()
	registerCreate()
	registerUpdate()
	registerDelete()
	registerGroups()
	registerAddresses()
}

func registerList() {
	tools.Register(&tools.Tool{
		Name:     "haravan_customers_list",
		Short:    "List customers",
		Long:     "List customers with pagination and date filters.",
		Category: tools.CatCustomers,
		Scopes:   readCustomers,
		Flags: []tools.Flag{
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page (max 250)"},
			{Name: "since_id", Type: tools.FlagInt64, Description: "Results after this ID"},
			{Name: "created_at_min", Type: tools.FlagDateTimeISO, Description: "Created after (ISO 8601)"},
			{Name: "created_at_max", Type: tools.FlagDateTimeISO, Description: "Created before (ISO 8601)"},
			{Name: "updated_at_min", Type: tools.FlagDateTimeISO, Description: "Updated after (ISO 8601)"},
			{Name: "updated_at_max", Type: tools.FlagDateTimeISO, Description: "Updated before (ISO 8601)"},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
			{Name: "fetch_all", Type: tools.FlagBool, Description: "Auto-paginate to fetch every page"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "page", "limit", "since_id", "fields",
				"created_at_min", "created_at_max", "updated_at_min", "updated_at_max")
			if in.Bool("fetch_all") {
				return tools.Paginate(ctx, deps, "/com/customers.json", q, "customers")
			}
			resp, err := deps.Client.Get(ctx, "/com/customers.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerSearch() {
	tools.Register(&tools.Tool{
		Name:     "haravan_customers_search",
		Short:    "Search customers (email, phone, name, …)",
		Category: tools.CatCustomers,
		Scopes:   readCustomers,
		Flags: []tools.Flag{
			{Name: "query", Type: tools.FlagString, Description: "Search query", Required: true},
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "query", "page", "limit", "fields")
			resp, err := deps.Client.Get(ctx, "/com/customers/search.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerCount() {
	tools.Register(&tools.Tool{
		Name:     "haravan_customers_count",
		Short:    "Count customers with date filters",
		Category: tools.CatCustomers,
		Scopes:   readCustomers,
		Flags: []tools.Flag{
			{Name: "created_at_min", Type: tools.FlagDateTimeISO, Description: "Created after (ISO 8601)"},
			{Name: "created_at_max", Type: tools.FlagDateTimeISO, Description: "Created before (ISO 8601)"},
			{Name: "updated_at_min", Type: tools.FlagDateTimeISO, Description: "Updated after (ISO 8601)"},
			{Name: "updated_at_max", Type: tools.FlagDateTimeISO, Description: "Updated before (ISO 8601)"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "created_at_min", "created_at_max", "updated_at_min", "updated_at_max")
			resp, err := deps.Client.Get(ctx, "/com/customers/count.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerGet() {
	tools.Register(&tools.Tool{
		Name:     "haravan_customers_get",
		Short:    "Get a single customer by ID",
		Category: tools.CatCustomers,
		Scopes:   readCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/customers/%d.json", in.Int64("customer_id"))
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
		Name:     "haravan_customers_create",
		Short:    "Create a new customer",
		Long:     `Create a customer. Email OR phone is required. Pass the payload via --body (e.g. {"customer":{"email":"x@y.z"}}).`,
		Category: tools.CatCustomers,
		Scopes:   writeCustomers,
		Flags: []tools.Flag{
			{Name: "body", Type: tools.FlagJSON, Description: "Customer payload (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "customer")
			if err != nil {
				return nil, err
			}
			resp, err := deps.Client.Post(ctx, "/com/customers.json", body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerUpdate() {
	tools.Register(&tools.Tool{
		Name:     "haravan_customers_update",
		Short:    "Update an existing customer",
		Category: tools.CatCustomers,
		Scopes:   writeCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: "Customer patch (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "customer")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/customers/%d.json", in.Int64("customer_id"))
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
		Name:     "haravan_customers_delete",
		Short:    "Delete a customer by ID",
		Category: tools.CatCustomers,
		Scopes:   writeCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/customers/%d.json", in.Int64("customer_id"))
			resp, err := deps.Client.Delete(ctx, path)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerGroups() {
	tools.Register(&tools.Tool{
		Name:     "haravan_customers_groups",
		Short:    "List all customer groups",
		Category: tools.CatCustomers,
		Scopes:   readCustomers,
		Handler: func(ctx context.Context, deps *tools.Deps, _ tools.Input) (*tools.Result, error) {
			resp, err := deps.Client.Get(ctx, "/com/customers/groups.json", nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerAddresses() {
	tools.Register(&tools.Tool{
		Name:     "haravan_customer_addresses_list",
		Short:    "List addresses of a customer",
		Category: tools.CatCustomers,
		Scopes:   readCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "page", Type: tools.FlagInt, Description: "Page number"},
			{Name: "limit", Type: tools.FlagInt, Description: "Results per page"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/customers/%d/addresses.json", in.Int64("customer_id"))
			resp, err := deps.Client.Get(ctx, path, tools.Query(in, "page", "limit"))
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_customer_addresses_get",
		Short:    "Get a specific address of a customer",
		Category: tools.CatCustomers,
		Scopes:   readCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "address_id", Type: tools.FlagInt64, Description: "Address ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/customers/%d/addresses/%d.json", in.Int64("customer_id"), in.Int64("address_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_customer_addresses_create",
		Short:    "Create a new address for a customer",
		Category: tools.CatCustomers,
		Scopes:   writeCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: "Address payload (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "address")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/customers/%d/addresses.json", in.Int64("customer_id"))
			resp, err := deps.Client.Post(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_customer_addresses_update",
		Short:    "Update a customer address",
		Category: tools.CatCustomers,
		Scopes:   writeCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "address_id", Type: tools.FlagInt64, Description: "Address ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: "Address patch (wrapped or bare)", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := tools.EnvelopeWrap(in.JSON("body"), "address")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/customers/%d/addresses/%d.json", in.Int64("customer_id"), in.Int64("address_id"))
			resp, err := deps.Client.Put(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_customer_addresses_delete",
		Short:    "Delete a customer address",
		Category: tools.CatCustomers,
		Scopes:   writeCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "address_id", Type: tools.FlagInt64, Description: "Address ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/customers/%d/addresses/%d.json", in.Int64("customer_id"), in.Int64("address_id"))
			resp, err := deps.Client.Delete(ctx, path)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_customer_addresses_set_default",
		Short:    "Set a customer address as default",
		Category: tools.CatCustomers,
		Scopes:   writeCustomers,
		Flags: []tools.Flag{
			{Name: "customer_id", Type: tools.FlagInt64, Description: "Customer ID", Required: true},
			{Name: "address_id", Type: tools.FlagInt64, Description: "Address ID to set as default", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/customers/%d/addresses/%d/default.json", in.Int64("customer_id"), in.Int64("address_id"))
			resp, err := deps.Client.Put(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}
