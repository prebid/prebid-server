package tmp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/openrtb/v20/openrtb2"
)

// tmpFixture spins up an in-memory property registry and a fake TMP provider
// that answers both /context and /identity, and returns a Module wired to them.
// Callers customize the handlers via the returned pointers.
type tmpFixture struct {
	Module         *Module
	Registry       *httptest.Server
	Provider       *httptest.Server
	ContextHandler http.HandlerFunc
	IdentHandler   http.HandlerFunc
}

func newFixture(t *testing.T) *tmpFixture {
	t.Helper()
	f := &tmpFixture{}
	f.Registry = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("domain")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"property": map[string]any{
				"property_rid":  "01916f3a-1234-7000-8000-000000000001",
				"property_id":   "fixture",
				"property_type": "website",
				"domain":        domain,
			},
		})
	}))
	f.Provider = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/context":
			if f.ContextHandler != nil {
				f.ContextHandler(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(tmproto.ContextMatchResponse{
				Type:      "context_match_response",
				RequestID: "req",
				Offers:    []tmproto.Offer{{PackageID: "pkg-a"}, {PackageID: "pkg-b"}},
				Signals:   map[string]any{"segment": "auto_intender"},
			})
		case "/identity":
			if f.IdentHandler != nil {
				f.IdentHandler(w, r)
				return
			}
			_ = json.NewEncoder(w).Encode(tmproto.IdentityMatchResponse{
				Type:               "identity_match_response",
				RequestID:          "req",
				EligiblePackageIDs: []string{"pkg-a"},
				ServeWindowSec:     60,
			})
		default:
			http.NotFound(w, r)
		}
	}))

	cfg := Config{
		SellerAgentURL: "https://seller.example.com",
		Signing: SigningConfig{
			KeyID:         "kid-1",
			PrivateKeyPEM: genTestKey(t),
		},
		PropertyRegistry: PropertyRegistryConfig{Endpoint: f.Registry.URL},
		Providers: []ProviderConfig{{
			Name:        "prov",
			IdentityURL: f.Provider.URL + "/identity",
			ContextURL:  f.Provider.URL + "/context",
		}},
	}
	priv, err := cfg.validated()
	if err != nil {
		t.Fatalf("validated: %v", err)
	}
	signer, err := tmproto.NewSigner(cfg.Signing.KeyID, priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	f.Module = &Module{
		cfg:      cfg,
		signer:   signer,
		http:     http.DefaultClient,
		registry: newPropertyResolver(cfg.PropertyRegistry, nil),
	}
	return f
}

func (f *tmpFixture) Close() {
	f.Registry.Close()
	f.Provider.Close()
}

func sampleBidRequest() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "publisher.example", Page: "https://publisher.example/story"},
		Imp:  []openrtb2.Imp{{TagID: "slot-1"}},
		User: &openrtb2.User{
			EIDs: []openrtb2.EID{
				{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "ramp-x"}}},
			},
		},
		Device: &openrtb2.Device{
			Geo: &openrtb2.Geo{Country: "US"},
		},
	}
}

func TestFanOut_JoinsContextAndIdentity(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	res := f.Module.fanOut(context.Background(), sampleBidRequest())
	if res == nil || len(res.Segments) == 0 {
		t.Fatalf("expected segments, got %+v", res)
	}
	// pkg-b should be filtered out because identity only returned pkg-a.
	sawPkgA := false
	sawPkgB := false
	for _, s := range res.Segments {
		if s == "prov_package=pkg-a" {
			sawPkgA = true
		}
		if s == "prov_package=pkg-b" {
			sawPkgB = true
		}
	}
	if !sawPkgA {
		t.Errorf("expected prov_package=pkg-a in segments; got %v", res.Segments)
	}
	if sawPkgB {
		t.Errorf("pkg-b should have been filtered by identity eligibility; got %v", res.Segments)
	}
}

func TestFanOut_ContextOnlyWhenNoIdentityTokens(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	req := sampleBidRequest()
	req.User = nil // no eids

	res := f.Module.fanOut(context.Background(), req)
	if res == nil || len(res.Segments) == 0 {
		t.Fatalf("expected segments even without identity; got %+v", res)
	}
	// Both packages should be present because identity eligibility is not enforced.
	sawA, sawB := false, false
	for _, s := range res.Segments {
		if s == "prov_package=pkg-a" {
			sawA = true
		}
		if s == "prov_package=pkg-b" {
			sawB = true
		}
	}
	if !sawA || !sawB {
		t.Errorf("expected both packages without identity; got %v", res.Segments)
	}
}

func TestFanOut_UnknownDomainReturnsEmpty(t *testing.T) {
	f := newFixture(t)
	defer f.Close()
	// Replace registry with a 404-only server.
	f.Registry.Close()
	f.Registry = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	f.Module.registry = newPropertyResolver(PropertyRegistryConfig{
		Endpoint:                f.Registry.URL,
		NegativeCacheTTLSeconds: 60,
		CacheSize:               4,
		TimeoutMs:               500,
	}, nil)

	res := f.Module.fanOut(context.Background(), sampleBidRequest())
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if len(res.Segments) != 0 {
		t.Errorf("expected empty segments for unknown domain; got %v", res.Segments)
	}
}

