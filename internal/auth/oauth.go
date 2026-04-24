package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// Haravan OAuth 2.0 endpoints.
const (
	AuthorizeURL = "https://accounts.haravan.com/connect/authorize"
	TokenURL     = "https://accounts.haravan.com/connect/token"
)

// OAuthConfig carries the parameters required to run an authorization-code flow.
type OAuthConfig struct {
	AppID       string
	AppSecret   string
	Scopes      []string
	RedirectURL string        // default: http://localhost:<port>/callback
	Port        int           // default: 3000
	Timeout     time.Duration // default: 5 minutes
}

func (c *OAuthConfig) applyDefaults() {
	if c.Port == 0 {
		c.Port = 3000
	}
	if c.RedirectURL == "" {
		c.RedirectURL = fmt.Sprintf("http://localhost:%d/callback", c.Port)
	}
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Minute
	}
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
}

// PerformLogin runs the OAuth authorization-code flow:
//  1. starts a local HTTP callback server
//  2. opens the browser to the authorize URL with a CSRF state token
//  3. waits for the callback, verifies state, exchanges code for tokens
//
// It returns the fresh StoredToken. The caller is responsible for persisting.
// Log messages are written via logger.Infof so stdout stays clean.
type LoginFeedback struct {
	OnAuthURL   func(url string)
	OnBrowserOK func()
	OnSuccess   func()
}

func PerformLogin(ctx context.Context, cfg OAuthConfig, fb LoginFeedback) (StoredToken, error) {
	cfg.applyDefaults()
	if cfg.AppID == "" || cfg.AppSecret == "" {
		return StoredToken{}, errors.New("app_id and app_secret are required for OAuth login")
	}

	stateBytes := make([]byte, 32)
	if _, err := rand.Read(stateBytes); err != nil {
		return StoredToken{}, fmt.Errorf("generate state: %w", err)
	}
	state := hex.EncodeToString(stateBytes)

	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.Port))
	if err != nil {
		return StoredToken{}, fmt.Errorf("bind callback port %d: %w", cfg.Port, err)
	}

	type cbResult struct {
		token StoredToken
		err   error
	}
	done := make(chan cbResult, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotState := q.Get("state")
		if gotState != state {
			http.Error(w, "invalid state (possible CSRF)", http.StatusForbidden)
			done <- cbResult{err: errors.New("oauth state mismatch")}
			return
		}
		code := q.Get("code")
		if code == "" {
			http.Error(w, "no authorization code", http.StatusBadRequest)
			done <- cbResult{err: errors.New("oauth: no authorization code")}
			return
		}
		tok, err := exchangeCode(r.Context(), cfg, code)
		if err != nil {
			http.Error(w, "token exchange failed: "+err.Error(), http.StatusInternalServerError)
			done <- cbResult{err: err}
			return
		}
		fmt.Fprint(w, `<!doctype html><html><body style="font-family:system-ui;padding:40px;text-align:center">
<h1>haravan-cli — login successful</h1>
<p>You can close this tab and return to your terminal.</p>
</body></html>`)
		done <- cbResult{token: tok}
	})

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = srv.Serve(listener) }()
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	authURL := buildAuthorizeURL(cfg, state)
	if fb.OnAuthURL != nil {
		fb.OnAuthURL(authURL)
	}
	if err := openBrowser(authURL); err == nil {
		if fb.OnBrowserOK != nil {
			fb.OnBrowserOK()
		}
	}

	ctx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	select {
	case r := <-done:
		if r.err != nil {
			return StoredToken{}, r.err
		}
		if fb.OnSuccess != nil {
			fb.OnSuccess()
		}
		return r.token, nil
	case <-ctx.Done():
		return StoredToken{}, fmt.Errorf("oauth login timed out after %s", cfg.Timeout)
	}
}

func buildAuthorizeURL(cfg OAuthConfig, state string) string {
	q := url.Values{}
	q.Set("response_type", "code")
	q.Set("client_id", cfg.AppID)
	q.Set("redirect_uri", cfg.RedirectURL)
	q.Set("scope", strings.Join(cfg.Scopes, " "))
	q.Set("state", state)
	q.Set("nonce", fmt.Sprintf("%d", time.Now().UnixMilli()))
	return AuthorizeURL + "?" + q.Encode()
}

func exchangeCode(ctx context.Context, cfg OAuthConfig, code string) (StoredToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", cfg.RedirectURL)
	form.Set("client_id", cfg.AppID)
	form.Set("client_secret", cfg.AppSecret)
	return doTokenRequest(ctx, cfg.AppID, cfg.Scopes, form)
}

// Refresh trades a refresh token for a fresh access token.
func Refresh(ctx context.Context, appID, appSecret, refreshToken string) (StoredToken, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", appID)
	form.Set("client_secret", appSecret)
	return doTokenRequest(ctx, appID, nil, form)
}

func doTokenRequest(ctx context.Context, appID string, fallbackScopes []string, form url.Values) (StoredToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return StoredToken{}, fmt.Errorf("build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return StoredToken{}, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return StoredToken{}, fmt.Errorf("token endpoint %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return StoredToken{}, fmt.Errorf("decode token response: %w", err)
	}
	scope := fallbackScopes
	if tr.Scope != "" {
		scope = strings.Fields(tr.Scope)
	}
	tok := StoredToken{
		AccessToken:  tr.AccessToken,
		RefreshToken: tr.RefreshToken,
		AppID:        appID,
		Scope:        scope,
		CreatedAt:    time.Now().UnixMilli(),
	}
	if tr.ExpiresIn > 0 {
		tok.ExpiresAt = time.Now().UnixMilli() + tr.ExpiresIn*1000
	}
	return tok, nil
}

// openBrowser launches the user's default browser. Best-effort; callers should
// always print the URL so the user can open it manually.
func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}
