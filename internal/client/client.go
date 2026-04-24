package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/pluginmd/haravan-cli/internal/build"
	"github.com/pluginmd/haravan-cli/internal/logger"
)

// Response is the decoded envelope returned from every request.
type Response struct {
	Status        int
	Header        http.Header
	Body          json.RawMessage // raw body for flexible parsing
	RateLimitUsed float64
	RateLimitMax  float64
	RetryAfter    float64
}

// Decode unmarshals the body into v.
func (r *Response) Decode(v any) error {
	if len(r.Body) == 0 {
		return nil
	}
	return json.Unmarshal(r.Body, v)
}

// Options configures a client.
type Options struct {
	BaseURL     string
	AccessToken string
	Timeout     time.Duration
	UserAgent   string
	HTTPClient  *http.Client // optional override; nil → default
}

// Client is a thin, typed wrapper around the Haravan REST API.
// It is safe for concurrent use.
type Client struct {
	baseURL     *url.URL
	accessToken string
	userAgent   string
	httpClient  *http.Client
	bucket      *bucket
}

// New returns a ready-to-use client. BaseURL and AccessToken are required.
func New(opts Options) (*Client, error) {
	if opts.BaseURL == "" {
		return nil, fmt.Errorf("client: BaseURL is required")
	}
	if opts.AccessToken == "" {
		return nil, fmt.Errorf("client: AccessToken is required")
	}
	u, err := url.Parse(opts.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	hc := opts.HTTPClient
	if hc == nil {
		timeout := opts.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}
		hc = &http.Client{Timeout: timeout}
	}
	ua := opts.UserAgent
	if ua == "" {
		ua = fmt.Sprintf("haravan-cli/%s", build.Version)
	}
	return &Client{
		baseURL:     u,
		accessToken: opts.AccessToken,
		userAgent:   ua,
		httpClient:  hc,
		bucket:      newBucket(),
	}, nil
}

// SetAccessToken updates the bearer token used on subsequent calls.
func (c *Client) SetAccessToken(token string) { c.accessToken = token }

// Get sends a GET request. params are URL-encoded as the query string.
func (c *Client) Get(ctx context.Context, path string, params url.Values) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, params, nil)
}

// Post sends a POST with a JSON body.
func (c *Client) Post(ctx context.Context, path string, body any) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, nil, body)
}

// Put sends a PUT with a JSON body.
func (c *Client) Put(ctx context.Context, path string, body any) (*Response, error) {
	return c.do(ctx, http.MethodPut, path, nil, body)
}

// Delete sends a DELETE.
func (c *Client) Delete(ctx context.Context, path string) (*Response, error) {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// Request is the escape hatch for arbitrary HTTP verbs (used by the api
// command and rare one-off tools).
func (c *Client) Request(ctx context.Context, method, path string, params url.Values, body any) (*Response, error) {
	return c.do(ctx, strings.ToUpper(method), path, params, body)
}

func (c *Client) do(ctx context.Context, method, path string, params url.Values, body any) (*Response, error) {
	if err := c.bucket.waitIfFull(ctx); err != nil {
		return nil, err
	}

	u := c.resolveURL(path, params)

	var bodyReader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	start := time.Now()
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, u, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	used, max, ok := parseLimitHeader(resp.Header.Get("X-Haravan-Api-Call-Limit"))
	if ok {
		c.bucket.update(used, max)
	}
	retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))

	logger.Debugf("%s %s -> %d (%.0fms, rate %.0f/%.0f)",
		method, u, resp.StatusCode, float64(time.Since(start).Milliseconds()), used, max)

	out := &Response{
		Status:        resp.StatusCode,
		Header:        resp.Header,
		Body:          json.RawMessage(raw),
		RateLimitUsed: used,
		RateLimitMax:  max,
		RetryAfter:    retryAfter,
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		apiErr := &APIError{
			Status:     resp.StatusCode,
			Path:       path,
			Method:     method,
			Message:    extractMessage(raw),
			Details:    json.RawMessage(raw),
			RetryAfter: retryAfter,
		}
		return out, classifyStatus(apiErr)
	}
	return out, nil
}

func (c *Client) resolveURL(path string, params url.Values) string {
	ref, err := url.Parse(path)
	if err != nil {
		// fall back to naive join if path isn't parseable
		return strings.TrimRight(c.baseURL.String(), "/") + "/" + strings.TrimLeft(path, "/")
	}
	u := c.baseURL.ResolveReference(ref)
	if len(params) > 0 {
		q := u.Query()
		for k, vs := range params {
			for _, v := range vs {
				q.Add(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String()
}

// extractMessage tries to pull a human-oriented error message from a JSON body.
// Haravan commonly returns {"errors": "..."} or {"errors": {"field": [...]}}.
func extractMessage(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var env struct {
		Errors  json.RawMessage `json:"errors"`
		Error   string          `json:"error"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		if len(raw) > 200 {
			return string(raw[:200])
		}
		return string(raw)
	}
	if env.Message != "" {
		return env.Message
	}
	if env.Error != "" {
		return env.Error
	}
	if len(env.Errors) > 0 {
		s := string(env.Errors)
		if len(s) > 200 {
			s = s[:200]
		}
		return s
	}
	return ""
}
