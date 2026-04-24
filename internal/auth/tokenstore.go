// Package auth handles Haravan token storage and OAuth flow.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pluginmd/haravan-cli/internal/config"
)

const tokenFileName = "tokens.json"

// StoredToken represents a persisted credential for one app.
type StoredToken struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token,omitempty"`
	ExpiresAt    int64    `json:"expires_at,omitempty"` // unix millis; 0 = no expiry tracked
	AppID        string   `json:"app_id,omitempty"`
	Scope        []string `json:"scope,omitempty"`
	CreatedAt    int64    `json:"created_at"`
}

// IsExpired reports whether the token's expiry is in the past.
// Tokens without ExpiresAt are assumed valid (private app tokens).
func (t StoredToken) IsExpired() bool {
	if t.ExpiresAt == 0 {
		return false
	}
	return time.Now().UnixMilli() > t.ExpiresAt
}

// TokenStore wraps tokens.json read/write operations.
type TokenStore struct {
	path string
}

// NewTokenStore returns a store anchored under the CLI's config home.
func NewTokenStore() (*TokenStore, error) {
	home, err := config.Home()
	if err != nil {
		return nil, err
	}
	return &TokenStore{path: filepath.Join(home, tokenFileName)}, nil
}

// Path returns the absolute path to tokens.json.
func (s *TokenStore) Path() string { return s.path }

// LoadAll returns all stored tokens keyed by appID. Missing file returns an empty map.
func (s *TokenStore) LoadAll() (map[string]StoredToken, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]StoredToken{}, nil
		}
		return nil, fmt.Errorf("read token store: %w", err)
	}
	out := map[string]StoredToken{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("parse token store: %w", err)
	}
	return out, nil
}

// Load returns the token for appID, or (zero, false) if not found.
func (s *TokenStore) Load(appID string) (StoredToken, bool, error) {
	all, err := s.LoadAll()
	if err != nil {
		return StoredToken{}, false, err
	}
	t, ok := all[appID]
	return t, ok, nil
}

// Save upserts a token for appID and writes atomically with 0600 permissions.
func (s *TokenStore) Save(appID string, tok StoredToken) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("mkdir token dir: %w", err)
	}
	all, err := s.LoadAll()
	if err != nil {
		return err
	}
	all[appID] = tok
	return s.writeAll(all)
}

// Delete removes a token for appID. No-op if not present.
func (s *TokenStore) Delete(appID string) error {
	all, err := s.LoadAll()
	if err != nil {
		return err
	}
	if _, ok := all[appID]; !ok {
		return nil
	}
	delete(all, appID)
	return s.writeAll(all)
}

func (s *TokenStore) writeAll(all map[string]StoredToken) error {
	buf, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		return fmt.Errorf("encode tokens: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o600); err != nil {
		return fmt.Errorf("write temp tokens: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("rename tokens: %w", err)
	}
	return nil
}
