package tmp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

	res := f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
	if res == nil || len(res.Segments) == 0 {
		t.Fatalf("expected segments, got %+v", res)
	}
	// pkg-b should be filtered out because identity only returned pkg-a.
	// Package IDs land as a single comma-joined entry under the configured
	// PackageTargetingKey (default adcp_package_id).
	var pkgSeg string
	for _, s := range res.Segments {
		if strings.HasPrefix(s, "adcp_package_id=") {
			pkgSeg = s
			break
		}
	}
	if pkgSeg == "" {
		t.Errorf("expected adcp_package_id=... in segments; got %v", res.Segments)
	}
	if !strings.Contains(pkgSeg, "pkg-a") {
		t.Errorf("expected pkg-a in %q; got %v", pkgSeg, res.Segments)
	}
	if strings.Contains(pkgSeg, "pkg-b") {
		t.Errorf("pkg-b should have been filtered by identity eligibility; got %v", res.Segments)
	}
}

func TestFanOut_ContextOnlyWhenNoIdentityTokens(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	req := sampleBidRequest()
	req.User = nil // no eids

	res := f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, req))
	if res == nil || len(res.Segments) == 0 {
		t.Fatalf("expected segments even without identity; got %+v", res)
	}
	// Both packages should be present because identity eligibility is not enforced.
	var pkgSeg string
	for _, s := range res.Segments {
		if strings.HasPrefix(s, "adcp_package_id=") {
			pkgSeg = s
			break
		}
	}
	if !strings.Contains(pkgSeg, "pkg-a") || !strings.Contains(pkgSeg, "pkg-b") {
		t.Errorf("expected both pkg-a and pkg-b in %q; got %v", pkgSeg, res.Segments)
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

	res := f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
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

	res := f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, req))
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	if len(res.Segments) != 0 {
		t.Errorf("expected empty segments without a placement id; got %v", res.Segments)
	}
}

// Provider decode error must surface as fan-out completing with empty
// segments — not a crash, not a hang. This is the low-level "error
// tolerance" test; genuine panic-recovery is exercised by
// TestFanOut_PanickingRoundTripper below.
func TestFanOut_ProviderDecodeErrorSurvives(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	f.ContextHandler = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}
	f.IdentHandler = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}

	res := f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
	if res == nil {
		t.Fatal("expected non-nil result even when both provider calls error")
	}
	if res.ErrCount != 1 {
		t.Errorf("expected 1 provider with errors; got %d", res.ErrCount)
	}
	if len(res.Segments) != 0 {
		t.Errorf("no segments expected on total decode failure; got %v", res.Segments)
	}
}

// panickingRoundTripper panics inside RoundTrip. The fan-out's inner
// goroutine must recover, record the error, and let the sibling call
// complete instead of taking the process down.
type panickingRoundTripper struct{}

func (panickingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	panic("boom in RoundTrip")
}

func TestFanOut_PanickingRoundTripper(t *testing.T) {
	f := newFixture(t)
	defer f.Close()
	// Only the context path panics; identity still works via the fixture's
	// default handler. If panic recovery is broken this either crashes
	// the test process or leaves the fan-out wedged forever.
	f.Module.http = &http.Client{Transport: panickingRoundTripper{}}

	done := make(chan *routerResult, 1)
	go func() {
		done <- f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
	}()

	select {
	case res := <-done:
		if res == nil {
			t.Fatal("expected non-nil result after panic recovery")
		}
		if res.ErrCount != 1 {
			t.Errorf("expected 1 provider with errors; got %d", res.ErrCount)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("fan-out did not complete after transport panic — recovery is broken")
	}
}

// Fail-closed: identity call errors → offers dropped, not emitted
// unfiltered. Confirms the fix for review finding #3.
func TestFanOut_IdentityErrorDropsOffers(t *testing.T) {
	f := newFixture(t)
	defer f.Close()

	f.IdentHandler = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	// Context returns real offers.
	f.ContextHandler = func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(tmproto.ContextMatchResponse{
			Type:      "context_match_response",
			RequestID: "req",
			Offers:    []tmproto.Offer{{PackageID: "pkg-a"}, {PackageID: "pkg-b"}},
		})
	}

	res := f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
	if res == nil {
		t.Fatal("expected non-nil result")
	}
	for _, s := range res.Segments {
		if strings.HasPrefix(s, "adcp_package_id=") {
			t.Errorf("expected no package segments when identity call errored; got %q", s)
		}
	}
}

