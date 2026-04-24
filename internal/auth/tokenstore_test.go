package auth

import (
	"testing"
	"time"
)

func TestTokenStoreRoundTrip(t *testing.T) {
	t.Setenv("HARAVAN_CLI_HOME", t.TempDir())
	store, err := NewTokenStore()
	if err != nil {
		t.Fatalf("NewTokenStore: %v", err)
	}

	tok := StoredToken{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		AppID:        "app-1",
		Scope:        []string{"com.read_orders"},
		ExpiresAt:    time.Now().Add(time.Hour).UnixMilli(),
		CreatedAt:    time.Now().UnixMilli(),
	}

	if err := store.Save("app-1", tok); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok, err := store.Load("app-1")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !ok {
		t.Fatal("expected token to be present")
	}
	if got.AccessToken != "access-123" {
		t.Errorf("AccessToken: %q", got.AccessToken)
	}

	// Overwrite + second app
	if err := store.Save("app-2", StoredToken{AccessToken: "second", CreatedAt: 1}); err != nil {
		t.Fatalf("Save second: %v", err)
	}
	all, err := store.LoadAll()
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 tokens, got %d", len(all))
	}

	if err := store.Delete("app-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, ok, _ := store.Load("app-1"); ok {
		t.Error("app-1 should be gone after Delete")
	}
	if _, ok, _ := store.Load("app-2"); !ok {
		t.Error("app-2 should remain after deleting app-1")
	}
}

func TestStoredTokenIsExpired(t *testing.T) {
	now := time.Now().UnixMilli()
	cases := []struct {
		name string
		tok  StoredToken
		want bool
	}{
		{"no-expiry", StoredToken{}, false},
		{"future", StoredToken{ExpiresAt: now + 60_000}, false},
		{"past", StoredToken{ExpiresAt: now - 60_000}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.tok.IsExpired(); got != tc.want {
				t.Errorf("IsExpired: got %v, want %v", got, tc.want)
			}
		})
	}
}
