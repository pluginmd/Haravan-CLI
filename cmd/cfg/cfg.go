// Package cfg provides `haravan-cli config` subcommands.
//
// The package is named cfg (not config) to avoid colliding with the
// internal/config import in files that consume both.
package cfg

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pluginmd/haravan-cli/internal/cmdutil"
	"github.com/pluginmd/haravan-cli/internal/config"
)

// NewCmd returns the `config` command group.
func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage persistent CLI configuration",
	}
	cmd.AddCommand(newShowCmd(f))
	cmd.AddCommand(newSetCmd(f))
	cmd.AddCommand(newPathCmd(f))
	return cmd
}

func newShowCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			view := map[string]any{
				"api_base":     cfg.ResolveAPIBase(),
				"app_id":       cfg.AppID,
				"app_secret":   maskSecret(cfg.AppSecret),
				"default_auth": cfg.DefaultAuth,
			}
			if jsonOut {
				enc := json.NewEncoder(f.IOStreams.Out)
				enc.SetIndent("", "  ")
				return enc.Encode(view)
			}
			for _, k := range []string{"api_base", "app_id", "app_secret", "default_auth"} {
				fmt.Fprintf(f.IOStreams.Out, "%-13s %v\n", k+":", view[k])
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	return cmd
}

func newSetCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		apiBase     string
		appID       string
		appSecret   string
		defaultAuth string
	)
	cmd := &cobra.Command{
		Use:   "set",
		Short: "Update configuration fields",
		Long: `Update one or more configuration fields. Only flags provided are touched;
unset flags leave existing values in place.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			touched := false
			if cmd.Flags().Changed("api-base") {
				cfg.APIBase = apiBase
				touched = true
			}
			if cmd.Flags().Changed("app-id") {
				cfg.AppID = appID
				touched = true
			}
			if cmd.Flags().Changed("app-secret") {
				cfg.AppSecret = appSecret
				touched = true
			}
			if cmd.Flags().Changed("default-auth") {
				switch defaultAuth {
				case "oauth", "token", "":
					cfg.DefaultAuth = defaultAuth
				default:
					return errors.New("default-auth must be 'oauth' or 'token'")
				}
				touched = true
			}
			if !touched {
				return errors.New("no fields provided (see --help)")
			}
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Fprintln(f.IOStreams.Out, "configuration saved")
			return nil
		},
	}
	cmd.Flags().StringVar(&apiBase, "api-base", "", "Haravan API base URL (default https://apis.haravan.com)")
	cmd.Flags().StringVar(&appID, "app-id", "", "default Haravan app client_id")
	cmd.Flags().StringVar(&appSecret, "app-secret", "", "default Haravan app client_secret")
	cmd.Flags().StringVar(&defaultAuth, "default-auth", "", "default auth mode: oauth|token")
	return cmd
}

func newPathCmd(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			p, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(f.IOStreams.Out, p)
			return nil
		},
	}
}

func maskSecret(s string) string {
	if s == "" {
		return ""
	}
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "***" + s[len(s)-3:]
}
