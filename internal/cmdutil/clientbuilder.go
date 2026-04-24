package cmdutil

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/pluginmd/haravan-cli/internal/auth"
	"github.com/pluginmd/haravan-cli/internal/client"
	"github.com/pluginmd/haravan-cli/internal/config"
)

// Persistent flag names used by both the root command and tool subcommands.
const (
	FlagToken     = "token"
	FlagAppID     = "app-id"
	FlagAppSecret = "app-secret"
	FlagAPIBase   = "api-base"
)

// RegisterAuthFlags adds --token / --app-id / --app-secret / --api-base
// as persistent flags on the given command (normally the root). They're
// picked up by BuildClient further down the tree.
func RegisterAuthFlags(cmd *cobra.Command) {
	pf := cmd.PersistentFlags()
	pf.String(FlagToken, "", "Haravan access token (env HARAVAN_ACCESS_TOKEN)")
	pf.String(FlagAppID, "", "Haravan app client_id for OAuth (env HARAVAN_APP_ID)")
	pf.String(FlagAppSecret, "", "Haravan app client_secret for OAuth refresh (env HARAVAN_APP_SECRET)")
	pf.String(FlagAPIBase, "", "Haravan API base URL (env HARAVAN_API_BASE)")
}

// BuildClient resolves credentials (flag > env > stored) and returns a
// ready-to-use Haravan client. It is the single entry point used by tool
// handlers, mcp serve, and the raw `api` command.
func BuildClient(ctx context.Context, cmd *cobra.Command, cfg *config.Config) (*client.Client, error) {
	flagVal := func(name string) string {
		if cmd == nil {
			return ""
		}
		v, err := cmd.Flags().GetString(name)
		if err != nil {
			return ""
		}
		return v
	}

	store, err := auth.NewTokenStore()
	if err != nil {
		return nil, err
	}
	res := &auth.Resolver{
		Token:     flagVal(FlagToken),
		AppID:     flagVal(FlagAppID),
		AppSecret: flagVal(FlagAppSecret),
		Store:     store,
		Config:    cfg,
	}
	token, err := res.Resolve(ctx)
	if err != nil {
		return nil, err
	}

	base := flagVal(FlagAPIBase)
	if base == "" && cfg != nil {
		base = cfg.ResolveAPIBase()
	}
	if base == "" {
		base = config.DefaultAPIBase
	}

	c, err := client.New(client.Options{BaseURL: base, AccessToken: token})
	if err != nil {
		return nil, fmt.Errorf("build client: %w", err)
	}
	return c, nil
}
