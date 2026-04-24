// Package auth provides `haravan-cli auth` subcommands.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/pluginmd/haravan-cli/internal/auth"
	"github.com/pluginmd/haravan-cli/internal/cmdutil"
	"github.com/pluginmd/haravan-cli/internal/config"
)

var defaultScopes = []string{
	"com_insights",
	"com_customer",
	"com_product",
	"com_order",
	"com_content",
	"com_setting",
}

// NewCmd returns the `auth` command group.
func NewCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage Haravan credentials",
		Long:  "Login, logout, and inspect the stored OAuth credential for a Haravan app.",
	}
	cmd.AddCommand(newLoginCmd(f))
	cmd.AddCommand(newLogoutCmd(f))
	cmd.AddCommand(newStatusCmd(f))
	return cmd
}

func newLoginCmd(f *cmdutil.Factory) *cobra.Command {
	var (
		appID     string
		appSecret string
		scopesCSV string
		port      int
		save      bool
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "OAuth login to a Haravan app",
		Long: `Run the Haravan OAuth 2.0 authorization-code flow.

Starts a local callback server, opens the browser to the consent page,
then exchanges the returned code for an access token and stores it at
~/.haravan-cli/tokens.json.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			if appID == "" {
				appID = firstNonEmpty(os.Getenv(config.EnvAppID), cfg.AppID)
			}
			if appSecret == "" {
				appSecret = firstNonEmpty(os.Getenv(config.EnvAppSecret), cfg.AppSecret)
			}
			if appID == "" || appSecret == "" {
				return errors.New("app_id and app_secret are required (flags, env HARAVAN_APP_ID/HARAVAN_APP_SECRET, or `haravan-cli config set`)")
			}

			scopes := defaultScopes
			if scopesCSV != "" {
				scopes = splitScopes(scopesCSV)
			}

			store, err := auth.NewTokenStore()
			if err != nil {
				return err
			}

			ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			fmt.Fprintf(f.IOStreams.ErrOut, "→ requesting scopes: %s\n", strings.Join(scopes, " "))
			tok, err := auth.PerformLogin(ctx, auth.OAuthConfig{
				AppID:     appID,
				AppSecret: appSecret,
				Scopes:    scopes,
				Port:      port,
				Timeout:   5 * time.Minute,
			}, auth.LoginFeedback{
				OnAuthURL: func(u string) {
					fmt.Fprintf(f.IOStreams.ErrOut, "→ open this URL if the browser didn't launch:\n  %s\n", u)
				},
				OnBrowserOK: func() {
					fmt.Fprintln(f.IOStreams.ErrOut, "→ browser opened, waiting for callback…")
				},
				OnSuccess: func() {
					fmt.Fprintln(f.IOStreams.ErrOut, "✓ login successful")
				},
			})
			if err != nil {
				return err
			}

			if err := store.Save(appID, tok); err != nil {
				return fmt.Errorf("persist token: %w", err)
			}
			fmt.Fprintf(f.IOStreams.Out, "token stored at %s for app %s\n", store.Path(), appID)

			if save {
				cfg.AppID = appID
				cfg.AppSecret = appSecret
				cfg.DefaultAuth = "oauth"
				if err := config.Save(cfg); err != nil {
					return fmt.Errorf("save app credentials to config: %w", err)
				}
				fmt.Fprintln(f.IOStreams.Out, "app credentials saved to config")
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&appID, "app-id", "a", "", "Haravan app client_id (env HARAVAN_APP_ID)")
	cmd.Flags().StringVarP(&appSecret, "app-secret", "s", "", "Haravan app client_secret (env HARAVAN_APP_SECRET)")
	cmd.Flags().StringVar(&scopesCSV, "scopes", "", "space- or comma-separated OAuth scopes (default: read/write scopes for all domains)")
	cmd.Flags().IntVar(&port, "port", 3000, "local callback port")
	cmd.Flags().BoolVar(&save, "save-app", false, "persist app-id/app-secret to ~/.haravan-cli/config.json")
	return cmd
}

func newLogoutCmd(f *cmdutil.Factory) *cobra.Command {
	var appID string
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove a stored token",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := f.Config()
			if err != nil {
				return err
			}
			if appID == "" {
				appID = firstNonEmpty(os.Getenv(config.EnvAppID), cfg.AppID)
			}
			if appID == "" {
				return errors.New("--app-id required (or set HARAVAN_APP_ID, or configure a default app)")
			}
			store, err := auth.NewTokenStore()
			if err != nil {
				return err
			}
			if err := store.Delete(appID); err != nil {
				return err
			}
			fmt.Fprintf(f.IOStreams.Out, "removed token for app %s\n", appID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&appID, "app-id", "a", "", "app to log out")
	return cmd
}

func newStatusCmd(f *cmdutil.Factory) *cobra.Command {
	var jsonOut bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show credential status",
		Long:  "Print the source of the effective access token (flag > env > stored) and any stored tokens.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := f.Config()
			store, err := auth.NewTokenStore()
			if err != nil {
				return err
			}
			all, err := store.LoadAll()
			if err != nil {
				return err
			}

			type entry struct {
				AppID     string   `json:"app_id"`
				Scope     []string `json:"scope,omitempty"`
				ExpiresAt int64    `json:"expires_at,omitempty"`
				Expired   bool     `json:"expired"`
			}
			entries := make([]entry, 0, len(all))
			for id, tok := range all {
				entries = append(entries, entry{
					AppID:     id,
					Scope:     tok.Scope,
					ExpiresAt: tok.ExpiresAt,
					Expired:   tok.IsExpired(),
				})
			}

			hasEnv := os.Getenv(config.EnvAccessToken) != ""
			source := "none"
			switch {
			case hasEnv:
				source = "env HARAVAN_ACCESS_TOKEN"
			case len(entries) > 0:
				source = "stored (oauth)"
			}

			if jsonOut {
				return writeJSON(cmd, map[string]any{
					"source":        source,
					"stored_tokens": entries,
					"config_app_id": cfg.AppID,
				})
			}

			fmt.Fprintf(f.IOStreams.Out, "effective credential source: %s\n", source)
			if cfg.AppID != "" {
				fmt.Fprintf(f.IOStreams.Out, "default app_id (config): %s\n", cfg.AppID)
			}
			if len(entries) == 0 {
				fmt.Fprintln(f.IOStreams.Out, "no OAuth tokens stored")
				return nil
			}
			fmt.Fprintln(f.IOStreams.Out, "stored tokens:")
			for _, e := range entries {
				exp := "never"
				if e.ExpiresAt > 0 {
					t := time.UnixMilli(e.ExpiresAt).UTC().Format(time.RFC3339)
					if e.Expired {
						exp = t + " (EXPIRED)"
					} else {
						exp = t
					}
				}
				fmt.Fprintf(f.IOStreams.Out, "  - %s  expires=%s  scopes=%s\n", e.AppID, exp, strings.Join(e.Scope, " "))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	return cmd
}

func writeJSON(cmd *cobra.Command, v any) error {
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func splitScopes(csv string) []string {
	csv = strings.ReplaceAll(csv, ",", " ")
	parts := strings.Fields(csv)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
