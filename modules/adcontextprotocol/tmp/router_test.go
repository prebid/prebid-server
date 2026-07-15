package tmp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
