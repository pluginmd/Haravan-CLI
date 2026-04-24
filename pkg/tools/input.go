package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Input is a Handler's decoded inputs, keyed by Flag.Name. Values are the
// natural Go types corresponding to Flag.Type (string, int64, bool,
// []string, json.RawMessage). Missing keys indicate the flag was not set.
type Input map[string]any

// String returns the string value for the named flag, or "" if missing.
func (i Input) String(name string) string {
	v, _ := i[name].(string)
	return v
}

// StringOr returns the string value for the named flag, or fallback if missing/empty.
func (i Input) StringOr(name, fallback string) string {
	if v, ok := i[name].(string); ok && v != "" {
		return v
	}
	return fallback
}

// Int64 returns the int64 value for the named flag, or 0 if missing.
func (i Input) Int64(name string) int64 {
	switch v := i[name].(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// Int returns the int value for the named flag, or 0 if missing.
func (i Input) Int(name string) int { return int(i.Int64(name)) }

// Bool returns the bool value for the named flag, or false if missing.
func (i Input) Bool(name string) bool {
	v, _ := i[name].(bool)
	return v
}

// StringList returns the slice of strings for the named flag, or nil if missing.
func (i Input) StringList(name string) []string {
	switch v := i[name].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// JSON returns the raw JSON payload for the named flag, or nil if missing.
// The value is already a valid JSON document (object, array, scalar).
func (i Input) JSON(name string) json.RawMessage {
	switch v := i[name].(type) {
	case json.RawMessage:
		return v
	case []byte:
		return json.RawMessage(v)
	case string:
		if v == "" {
			return nil
		}
		return json.RawMessage(v)
	default:
		if v == nil {
			return nil
		}
		raw, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		return raw
	}
}

// Has reports whether the named flag was provided.
func (i Input) Has(name string) bool {
	_, ok := i[name]
	return ok
}

// resolveJSONValue turns a CLI JSON flag value into json.RawMessage.
// It accepts an inline JSON literal or "@path" to read from a file.
func resolveJSONValue(raw string) (json.RawMessage, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if strings.HasPrefix(raw, "@") {
		path := raw[1:]
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read json file %s: %w", path, err)
		}
		raw = strings.TrimSpace(string(data))
	}
	if !json.Valid([]byte(raw)) {
		return nil, fmt.Errorf("invalid JSON for flag body")
	}
	return json.RawMessage(raw), nil
}
