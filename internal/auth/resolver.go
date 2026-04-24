package auth

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/pluginmd/haravan-cli/internal/config"
)

// Resolver produces the effective access token for an outbound Haravan call.
//
// Priority order:
//  1. Explicit Token field (CLI --token flag)
//  2. HARAVAN_ACCESS_TOKEN env
//  3. Stored OAuth token for AppID (refreshing if expired)
//
// Callers that need a raw private app token should set Token directly and
// never reach the stored-token path.
type Resolver struct {
	Token     string // explicit, highest priority
	AppID     string
	AppSecret string
	Store     *TokenStore
	Config    *config.Config
}

// ErrNoToken is returned when no credential can be resolved.
var ErrNoToken = errors.New("no Haravan access token: pass --token, set HARAVAN_ACCESS_TOKEN, or run `haravan-cli auth login`")

// Resolve returns a valid access token or an error describing what's missing.
func (r *Resolver) Resolve(ctx context.Context) (string, error) {
	if r.Token != "" {
		return r.Token, nil
	}
	if v := os.Getenv(config.EnvAccessToken); v != "" {
		return v, nil
	}

	appID := r.effectiveAppID()
	if appID == "" || r.Store == nil {
		return "", ErrNoToken
	}

	tok, ok, err := r.Store.Load(appID)
	if err != nil {
		return "", fmt.Errorf("load stored token: %w", err)
	}
	if !ok {
		return "", ErrNoToken
	}

	if tok.IsExpired() {
		appSecret := r.effectiveAppSecret()
		if tok.RefreshToken == "" || appSecret == "" {
			return "", fmt.Errorf("token for app %q expired and cannot refresh (missing refresh token or app secret); run `haravan-cli auth login` again", appID)
		}
		fresh, err := Refresh(ctx, appID, appSecret, tok.RefreshToken)
		if err != nil {
			return "", fmt.Errorf("refresh token for app %q: %w", appID, err)
		}
		if err := r.Store.Save(appID, fresh); err != nil {
			return "", fmt.Errorf("persist refreshed token: %w", err)
		}
		return fresh.AccessToken, nil
	}
	return tok.AccessToken, nil
}

func (r *Resolver) effectiveAppID() string {
	if r.AppID != "" {
		return r.AppID
	}
	if v := os.Getenv(config.EnvAppID); v != "" {
		return v
	}
	if r.Config != nil {
		return r.Config.AppID
	}
	return ""
}

func (r *Resolver) effectiveAppSecret() string {
	if r.AppSecret != "" {
		return r.AppSecret
	}
	if v := os.Getenv(config.EnvAppSecret); v != "" {
		return v
	}
	if r.Config != nil {
		return r.Config.AppSecret
	}
	return ""
}
