package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInputAccessors(t *testing.T) {
	in := Input{
		"s":   "hello",
		"i":   123,
		"i64": int64(456),
		"b":   true,
		"sl":  []string{"a", "b"},
	}
	if in.String("s") != "hello" {
		t.Error("String")
	}
	if in.Int("i") != 123 {
		t.Error("Int")
	}
	if in.Int64("i64") != 456 {
		t.Error("Int64")
	}
	if !in.Bool("b") {
		t.Error("Bool")
	}
	if got := in.StringList("sl"); len(got) != 2 || got[0] != "a" {
		t.Errorf("StringList: %v", got)
	}
	if !in.Has("s") || in.Has("missing") {
		t.Error("Has")
	}
	if got := in.StringOr("missing", "default"); got != "default" {
		t.Errorf("StringOr fallback: %v", got)
	}
}

func TestResolveJSONInline(t *testing.T) {
	raw, err := resolveJSONValue(`{"x":1}`)
	if err != nil {
		t.Fatalf("resolveJSONValue: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["x"] != float64(1) {
		t.Errorf("decoded: %v", got)
	}
}

func TestResolveJSONFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "body.json")
	if err := os.WriteFile(path, []byte(`{"y":2}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	raw, err := resolveJSONValue("@" + path)
	if err != nil {
		t.Fatalf("resolveJSONValue: %v", err)
	}
	if string(raw) != `{"y":2}` {
		t.Errorf("file body: %s", raw)
	}
}

func TestResolveJSONInvalid(t *testing.T) {
	if _, err := resolveJSONValue(`not-json`); err == nil {
		t.Error("expected error for invalid JSON")
	}
}
