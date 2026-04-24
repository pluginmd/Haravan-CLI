package tools

import (
	"encoding/json"
	"errors"
	"fmt"
)

// EnvelopeWrap returns raw wrapped as {wrapper: raw} unless it's already
// been wrapped that way. Haravan endpoints uniformly expect single-key
// envelopes (order, product, variant, customer, page, …) as the request
// body; this helper lets CLI users pass either form.
//
//	EnvelopeWrap({"title": "t"}, "product")       -> {"product": {"title":"t"}}
//	EnvelopeWrap({"product": {...}}, "product")   -> {"product": {...}} unchanged
func EnvelopeWrap(raw json.RawMessage, wrapper string) (json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, errors.New("empty body")
	}
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("body must be a JSON object: %w", err)
	}
	if _, ok := probe[wrapper]; ok {
		return raw, nil
	}
	return json.Marshal(map[string]json.RawMessage{wrapper: raw})
}
