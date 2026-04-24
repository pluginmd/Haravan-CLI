// Package cmd wires the top-level Cobra command tree for haravan-cli.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/pluginmd/haravan-cli/internal/build"
	"github.com/pluginmd/haravan-cli/internal/cmdutil"
	"github.com/pluginmd/haravan-cli/internal/config"
	"github.com/pluginmd/haravan-cli/internal/logger"
)

// Execute builds the root command and runs it with os.Args.
func Execute() error {
	f := cmdutil.New()
	root := NewRootCmd(f)
	return root.Execute()
}

// NewRootCmd returns the configured root command. Exposed for tests.
func NewRootCmd(f *cmdutil.Factory) *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:   "haravan-cli",
		Short: "haravan-cli — Haravan e-commerce CLI and MCP server",
		Long: `haravan-cli is a unified CLI and MCP server for the Haravan API.

It wraps 70 Haravan endpoints (products, orders, customers, inventory,
shop, content, webhooks) and exposes them as both shell commands and
Model Context Protocol tools for AI assistants.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			lvl := logLevel
			if lvl == "" {
				lvl = os.Getenv(config.EnvLogLevel)
			}
			if lvl != "" {
				logger.SetLevel(logger.ParseLevel(lvl))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "", "log level: debug|info|warn|error|off")

	root.AddCommand(NewVersionCmd(f))

	return root
}

// NewVersionCmd prints build metadata.
func NewVersionCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintf(f.IOStreams.Out, "haravan-cli %s (commit %s, built %s)\n",
				build.Version, build.Commit, build.Date)
			return nil
		},
	}
}
