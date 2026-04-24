// Package client wraps the Haravan REST API.
package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// APIError is the base error type for non-2xx responses from Haravan.
// Specific status codes (401, 403, 404, 422, 429, 5xx) are exposed via
// the typed errors below; callers can use errors.As to discriminate.
type APIError struct {
	Status     int
	Path       string
	Method     string
	Message    string
	Details    json.RawMessage // raw response body
	RetryAfter float64         // seconds, only set for 429
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "haravan api: %s %s", e.Method, e.Path)
	if e.Status != 0 {
		fmt.Fprintf(&b, " -> %d", e.Status)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ": %s", e.Message)
	}
	return b.String()
}

// Typed wrappers keep call sites readable via errors.As.
type (
	UnauthorizedError struct{ *APIError }
	ForbiddenError    struct{ *APIError }
	NotFoundError     struct{ *APIError }
	ValidationError   struct{ *APIError }
	RateLimitError    struct{ *APIError }
	ServerError       struct{ *APIError }
)

func (e *UnauthorizedError) Unwrap() error { return e.APIError }
func (e *ForbiddenError) Unwrap() error    { return e.APIError }
func (e *NotFoundError) Unwrap() error     { return e.APIError }
func (e *ValidationError) Unwrap() error   { return e.APIError }
func (e *RateLimitError) Unwrap() error    { return e.APIError }
func (e *ServerError) Unwrap() error       { return e.APIError }

// classifyStatus wraps an APIError in the matching typed error.
func classifyStatus(e *APIError) error {
	switch {
	case e.Status == 401:
		return &UnauthorizedError{APIError: e}
	case e.Status == 403:
		return &ForbiddenError{APIError: e}
	case e.Status == 404:
		return &NotFoundError{APIError: e}
	case e.Status == 422:
		return &ValidationError{APIError: e}
	case e.Status == 429:
		return &RateLimitError{APIError: e}
	case e.Status >= 500:
		return &ServerError{APIError: e}
	default:
		return e
	}
}

// FriendlyMessage returns a human-oriented summary suitable for CLI output
// and MCP error text. It preserves the error's detail payload if present.
func FriendlyMessage(err error) string {
	if err == nil {
		return ""
	}
	var (
		u  *UnauthorizedError
		f  *ForbiddenError
		nf *NotFoundError
		v  *ValidationError
		rl *RateLimitError
		se *ServerError
		ge *APIError
	)
	switch {
	case errors.As(err, &u):
		return "Authentication failed — check your access token or re-run `haravan-cli auth login`."
	case errors.As(err, &f):
		return "Forbidden — the app is missing a required scope. " + f.APIError.detailLine()
	case errors.As(err, &nf):
		return "Resource not found: " + nf.APIError.Path
	case errors.As(err, &v):
		return "Validation error: " + v.APIError.detailLine()
	case errors.As(err, &rl):
		return fmt.Sprintf("Rate limited — retry after %.1fs.", rl.APIError.RetryAfter)
	case errors.As(err, &se):
		return fmt.Sprintf("Haravan server error (%d) — try again shortly.", se.APIError.Status)
	case errors.As(err, &ge):
		if ge.Message != "" {
			return ge.Message
		}
		return ge.Error()
	default:
		return err.Error()
	}
}

func (e *APIError) detailLine() string {
	if len(e.Details) == 0 {
		return e.Message
	}
	return e.Message + " " + string(e.Details)
}
