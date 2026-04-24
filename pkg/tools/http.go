package tools

import (
	"fmt"
	"net/url"
	"strings"
)

// Pathf returns path with each `{key}` placeholder replaced by the
// corresponding Input value. Missing keys leave the placeholder intact —
// the caller is expected to mark the relevant Flag as Required.
func Pathf(path string, in Input, keys ...string) string {
	for _, k := range keys {
		ph := "{" + k + "}"
		if !strings.Contains(path, ph) {
			continue
		}
		v, ok := in[k]
		if !ok {
			continue
		}
		path = strings.ReplaceAll(path, ph, fmt.Sprint(v))
	}
	return path
}

// Query builds a url.Values from the named Input keys, skipping empty
// strings, zero integers, and false booleans so upstream endpoints don't
// receive meaningless filters.
func Query(in Input, keys ...string) url.Values {
	q := url.Values{}
	for _, k := range keys {
		raw, ok := in[k]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case string:
			if v != "" {
				q.Set(k, v)
			}
		case int:
			if v != 0 {
				q.Set(k, fmt.Sprint(v))
			}
		case int64:
			if v != 0 {
				q.Set(k, fmt.Sprint(v))
			}
		case bool:
			if v {
				q.Set(k, "true")
			}
		case []string:
			if len(v) > 0 {
				q.Set(k, strings.Join(v, ","))
			}
		default:
			s := fmt.Sprint(v)
			if s != "" {
				q.Set(k, s)
			}
		}
	}
	return q
}
