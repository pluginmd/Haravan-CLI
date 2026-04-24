// Package config handles persistent CLI configuration at ~/.haravan-cli/config.json.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Environment variable names. Keep them namespaced under HARAVAN_*.
const (
	EnvHome        = "HARAVAN_CLI_HOME"
	EnvAPIBase     = "HARAVAN_API_BASE"
	EnvAccessToken = "HARAVAN_ACCESS_TOKEN"
	EnvAppID       = "HARAVAN_APP_ID"
	EnvAppSecret   = "HARAVAN_APP_SECRET"
	EnvLogLevel    = "HARAVAN_LOG_LEVEL"
)

const (
	DefaultAPIBase = "https://apis.haravan.com"
	configFileName = "config.json"
)

// Config is the persisted user configuration.
type Config struct {
	APIBase     string `json:"api_base,omitempty"`
	AppID       string `json:"app_id,omitempty"`
	AppSecret   string `json:"app_secret,omitempty"`
	DefaultAuth string `json:"default_auth,omitempty"` // "token" | "oauth"
}

// Home returns the config directory (~/.haravan-cli by default, overridable via HARAVAN_CLI_HOME).
func Home() (string, error) {
	if v := os.Getenv(EnvHome); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".haravan-cli"), nil
}

// Path returns the absolute path to config.json.
func Path() (string, error) {
	dir, err := Home()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configFileName), nil
}

// Load reads config.json. Missing file is not an error; a zero Config is returned.
func Load() (*Config, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}

// Save writes config.json atomically with 0600 permissions.
func Save(c *Config) error {
	dir, err := Home()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir config home: %w", err)
	}
	p, err := Path()
	if err != nil {
		return err
	}
	buf, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o600); err != nil {
		return fmt.Errorf("write temp config: %w", err)
	}
	if err := os.Rename(tmp, p); err != nil {
		return fmt.Errorf("rename config: %w", err)
	}
	return nil
}

// APIBase resolves the effective base URL: env > config > default.
func (c *Config) ResolveAPIBase() string {
	if v := os.Getenv(EnvAPIBase); v != "" {
		return v
	}
	if c != nil && c.APIBase != "" {
		return c.APIBase
	}
	return DefaultAPIBase
}
