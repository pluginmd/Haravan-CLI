package client

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

// page handler returns a page of fake orders; last page is short to signal end.
func pagedServer(totalPages, pageSize int, resourceKey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		page, _ := strconv.Atoi(q.Get("page"))
		if page == 0 {
			page = 1
		}
		count := pageSize
		if page == totalPages {
			count = pageSize / 2 // short page — triggers loop termination
		}
		if page > totalPages {
			count = 0
		}
		fmt.Fprintf(w, `{"%s":[`, resourceKey)
		for i := 0; i < count; i++ {
			if i > 0 {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"id":%d}`, (page-1)*pageSize+i)
		}
		fmt.Fprint(w, `]}`)
	}
}

func TestPaginateAllWalksUntilShortPage(t *testing.T) {
	srv := httptest.NewServer(pagedServer(3, 50, "orders"))
	defer srv.Close()
	c, err := New(Options{BaseURL: srv.URL, AccessToken: "x"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	p, err := PaginateAll(context.Background(), c, "/com/orders.json", url.Values{"limit": {"50"}}, "orders")
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}
	// 50 + 50 + 25 = 125
	if n := len(p.Items); n != 125 {
		t.Errorf("items: got %d, want 125", n)
	}
	if p.Pages != 3 {
		t.Errorf("pages: got %d, want 3", p.Pages)
	}
	if p.Truncated {
		t.Error("should not be truncated")
	}
	if p.ResourceKey != "orders" {
		t.Errorf("resourceKey: %q", p.ResourceKey)
	}
}

func TestPaginateAllAutoDetectsResourceKey(t *testing.T) {
	srv := httptest.NewServer(pagedServer(1, 3, "products"))
	defer srv.Close()
	c, _ := New(Options{BaseURL: srv.URL, AccessToken: "x"})
	p, err := PaginateAll(context.Background(), c, "/com/products.json", nil, "")
	if err != nil {
		t.Fatalf("PaginateAll: %v", err)
	}
	if p.ResourceKey != "products" {
		t.Errorf("resourceKey: %q", p.ResourceKey)
	}
}
