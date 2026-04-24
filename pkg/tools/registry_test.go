package tools

import (
	"context"
	"testing"
)

func stubHandler(_ context.Context, _ *Deps, _ Input) (*Result, error) { return nil, nil }

func TestRegistryRegisterAndLookup(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{Name: "a", Category: CatShop, Handler: stubHandler})
	r.Register(&Tool{Name: "b", Category: CatOrders, Handler: stubHandler})

	if got := r.Count(); got != 2 {
		t.Errorf("Count: got %d, want 2", got)
	}
	if r.Get("a") == nil || r.Get("b") == nil {
		t.Error("Get failed for registered tools")
	}
	if r.Get("missing") != nil {
		t.Error("Get returned non-nil for missing tool")
	}
	all := r.All()
	if len(all) != 2 || all[0].Name != "a" || all[1].Name != "b" {
		t.Errorf("All: %+v", all)
	}
	shop := r.ByCategory(CatShop)
	if len(shop) != 1 || shop[0].Name != "a" {
		t.Errorf("ByCategory shop: %+v", shop)
	}
	if cats := r.Categories(); len(cats) != 2 {
		t.Errorf("Categories: %+v", cats)
	}
}

func TestRegistryPanicsOnDuplicate(t *testing.T) {
	r := NewRegistry()
	r.Register(&Tool{Name: "x", Handler: stubHandler})
	defer func() {
		if recover() == nil {
			t.Error("expected panic on duplicate Name")
		}
	}()
	r.Register(&Tool{Name: "x", Handler: stubHandler})
}

func TestRegistryPanicsOnNilHandler(t *testing.T) {
	r := NewRegistry()
	defer func() {
		if recover() == nil {
			t.Error("expected panic on nil Handler")
		}
	}()
	r.Register(&Tool{Name: "y"})
}

func TestDefaultRegistryPopulated(t *testing.T) {
	// The default registry is populated by tool packages at init time.
	// We don't import them here (to avoid coupling), so this test just
	// asserts the type compiles and is non-nil. Smoke test only.
	if Default() == nil {
		t.Fatal("Default() returned nil")
	}
}
