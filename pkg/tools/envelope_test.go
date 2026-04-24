package tools

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEnvelopeWrapWrapsBare(t *testing.T) {
	got, err := EnvelopeWrap([]byte(`{"title":"t","price":10}`), "product")
	if err != nil {
		t.Fatalf("EnvelopeWrap: %v", err)
	}
	var out map[string]map[string]any
	if err := json.Unmarshal(got, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	p := out["product"]
	if p == nil || p["title"] != "t" {
		t.Errorf("expected wrapped object, got %s", got)
	}
}

func TestEnvelopeWrapLeavesWrappedAlone(t *testing.T) {
	raw := []byte(`{"product":{"title":"t"}}`)
	got, err := EnvelopeWrap(raw, "product")
	if err != nil {
		t.Fatalf("EnvelopeWrap: %v", err)
	}
	// Normalize both sides by re-unmarshaling
	if !jsonEqual(t, got, raw) {
		t.Errorf("expected passthrough, got %s", got)
	}
}

func TestEnvelopeWrapErrorsOnInvalidJSON(t *testing.T) {
	_, err := EnvelopeWrap([]byte(`not-json`), "x")
	if err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Errorf("expected object error, got %v", err)
	}
}

func jsonEqual(t *testing.T, a, b []byte) bool {
	t.Helper()
	var aa, bb any
	if err := json.Unmarshal(a, &aa); err != nil {
		return false
	}
	if err := json.Unmarshal(b, &bb); err != nil {
		return false
	}
	ja, _ := json.Marshal(aa)
	jb, _ := json.Marshal(bb)
	return string(ja) == string(jb)
}
