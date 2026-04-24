// Package mcp provides `haravan-cli mcp` subcommands (currently: serve).
package mcp

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/pluginmd/haravan-cli/internal/cmdutil"
	"github.com/pluginmd/haravan-cli/internal/logger"
	"github.com/pluginmd/haravan-cli/internal/mcpserver"
	"github.com/pluginmd/haravan-cli/pkg/tools"
)

// NewCmd returns the `mcp` command group.
func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Model Context Protocol server (stdio or HTTP/SSE)",
	}
	cmd.AddCommand(newServeCmd(f))
	cmd.AddCommand(newListCmd(f))
	return cmd
}

func newServeCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		useHTTP bool
		addr    string
		baseURL string
	)
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server (default transport: stdio)",
		Long: `Start the Haravan MCP server, exposing every registered tool to any
MCP-compatible client (Claude.ai, Claude Code, Cursor, etc.).

Default transport is stdio. Pass --http to run HTTP/SSE on --addr (default :4567).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			c, err := cmdutil.BuildClient(ctx, cmd, cfg)
			if err != nil {
				return fmt.Errorf("build api client: %w", err)
			}
			wc, err := cmdutil.BuildWebhookClient(ctx, cmd, cfg)
			if err != nil {
				logger.Debugf("webhook client unavailable: %v", err)
			}

			srv, err := mcpserver.NewServer(mcpserver.Options{
				Client:        c,
				WebhookClient: wc,
				IO:            f.IOStreams,
			})
			if err != nil {
				return err
			}
			logger.Infof("mcp server starting (tools=%d transport=%s)", tools.Default().Count(), transportLabel(useHTTP, addr))
			return mcpserver.Serve(ctx, srv, mcpserver.TransportConfig{
				HTTP:    useHTTP,
				Addr:    addr,
				BaseURL: baseURL,
			})
		},
	}
	cmd.Flags().BoolVar(&useHTTP, "http", false, "serve HTTP/SSE instead of stdio")
	cmd.Flags().StringVar(&addr, "addr", ":4567", "HTTP listen address (only used with --http)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "public base URL advertised to SSE clients (optional)")
	return cmd
}

func newListCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "List every tool the MCP server advertises",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := tools.Default()
			all := reg.All()
			if jsonOut {
				type entry struct {
					Name     string   `json:"name"`
					Category string   `json:"category"`
					Short    string   `json:"short"`
					Scopes   []string `json:"scopes,omitempty"`
				}
				out := make([]entry, 0, len(all))
				for _, t := range all {
					out = append(out, entry{t.Name, string(t.Category), t.Short, t.Scopes})
				}
				return writeJSON(f, out)
			}
			for _, cat := range reg.Categories() {
				fmt.Fprintf(f.IOStreams.Out, "# %s\n", cat)
				for _, t := range reg.ByCategory(cat) {
					fmt.Fprintf(f.IOStreams.Out, "  %-42s %s\n", t.Name, t.Short)
				}
			}
			fmt.Fprintf(f.IOStreams.Out, "\ntotal: %d tools\n", reg.Count())
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	return cmd
}

func transportLabel(useHTTP bool, addr string) string {
	if useHTTP {
		return "http " + addr
	}
	return "stdio"
}

func writeJSON(f *cmdutil.Factory, v any) error {
	return cmdutil.WriteJSON(f.IOStreams.Out, v)
}
