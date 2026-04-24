package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, h http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	c, err := New(Options{BaseURL: srv.URL, AccessToken: "tok_test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c, srv
}

func TestClientGetInjectsAuthAndParses(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok_test" {
			t.Errorf("auth header: %q", got)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Errorf("query.limit: %q", got)
		}
		w.Header().Set("X-Haravan-Api-Call-Limit", "7/80")
		w.WriteHeader(200)
		_, _ = io.WriteString(w, `{"shop":{"name":"demo"}}`)
	})
	defer srv.Close()

	q := url.Values{}
	q.Set("limit", "5")
	resp, err := c.Get(context.Background(), "/com/shop.json", q)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if resp.Status != 200 {
		t.Errorf("status: %d", resp.Status)
	}
	if resp.RateLimitUsed != 7 || resp.RateLimitMax != 80 {
		t.Errorf("rate limit: %v/%v", resp.RateLimitUsed, resp.RateLimitMax)
	}
	var out struct{ Shop struct{ Name string } }
	if err := resp.Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Shop.Name != "demo" {
		t.Errorf("shop.name: %q", out.Shop.Name)
	}
}

func TestClientPostMarshalsJSON(t *testing.T) {
	var gotBody map[string]any
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type: %q", ct)
		}
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(201)
		_, _ = io.WriteString(w, `{}`)
	})
	defer srv.Close()

	_, err := c.Post(context.Background(), "/com/orders.json", map[string]any{
		"order": map[string]any{"email": "x@y.z"},
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	order, ok := gotBody["order"].(map[string]any)
	if !ok || order["email"] != "x@y.z" {
		t.Errorf("body: %+v", gotBody)
	}
}

func TestClientErrorClassification(t *testing.T) {
	cases := []struct {
		status int
		body   string
		test   func(error) bool
	}{
		{401, `{"errors":"Unauthorized"}`, func(err error) bool {
			var e *UnauthorizedError
			return errors.As(err, &e)
		}},
		{403, `{"errors":"Scope missing"}`, func(err error) bool {
			var e *ForbiddenError
			return errors.As(err, &e)
		}},
		{404, `{}`, func(err error) bool {
			var e *NotFoundError
			return errors.As(err, &e)
		}},
		{422, `{"errors":{"email":["taken"]}}`, func(err error) bool {
			var e *ValidationError
			return errors.As(err, &e)
		}},
		{429, `{"errors":"Too many"}`, func(err error) bool {
			var e *RateLimitError
			return errors.As(err, &e)
		}},
		{500, `oops`, func(err error) bool {
			var e *ServerError
			return errors.As(err, &e)
		}},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = io.WriteString(w, tc.body)
			})
			defer srv.Close()
			_, err := c.Get(context.Background(), "/x", nil)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.test(err) {
				t.Errorf("status %d not classified: %v", tc.status, err)
			}
		})
	}
}

func TestClientRetryAfterParsed(t *testing.T) {
	c, srv := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "3.5")
		w.WriteHeader(429)
		_, _ = io.WriteString(w, `{"errors":"slow down"}`)
	})
	defer srv.Close()
	_, err := c.Get(context.Background(), "/x", nil)
	var rl *RateLimitError
	if !errors.As(err, &rl) {
		t.Fatalf("want RateLimitError, got %v", err)
	}
	if rl.APIError.RetryAfter != 3.5 {
		t.Errorf("retry-after: %v", rl.APIError.RetryAfter)
	}
}

func TestExtractMessagePrefers(t *testing.T) {
	if got := extractMessage([]byte(`{"message":"m","error":"e","errors":"errs"}`)); got != "m" {
		t.Errorf("message: %q", got)
	}
	if got := extractMessage([]byte(`{"error":"e","errors":"errs"}`)); got != "e" {
		t.Errorf("error: %q", got)
	}
	if got := extractMessage([]byte(`{"errors":"errs"}`)); !strings.Contains(got, "errs") {
		t.Errorf("errors: %q", got)
	}
	if got := extractMessage([]byte(`plain text`)); got != "plain text" {
		t.Errorf("plain: %q", got)
	}
}