// Randomization is only observable if the second-to-spawn call is
// noticeably delayed vs the first — otherwise HTTP arrival order at the
// fake server is scheduler noise, not evidence of a shuffle. Set a
// large DecorrelationMaxDelayMs so the second call deterministically
// sleeps up to N ms before its HTTP round-trip; assert both orderings
// appear across enough iterations that a broken shuffle fails loudly.
func TestFanOut_RandomizesContextIdentityOrder(t *testing.T) {
	f := newFixture(t)
	defer f.Close()
	f.Module.cfg.DecorrelationMaxDelayMs = 30

	var mu sync.Mutex
	seen := map[string]int{}
	var currentIteration string
	setFirst := func(kind string) {
		mu.Lock()
		defer mu.Unlock()
		if currentIteration == "" {
			currentIteration = kind
		}
	}
	f.ContextHandler = func(w http.ResponseWriter, _ *http.Request) {
		setFirst("context")
		_ = json.NewEncoder(w).Encode(tmproto.ContextMatchResponse{Type: "context_match_response", Offers: []tmproto.Offer{{PackageID: "pkg"}}})
	}
	f.IdentHandler = func(w http.ResponseWriter, _ *http.Request) {
		setFirst("identity")
		_ = json.NewEncoder(w).Encode(tmproto.IdentityMatchResponse{Type: "identity_match_response", EligiblePackageIDs: []string{"pkg"}})
	}

	const iterations = 40
	for range iterations {
		mu.Lock()
		currentIteration = ""
		mu.Unlock()
		_ = f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
		mu.Lock()
		if currentIteration != "" {
			seen[currentIteration+"-first"]++
		}
		mu.Unlock()
	}

	if seen["context-first"] == 0 {
		t.Errorf("context never fired first across %d iterations; shuffle is not randomizing order", iterations)
	}
	if seen["identity-first"] == 0 {
		t.Errorf("identity never fired first across %d iterations; shuffle is not randomizing order", iterations)
	}
}

func TestFanOut_DecorrelationDelayDisabledByDefault(t *testing.T) {
	f := newFixture(t)
	defer f.Close()
	if f.Module.cfg.DecorrelationMaxDelayMs != 0 {
		t.Errorf("expected default DecorrelationMaxDelayMs = 0 (off), got %d", f.Module.cfg.DecorrelationMaxDelayMs)
	}

	start := time.Now()
	_ = f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
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

	_ = f.Module.fanOut(context.Background(), deriveInputs(&f.Module.cfg, sampleBidRequest()))
	if sawSig == "" {
		t.Error("expected X-AdCP-Signature to be set on outbound context call")
	}
	if sawKid != "kid-1" {
		t.Errorf("X-AdCP-Key-Id = %q, want kid-1", sawKid)
	}
}

// TestMergeSegments_TMPXAndOfferMacros verifies the four spec surfaces the
// module surfaces onto prebid targeting: package IDs (comma-joined under
// the configurable single key), per-offer creative macros, response-level
// context signals, and identity TMPX macros — each with its raw key intact,
// no provider-name prefix.
func TestMergeSegments_TMPXAndOfferMacros(t *testing.T) {
	m := &Module{cfg: Config{
		PackageTargetingKey: "adcp_package_id",
		MaxSegments:         64,
		MaxSegmentValueLen:  256,
	}}
	results := []providerResult{{
		Name: "prov",
		Context: &tmproto.ContextMatchResponse{
			Offers: []tmproto.Offer{
				{PackageID: "pkg-a", Macros: map[string]string{"brand": "Acme"}},
				{PackageID: "pkg-b"},
			},
			Signals: map[string]any{"iab_cat": "sports"},
		},
		Identity: &tmproto.IdentityMatchResponse{
			EligiblePackageIDs: []string{"pkg-a", "pkg-b"},
			TmpxMacros: []tmproto.TmpxMacro{
				{Name: "SCOPE3_TMPX_1", Value: "opaque-chunk-1"},
			},
		},
	}}

	out := m.mergeSegments(results)
	want := map[string]string{
		"brand":           "Acme",
		"iab_cat":         "sports",
		"SCOPE3_TMPX_1":   "opaque-chunk-1",
		"adcp_package_id": "pkg-a,pkg-b",
	}
	for _, s := range out {
		kv := strings.SplitN(s, "=", 2)
		if len(kv) != 2 {
			t.Errorf("malformed segment %q", s)
			continue
		}
		if want[kv[0]] != kv[1] {
			t.Errorf("segment %q: want %q for key %q", s, want[kv[0]], kv[0])
		}
		delete(want, kv[0])
	}
	if len(want) != 0 {
		t.Errorf("missing expected keys: %v (got %v)", want, out)
	}
}

