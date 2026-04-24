package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/pluginmd/haravan-cli/pkg/tools"
)

var (
	readOrders  = []string{"com.read_orders"}
	writeOrders = []string{"com.write_orders"}
)

// Common list filters shared by orders_list and orders_count.
func listFilterFlags() []tools.Flag {
	return []tools.Flag{
		{Name: "status", Type: tools.FlagString, Description: "Order status filter",
			Enum: []string{"open", "closed", "cancelled", "any"}},
		{Name: "financial_status", Type: tools.FlagString, Description: "Financial status filter",
			Enum: []string{"pending", "authorized", "partially_paid", "paid", "partially_refunded", "refunded", "voided", "any"}},
		{Name: "fulfillment_status", Type: tools.FlagString, Description: "Fulfillment status filter",
			Enum: []string{"fulfilled", "partial", "unshipped", "any"}},
		{Name: "created_at_min", Type: tools.FlagDateTimeISO, Description: "Created after (ISO 8601)"},
		{Name: "created_at_max", Type: tools.FlagDateTimeISO, Description: "Created before (ISO 8601)"},
		{Name: "updated_at_min", Type: tools.FlagDateTimeISO, Description: "Updated after (ISO 8601)"},
		{Name: "updated_at_max", Type: tools.FlagDateTimeISO, Description: "Updated before (ISO 8601)"},
	}
}

func init() {
	registerList()
	registerCount()
	registerGet()
	registerCreate()
	registerUpdate()
	registerStatusTransitions()
	registerCancel()
	registerAssign()
	registerTransactions()
}

