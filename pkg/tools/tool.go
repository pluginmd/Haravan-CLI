// Package tools defines the Tool abstraction used by both the CLI and MCP
// surface areas. Every Haravan endpoint is modeled as one Tool whose
// Handler is invoked by a cobra command or an MCP tool call.
package tools

import (
	"context"

	"github.com/pluginmd/haravan-cli/internal/client"
	"github.com/pluginmd/haravan-cli/internal/iostreams"
	"github.com/pluginmd/haravan-cli/internal/logger"
)

// Category groups related tools (matches Haravan API domains).
type Category string

const (
	CatOrders    Category = "orders"
	CatProducts  Category = "products"
	CatCustomers Category = "customers"
	CatInventory Category = "inventory"
	CatShop      Category = "shop"
	CatContent   Category = "content"
	CatWebhooks  Category = "webhooks"
	CatSmart     Category = "smart"
)

// Tool is the single source of truth for one Haravan operation.
// A Tool is registered via Registry.Register() and surfaces twice:
//  1. As a cobra subcommand under the category group (CLI use)
//  2. As an MCP tool advertised over stdio/SSE (AI-assistant use)
type Tool struct {
	// Name is the public identifier, e.g. "haravan_orders_list".
	// Use dot-free snake_case so MCP clients can reference it verbatim.
	Name string

	// Short is the one-line help text shown in `haravan-cli <cat> --help`.
	Short string

	// Long is a longer description; defaults to Short if empty.
	Long string

	// Category groups the tool in the CLI tree and MCP metadata.
	Category Category

	// Scopes lists the OAuth scopes the tool requires. Advisory only —
	// enforcement is done by the Haravan API on the server side.
	Scopes []string

	// Flags declares the tool's inputs. Each Flag produces both a cobra
	// flag and a JSON Schema property. See flag.go for the Flag type.
	Flags []Flag

	// Handler is the business logic. Deps are pre-built by the binder
	// (auth-resolved Client, logger, etc.). Input is a decoded map keyed
	// by Flag.Name — callers use InputAs* helpers to type-assert safely.
	Handler func(ctx context.Context, deps *Deps, input Input) (*Result, error)
}

// Deps holds the resources a Handler typically needs. Additional fields
// can be introduced without breaking existing Handlers.
type Deps struct {
	Client *client.Client
	Logger *logger.Logger
	IO     *iostreams.IOStreams
}
