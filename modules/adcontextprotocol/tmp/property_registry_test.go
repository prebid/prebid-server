package tmp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestPropertyResolver_Cache(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		domain := r.URL.Query().Get("domain")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"property": map[string]any{
				"property_rid":  "01916f3a-1234-7000-8000-000000000001",
				"property_id":   "example",
				"property_type": "website",
				"domain":        domain,
			},
		})
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:        srv.URL,
		CacheTTLSeconds: 60,
		CacheSize:       16,
		TimeoutMs:       500,
	}, nil)

	ctx := context.Background()
	rec1, ok, err := r.Resolve(ctx, "example.com")
	if err != nil || !ok || rec1.PropertyRID == "" {
		t.Fatalf("first resolve: rec=%+v ok=%v err=%v", rec1, ok, err)
	}
	rec2, ok, err := r.Resolve(ctx, "example.com")
	if err != nil || !ok || rec2.PropertyRID != rec1.PropertyRID {
		t.Fatalf("second resolve did not hit cache: rec=%+v ok=%v err=%v", rec2, ok, err)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 upstream call, got %d", got)
	}
}

func TestPropertyResolver_NotFound_NegativelyCached(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:                srv.URL,
		CacheTTLSeconds:         60,
		NegativeCacheTTLSeconds: 60,
		CacheSize:               16,
		TimeoutMs:               500,
	}, nil)

	ctx := context.Background()
	for i := range 3 {
		rec, ok, err := r.Resolve(ctx, "nowhere.example")
		if err != nil {
			t.Fatalf("resolve[%d]: %v", i, err)
		}
		if ok || rec != nil {
			t.Fatalf("resolve[%d]: expected not-found, got rec=%+v ok=%v", i, rec, ok)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("expected 1 upstream call (rest served from negative cache), got %d", got)
	}
}

func TestPropertyResolver_UpstreamError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:  srv.URL,
		CacheSize: 4,
		TimeoutMs: 500,
	}, nil)
	_, _, err := r.Resolve(context.Background(), "x.example")
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestPropertyResolver_BearerAuth(t *testing.T) {
	var sawAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:                srv.URL,
		AuthBearer:              "secret-token",
		NegativeCacheTTLSeconds: 60,
		CacheSize:               4,
		TimeoutMs:               500,
	}, nil)
	_, _, _ = r.Resolve(context.Background(), "x.example")
	if sawAuth != "Bearer secret-token" {
		t.Errorf("Authorization header = %q, want %q", sawAuth, "Bearer secret-token")
	}
}

// Trigger LRU eviction to make sure the cache does not grow unbounded.
func TestPropertyResolver_LRUEviction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"property": map[string]any{
				"property_rid":  "rid-" + domain,
				"property_id":   domain,
				"property_type": "website",
				"domain":        domain,
			},
		})
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:        srv.URL,
		CacheTTLSeconds: 60,
		CacheSize:       2,
		TimeoutMs:       500,
	}, nil)

	ctx := context.Background()
	for i := range 5 {
		if _, _, err := r.Resolve(ctx, fmt.Sprintf("d%d.example", i)); err != nil {
			t.Fatalf("resolve[%d]: %v", i, err)
		}
	}
	if r.order.Len() > 2 {
		t.Errorf("cache size = %d, want <= 2", r.order.Len())
	}
}