// TestMergeSegments_PackageIDsDedupedAcrossProviders confirms two providers
// returning the same PackageID collapse into a single value in the joined
// key — otherwise GAM's IN-targeting list would carry duplicates.
func TestMergeSegments_PackageIDsDedupedAcrossProviders(t *testing.T) {
	m := &Module{cfg: Config{
		PackageTargetingKey: "adcp_package_id",
		MaxSegments:         64,
		MaxSegmentValueLen:  256,
	}}
	results := []providerResult{
		{Name: "a", Context: &tmproto.ContextMatchResponse{Offers: []tmproto.Offer{{PackageID: "pkg-1"}, {PackageID: "pkg-2"}}}},
		{Name: "b", Context: &tmproto.ContextMatchResponse{Offers: []tmproto.Offer{{PackageID: "pkg-2"}, {PackageID: "pkg-3"}}}},
	}

	out := m.mergeSegments(results)
	var pkgSeg string
	for _, s := range out {
		if strings.HasPrefix(s, "adcp_package_id=") {
			pkgSeg = s
		}
	}
	if pkgSeg != "adcp_package_id=pkg-1,pkg-2,pkg-3" {
		t.Errorf("expected pkg-1,pkg-2,pkg-3 (deduped, first-seen order); got %q", pkgSeg)
	}
}

// TestMergeSegments_EmptyPackageKeyDisables verifies that setting
// PackageTargetingKey to "" omits the package line entirely — the escape
// hatch for operators whose ad server doesn't want it.
func TestMergeSegments_EmptyPackageKeyDisables(t *testing.T) {
	m := &Module{cfg: Config{
		PackageTargetingKey: "",
		MaxSegments:         64,
		MaxSegmentValueLen:  256,
	}}
	out := m.mergeSegments([]providerResult{{
		Name:    "prov",
		Context: &tmproto.ContextMatchResponse{Offers: []tmproto.Offer{{PackageID: "pkg-a"}}},
	}})
	for _, s := range out {
		if strings.HasPrefix(s, "adcp_package_id=") || strings.Contains(s, "package") {
			t.Errorf("expected no package segment when PackageTargetingKey is empty; got %q", s)
		}
	}
}

// TestMergeSegments_FailClosedDropsTMPX confirms the fail-closed path also
// suppresses TMPX macros. If the identity call errored the module has no
// way to know if the token is authorized for the request, so the safe
// answer is to drop it — otherwise a flaky identity endpoint could leak
// a token onto an impression the eligibility gate would have blocked.
func TestMergeSegments_FailClosedDropsTMPX(t *testing.T) {
	m := &Module{cfg: Config{
		PackageTargetingKey: "adcp_package_id",
		MaxSegments:         64,
		MaxSegmentValueLen:  256,
	}}
	results := []providerResult{{
		Name:              "prov",
		Context:           &tmproto.ContextMatchResponse{Offers: []tmproto.Offer{{PackageID: "pkg-a"}}},
		IdentityAttempted: true,
		Identity:          nil,
	}}
	out := m.mergeSegments(results)
	if len(out) != 0 {
		t.Errorf("expected empty segments on identity-attempted-but-failed; got %v", out)
	}
}