func registerList() {
	flags := append(listFilterFlags(),
		tools.Flag{Name: "page", Type: tools.FlagInt, Description: "Page number (default 1)"},
		tools.Flag{Name: "limit", Type: tools.FlagInt, Description: "Results per page (default 50, max 250)"},
		tools.Flag{Name: "since_id", Type: tools.FlagInt64, Description: "Results after this ID"},
		tools.Flag{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields to include"},
		tools.Flag{Name: "fetch_all", Type: tools.FlagBool, Description: "Auto-paginate until all records are returned"},
	)
	tools.Register(&tools.Tool{
		Name:     "haravan_orders_list",
		Short:    "List orders",
		Long:     "List orders. Filter by status, financial_status, fulfillment_status, created_at, updated_at. Supports pagination.",
		Category: tools.CatOrders,
		Scopes:   readOrders,
		Flags:    flags,
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in, "page", "limit", "since_id", "fields",
				"status", "financial_status", "fulfillment_status",
				"created_at_min", "created_at_max", "updated_at_min", "updated_at_max")
			if in.Bool("fetch_all") {
				return tools.Paginate(ctx, deps, "/com/orders.json", q, "orders")
			}
			resp, err := deps.Client.Get(ctx, "/com/orders.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerCount() {
	tools.Register(&tools.Tool{
		Name:     "haravan_orders_count",
		Short:    "Get total order count with optional filters",
		Category: tools.CatOrders,
		Scopes:   readOrders,
		Flags:    listFilterFlags(),
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			q := tools.Query(in,
				"status", "financial_status", "fulfillment_status",
				"created_at_min", "created_at_max", "updated_at_min", "updated_at_max")
			resp, err := deps.Client.Get(ctx, "/com/orders/count.json", q)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerGet() {
	tools.Register(&tools.Tool{
		Name:     "haravan_orders_get",
		Short:    "Get a single order by ID",
		Long:     "Get a single order by ID. Returns full order details including line_items, shipping, billing, transactions.",
		Category: tools.CatOrders,
		Scopes:   readOrders,
		Flags: []tools.Flag{
			{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID", Required: true},
			{Name: "fields", Type: tools.FlagString, Description: "Comma-separated fields"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/orders/%d.json", in.Int64("order_id"))
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
		Name:     "haravan_orders_create",
		Short:    "Create a new order",
		Long: `Create a new order. Pass the Haravan order payload via --body as inline JSON or @file.
The body MUST be wrapped as {"order": {...}}: line_items is required, billing/shipping
addresses, tags, discount_codes, note, source_name are all supported.`,
		Category: tools.CatOrders,
		Scopes:   writeOrders,
		Flags: []tools.Flag{
			{Name: "body", Type: tools.FlagJSON, Description: `Haravan order payload, e.g. {"order":{"line_items":[...]}}`, Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := unwrapOrWrap(in.JSON("body"), "order")
			if err != nil {
				return nil, err
			}
			resp, err := deps.Client.Post(ctx, "/com/orders.json", body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerUpdate() {
	tools.Register(&tools.Tool{
		Name:     "haravan_orders_update",
		Short:    "Update an existing order",
		Long:     "Update an existing order (note, tags, shipping_address, email, ...). Pass the payload via --body.",
		Category: tools.CatOrders,
		Scopes:   writeOrders,
		Flags: []tools.Flag{
			{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: `Haravan order patch, e.g. {"order":{"note":"..."}}`, Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := unwrapOrWrap(in.JSON("body"), "order")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/orders/%d.json", in.Int64("order_id"))
			resp, err := deps.Client.Put(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

// registerStatusTransitions covers confirm / close / open — all identical
// POST {order_id}/<action>.json with no body.
func registerStatusTransitions() {
	for _, spec := range []struct {
		Name, Action, Short string
	}{
		{"haravan_orders_confirm", "confirm", "Confirm an order"},
		{"haravan_orders_close", "close", "Close an order"},
		{"haravan_orders_open", "open", "Reopen a closed order"},
	} {
		spec := spec
		tools.Register(&tools.Tool{
			Name:     spec.Name,
			Short:    spec.Short,
			Category: tools.CatOrders,
			Scopes:   writeOrders,
			Flags: []tools.Flag{
				{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID", Required: true},
			},
			Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
				path := fmt.Sprintf("/com/orders/%d/%s.json", in.Int64("order_id"), spec.Action)
				resp, err := deps.Client.Post(ctx, path, nil)
				if err != nil {
					return nil, err
				}
				return tools.NewRaw(resp.Body), nil
			},
		})
	}
}

func registerCancel() {
	tools.Register(&tools.Tool{
		Name:     "haravan_orders_cancel",
		Short:    "Cancel an order",
		Long:     "Cancel an order. Reason must be one of: customer, fraud, inventory, declined, other.",
		Category: tools.CatOrders,
		Scopes:   writeOrders,
		Flags: []tools.Flag{
			{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID to cancel", Required: true},
			{Name: "reason", Type: tools.FlagString, Description: "Cancellation reason",
				Enum: []string{"customer", "fraud", "inventory", "declined", "other"}},
			{Name: "email", Type: tools.FlagBool, Description: "Send cancellation email"},
			{Name: "restock", Type: tools.FlagBool, Description: "Restock cancelled items"},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/orders/%d/cancel.json", in.Int64("order_id"))
			body := map[string]any{}
			if v := in.String("reason"); v != "" {
				body["reason"] = v
			}
			if in.Has("email") {
				body["email"] = in.Bool("email")
			}
			if in.Has("restock") {
				body["restock"] = in.Bool("restock")
			}
			var payload any
			if len(body) > 0 {
				payload = body
			}
			resp, err := deps.Client.Post(ctx, path, payload)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerAssign() {
	tools.Register(&tools.Tool{
		Name:     "haravan_orders_assign",
		Short:    "Assign staff to an order",
		Category: tools.CatOrders,
		Scopes:   writeOrders,
		Flags: []tools.Flag{
			{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID", Required: true},
			{Name: "user_id", Type: tools.FlagInt64, Description: "Staff user ID to assign", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/orders/%d/assign.json", in.Int64("order_id"))
			body := map[string]any{"user_id": in.Int64("user_id")}
			resp, err := deps.Client.Post(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

func registerTransactions() {
	tools.Register(&tools.Tool{
		Name:     "haravan_transactions_list",
		Short:    "List all transactions for an order",
		Category: tools.CatOrders,
		Scopes:   readOrders,
		Flags: []tools.Flag{
			{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/orders/%d/transactions.json", in.Int64("order_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_transactions_get",
		Short:    "Get a specific transaction",
		Category: tools.CatOrders,
		Scopes:   readOrders,
		Flags: []tools.Flag{
			{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID", Required: true},
			{Name: "transaction_id", Type: tools.FlagInt64, Description: "Transaction ID", Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			path := fmt.Sprintf("/com/orders/%d/transactions/%d.json", in.Int64("order_id"), in.Int64("transaction_id"))
			resp, err := deps.Client.Get(ctx, path, nil)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})

	tools.Register(&tools.Tool{
		Name:     "haravan_transactions_create",
		Short:    "Create a transaction for an order",
		Long:     "Create a transaction. Kind: Pending | Authorization | Sale | Capture | Void | Refund. Pass via --body.",
		Category: tools.CatOrders,
		Scopes:   writeOrders,
		Flags: []tools.Flag{
			{Name: "order_id", Type: tools.FlagInt64, Description: "Order ID", Required: true},
			{Name: "body", Type: tools.FlagJSON, Description: `Transaction payload, e.g. {"transaction":{"amount":100,"kind":"Sale"}}`, Required: true},
		},
		Handler: func(ctx context.Context, deps *tools.Deps, in tools.Input) (*tools.Result, error) {
			body, err := unwrapOrWrap(in.JSON("body"), "transaction")
			if err != nil {
				return nil, err
			}
			path := fmt.Sprintf("/com/orders/%d/transactions.json", in.Int64("order_id"))
			resp, err := deps.Client.Post(ctx, path, body)
			if err != nil {
				return nil, err
			}
			return tools.NewRaw(resp.Body), nil
		},
	})
}

// unwrapOrWrap accepts either a bare resource object or the already-wrapped
// envelope, and returns the wrapped form the Haravan API expects.
//
//	input {"line_items":[...]}        wrapper "order" -> {"order":{...}}
//	input {"order":{"line_items":[]}} wrapper "order" -> unchanged
func unwrapOrWrap(raw json.RawMessage, wrapper string) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty body")
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("body must be a JSON object: %w", err)
	}
	if _, ok := probe[wrapper]; ok {
		return raw, nil
	}
	wrapped, err := json.Marshal(map[string]json.RawMessage{wrapper: raw})
	if err != nil {
		return nil, err
	}
	return wrapped, nil
}

