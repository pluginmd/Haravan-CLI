// Package mcpserver wraps every registered tools.Tool as an MCP tool and
// serves it over either stdio (default) or HTTP/SSE.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpsrv "github.com/mark3labs/mcp-go/server"

	"github.com/pluginmd/haravan-cli/internal/build"
	"github.com/pluginmd/haravan-cli/internal/client"
	"github.com/pluginmd/haravan-cli/internal/iostreams"
	"github.com/pluginmd/haravan-cli/internal/logger"
	"github.com/pluginmd/haravan-cli/pkg/tools"
)

// Options bundle the runtime context needed to dispatch tool calls.
// The same Client and WebhookClient are shared across all invocations —
// the token is resolved once, when the server starts.
type Options struct {
	Client        *client.Client
	WebhookClient *client.Client
	IO            *iostreams.IOStreams
	ServerName    string // defaults to "haravan-cli"
}

// NewServer builds an MCP server populated from tools.Default().
func NewServer(opts Options) (*mcpsrv.MCPServer, error) {
	if opts.Client == nil {
		return nil, fmt.Errorf("mcpserver: Client is required")
	}
	name := opts.ServerName
	if name == "" {
		name = "haravan-cli"
	}

	srv := mcpsrv.NewMCPServer(
		name,
		build.Version,
		mcpsrv.WithToolCapabilities(true),
		mcpsrv.WithLogging(),
	)

	deps := &tools.Deps{
		Client:        opts.Client,
		WebhookClient: opts.WebhookClient,
		Logger:        logger.Default(),
		IO:            opts.IO,
	}

	for _, t := range tools.Default().All() {
		schema, err := json.Marshal(tools.BuildJSONSchema(t))
		if err != nil {
			return nil, fmt.Errorf("build schema for %s: %w", t.Name, err)
		}
		// Construct the mcp.Tool struct directly rather than via NewTool:
		// NewTool unconditionally initialises InputSchema.Type="object",
		// which then collides with RawInputSchema at marshal time
		// (errToolSchemaConflict). By leaving InputSchema zero we tell
		// mcp-go to marshal only our raw schema.
		mcpTool := mcp.Tool{
			Name:           t.Name,
			Description:    toolDescription(t),
			RawInputSchema: json.RawMessage(schema),
		}
		srv.AddTool(mcpTool, adapt(deps, t))
	}

	logger.Debugf("mcp server registered %d tools", tools.Default().Count())
	return srv, nil
}

// toolDescription produces the text shown to MCP clients. Uses Long if set,
// else Short, and appends the scope hint when present.
func toolDescription(t *tools.Tool) string {
	desc := t.Long
	if desc == "" {
		desc = t.Short
	}
	if len(t.Scopes) > 0 {
		desc += "\n\nRequired scopes: "
		for i, s := range t.Scopes {
			if i > 0 {
				desc += ", "
			}
			desc += s
		}
	}
	return desc
}

// adapt bridges tools.Tool's Handler to mcp-go's ToolHandlerFunc.
// It converts the incoming argument map into tools.Input, runs the Handler,
// and maps the returned Result (or error) into mcp.CallToolResult.
func adapt(deps *tools.Deps, t *tools.Tool) mcpsrv.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		in := tools.Input{}
		for k, v := range req.GetArguments() {
			in[k] = v
		}
		res, err := t.Handler(ctx, deps, in)
		if err != nil {
			msg := client.FriendlyMessage(err)
			if msg == "" {
				msg = err.Error()
			}
			return mcp.NewToolResultError(msg), nil
		}
		if res == nil {
			return mcp.NewToolResultText(""), nil
		}
		if len(res.Data) > 0 {
			return mcp.NewToolResultText(string(res.Data)), nil
		}
		if res.IsError && res.Text != "" {
			return mcp.NewToolResultError(res.Text), nil
		}
		return mcp.NewToolResultText(res.Text), nil
	}
}
