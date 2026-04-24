package tools

import "encoding/json"

// Result is what a Handler returns. Either Data (structured JSON) or Text
// (preformatted output) may be empty; at least one is expected.
type Result struct {
	// Data is the primary JSON payload. When present it is the canonical
	// output — CLI prints it (pretty by default), MCP emits it as a
	// JSON-text content block.
	Data json.RawMessage

	// Text is an optional human-oriented rendering. When Data is empty,
	// Text becomes the MCP content block verbatim.
	Text string

	// IsError flips both surfaces into "tool failed" mode. The framework
	// also sets this automatically when Handler returns an error.
	IsError bool
}

// NewJSON wraps any Go value as a JSON Result. Marshaling errors surface
// to the caller so they can choose to bail or return a text-only Result.
func NewJSON(v any) (*Result, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &Result{Data: b}, nil
}

// NewRaw wraps an existing json.RawMessage without re-marshaling.
func NewRaw(raw json.RawMessage) *Result { return &Result{Data: raw} }

// NewText returns a text-only Result.
func NewText(s string) *Result { return &Result{Text: s} }