func TestFanOut_EmptyPlacementIDShortCircuits(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	req := sampleBidRequest()
	req.Imp = nil

	res := f.Module.fanOut(context.Background(), req)
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if len(res.Segments) != 0 {
		t.Errorf("expected empty segments without a placement id; got %v", res.Segments)
	}
}

func TestFanOut_ProviderPanicRecovered(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	// Make the provider hang up mid-response so JSON decode panics on some
	// corrupt payload — but more simply, close the connection.
	f.ContextHandler = func(w http.ResponseWriter, r *http.Request) {
		panic("simulated context handler panic")
	}
	f.IdentHandler = func(w http.ResponseWriter, r *http.Request) {
		panic("simulated identity handler panic")
	}

	// httptest recovers server-side panics, so this only exercises client-side
	// recovery when the response body is malformed. We swap in a handler that
	// returns garbage JSON that decodes to an empty struct, and verify the
	// module keeps returning a non-nil routerResult (i.e. no crash).
	f.ContextHandler = func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}
	f.IdentHandler = func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}

	res := f.Module.fanOut(context.Background(), sampleBidRequest())
	if res == nil {
		t.Fatal("expected non-nil result even when both provider calls error")
	}
}

func TestFanOut_RandomizesContextIdentityOrder(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	var mu sync.Mutex
	seen := map[string]int{} // "context-first" / "identity-first"
	// Track which endpoint each request hit; whichever handler fires first
	// per iteration determines the order for that iteration.
	var currentIteration string
	setFirst := func(kind string) {
		mu.Lock()
		defer mu.Unlock()
		if currentIteration == "" {
			currentIteration = kind
		}
	}
	f.ContextHandler = func(w http.ResponseWriter, r *http.Request) {
		setFirst("context")
		_ = json.NewEncoder(w).Encode(tmproto.ContextMatchResponse{Type: "context_match_response", Offers: []tmproto.Offer{{PackageID: "pkg"}}})
	}
	f.IdentHandler = func(w http.ResponseWriter, r *http.Request) {
		setFirst("identity")
		_ = json.NewEncoder(w).Encode(tmproto.IdentityMatchResponse{Type: "identity_match_response", EligiblePackageIDs: []string{"pkg"}})
	}

	const iterations = 200
	for range iterations {
		mu.Lock()
		currentIteration = ""
		mu.Unlock()
		_ = f.Module.fanOut(context.Background(), sampleBidRequest())
		mu.Lock()
		if currentIteration != "" {
			seen[currentIteration+"-first"]++
		}
		mu.Unlock()
	}

	// Both orderings must appear at least once across 200 iterations. The
	// probability of a single-ordering run is (1/2)^200 — effectively zero.
	if seen["context-first"] == 0 {
		t.Errorf("context never fired first across %d iterations; ordering is not randomized", iterations)
	}
	if seen["identity-first"] == 0 {
		t.Errorf("identity never fired first across %d iterations; ordering is not randomized", iterations)
	}
}

func TestFanOut_DecorrelationDelayDisabledByDefault(t *testing.T) {
	f := newFixture(t)
	defer f.Close()
	if f.Module.cfg.DecorrelationMaxDelayMs != 0 {
		t.Errorf("expected default DecorrelationMaxDelayMs = 0 (off), got %d", f.Module.cfg.DecorrelationMaxDelayMs)
	}

	start := time.Now()
	_ = f.Module.fanOut(context.Background(), sampleBidRequest())
	elapsed := time.Since(start)
	// With the delay off, a healthy in-process fixture should complete well
	// under 100 ms. A generous bound catches regressions without being flaky.
	if elapsed > 100*time.Millisecond {
		t.Errorf("fan-out took %v with decorrelation disabled; want < 100ms", elapsed)
	}
}

func TestFanOut_SigningHeadersOnOutbound(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	var sawSig, sawKid string
	f.ContextHandler = func(w http.ResponseWriter, r *http.Request) {
		sawSig = r.Header.Get(tmproto.HeaderTMPSignature)
		sawKid = r.Header.Get(tmproto.HeaderTMPKeyID)
		_ = json.NewEncoder(w).Encode(tmproto.ContextMatchResponse{Type: "context_match_response", Offers: []tmproto.Offer{{PackageID: "pkg"}}})
	}
	f.IdentHandler = func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(tmproto.IdentityMatchResponse{Type: "identity_match_response", EligiblePackageIDs: []string{"pkg"}})
	}

	_ = f.Module.fanOut(context.Background(), sampleBidRequest())
	if sawSig == "" {
		t.Error("expected X-AdCP-Signature to be set on outbound context call")
	}
	if sawKid != "kid-1" {
		t.Errorf("X-AdCP-Key-Id = %q, want kid-1", sawKid)
	}
}
