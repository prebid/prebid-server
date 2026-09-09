package tmp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

// resolvedShape emits an adcp ResolveResponse envelope. Prod uses this
// exact shape (see adcp catalog-openapi.ts / server registry-api.ts).
func resolvedShape(propertyRID, classification, domain string) map[string]any {
	return map[string]any{
		"resolved": []map[string]any{{
			"identifier":     map[string]any{"type": "domain", "value": domain},
			"property_rid":   propertyRID,
			"classification": classification,
			"status":         "existing",
			"source":         "authoritative",
		}},
		"summary":          map[string]any{"total": 1, "resolved": 1},
		"server_timestamp": "2026-01-01T00:00:00Z",
	}
}

// unresolvedShape emits a null-property_rid entry — the shape the
// registry uses for excluded (ad_infra / publisher_mask) identifiers
// and for lookups that hit nothing. Both cache as negative in the
// module.
func unresolvedShape(domain string) map[string]any {
	return map[string]any{
		"resolved": []map[string]any{{
			"identifier":     map[string]any{"type": "domain", "value": domain},
			"property_rid":   nil,
			"classification": "unknown",
			"status":         "existing",
			"source":         nil,
		}},
		"summary":          map[string]any{"total": 1, "resolved": 0, "not_found": 1},
		"server_timestamp": "2026-01-01T00:00:00Z",
	}
}

// decodeResolveRequest returns the domain value the caller sent, so
// tests can echo it back in the response envelope.
func decodeResolveRequest(t *testing.T, r *http.Request) string {
	t.Helper()
	if r.Method != http.MethodPost {
		t.Errorf("expected POST, got %s", r.Method)
	}
	var body registryResolveRequest
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if len(body.Identifiers) != 1 {
		t.Fatalf("expected 1 identifier, got %d", len(body.Identifiers))
	}
	return body.Identifiers[0].Value
}

func TestPropertyResolver_Cache(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		domain := decodeResolveRequest(t, r)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resolvedShape("01916f3a-1234-7000-8000-000000000001", "website", domain))
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:        srv.URL,
		Mode:            "lookup",
		ProvenanceType:  "member_assertion",
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

func TestPropertyResolver_NullRID_NegativelyCached(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		domain := decodeResolveRequest(t, r)
		_ = json.NewEncoder(w).Encode(unresolvedShape(domain))
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:                srv.URL,
		Mode:                    "lookup",
		ProvenanceType:          "member_assertion",
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

func TestPropertyResolver_NotFound_NegativelyCached(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:                srv.URL,
		Mode:                    "lookup",
		ProvenanceType:          "member_assertion",
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
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:       srv.URL,
		Mode:           "lookup",
		ProvenanceType: "member_assertion",
		CacheSize:      4,
		TimeoutMs:      500,
	}, nil)
	_, _, err := r.Resolve(context.Background(), "x.example")
	if err == nil {
		t.Fatal("expected error on 500")
	}
}

func TestPropertyResolver_BearerAuth(t *testing.T) {
	var sawAuth, sawContentType string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		sawContentType = r.Header.Get("Content-Type")
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:                srv.URL,
		Mode:                    "resolve",
		ProvenanceType:          "member_assertion",
		AuthBearer:              "secret-token",
		NegativeCacheTTLSeconds: 60,
		CacheSize:               4,
		TimeoutMs:               500,
	}, nil)
	_, _, _ = r.Resolve(context.Background(), "x.example")
	if sawAuth != "Bearer secret-token" {
		t.Errorf("Authorization header = %q, want %q", sawAuth, "Bearer secret-token")
	}
	if sawContentType != "application/json" {
		t.Errorf("Content-Type header = %q, want application/json", sawContentType)
	}
}

// TestPropertyResolver_ProvenanceForwarded verifies the caller-configured
// provenance envelope reaches the upstream unchanged. The registry keys
// trust decisions off provenance.type, so a bug here would silently
// misattribute every resolve.
func TestPropertyResolver_ProvenanceForwarded(t *testing.T) {
	var seen registryProvenance
	var seenMode string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body registryResolveRequest
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &body)
		seen = body.Provenance
		seenMode = body.Mode
		_ = json.NewEncoder(w).Encode(resolvedShape("01916f3a-1234-7000-8000-000000000002", "website", "x.example"))
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:          srv.URL,
		Mode:              "lookup",
		ProvenanceType:    "publisher_declaration",
		ProvenanceContext: "prebid-integration",
		CacheSize:         4,
		TimeoutMs:         500,
	}, nil)
	_, _, err := r.Resolve(context.Background(), "x.example")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if seen.Type != "publisher_declaration" || seen.Context != "prebid-integration" {
		t.Errorf("provenance forwarded incorrectly: got %+v", seen)
	}
	if seenMode != "lookup" {
		t.Errorf("mode forwarded incorrectly: got %q", seenMode)
	}
}

// Trigger LRU eviction to make sure the cache does not grow unbounded.
func TestPropertyResolver_LRUEviction(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := decodeResolveRequest(t, r)
		_ = json.NewEncoder(w).Encode(resolvedShape("rid-"+domain, "website", domain))
	}))
	defer srv.Close()

	r := newPropertyResolver(PropertyRegistryConfig{
		Endpoint:        srv.URL,
		Mode:            "lookup",
		ProvenanceType:  "member_assertion",
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
