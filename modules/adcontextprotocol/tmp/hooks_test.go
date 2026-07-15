package tmp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/adcontextprotocol/adcp-go/tmproto"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// This file's tests exercise the module through the hookstage API
// exactly as the framework does, so regressions like "the fan-out
// context is Done before the goroutine runs" cannot slip through by
// only testing fanOut directly (which is what the initial PR did).
//
// The critical assertion: after HandleProcessedAuctionHook returns and
// HandleAuctionResponseHook completes, the bid response ext MUST carry
// the merged segments. If the fan-out ctx was derived from the
// entrypoint hook's ctx (framework cancels it on hook return), no
// segments land — this test catches that class of bug.

func newHooksFixtureModule(t *testing.T) (*Module, func()) {
	t.Helper()
	registry := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"property": map[string]any{
				"property_rid":  "01916f3a-1234-7000-8000-000000000001",
				"property_id":   "fixture",
				"property_type": "website",
				"domain":        r.URL.Query().Get("domain"),
			},
		})
	}))
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/context":
			_ = json.NewEncoder(w).Encode(tmproto.ContextMatchResponse{
				Type:      "context_match_response",
				RequestID: "req",
				Offers:    []tmproto.Offer{{PackageID: "pkg-a"}},
			})
		case "/identity":
			_ = json.NewEncoder(w).Encode(tmproto.IdentityMatchResponse{
				Type:               "identity_match_response",
				RequestID:          "req",
				EligiblePackageIDs: []string{"pkg-a"},
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
		PropertyRegistry: PropertyRegistryConfig{Endpoint: registry.URL},
		Providers: []ProviderConfig{{
			Name:        "prov",
			IdentityURL: provider.URL + "/identity",
			ContextURL:  provider.URL + "/context",
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
	m := &Module{
		cfg:      cfg,
		signer:   signer,
		http:     http.DefaultClient,
		registry: newPropertyResolver(cfg.PropertyRegistry, nil),
	}
	return m, func() {
		registry.Close()
		provider.Close()
	}
}

func TestHooks_EndToEndDeliversSegments(t *testing.T) {
	m, cleanup := newHooksFixtureModule(t)
	defer cleanup()

	// Simulate the framework's per-hook cancellation: pass each hook a
	// context that gets cancelled the moment the hook returns. If the
	// module were rooting its fan-out in this context (as it did before
	// this fix), the goroutine would fire with an already-cancelled ctx
	// and the segments would never land.
	runHook := func(fn func(ctx context.Context)) {
		hookCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		fn(hookCtx)
	}

	bidReq := &openrtb2.BidRequest{
		ID:   "auction-1",
		Site: &openrtb2.Site{Domain: "publisher.example"},
		Imp:  []openrtb2.Imp{{ID: "imp-1", TagID: "slot-1"}},
		User: &openrtb2.User{
			EIDs: []openrtb2.EID{{Source: "liveramp.com", UIDs: []openrtb2.UID{{ID: "ramp-x"}}}},
		},
	}
	wrapper := &openrtb_ext.RequestWrapper{BidRequest: bidReq}

	// Stage 1: HandleProcessedAuctionHook — snapshots inputs, spawns
	// fan-out on a Background-rooted ctx, returns the module context.
	var processedRes hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]
	runHook(func(ctx context.Context) {
		var err error
		processedRes, err = m.HandleProcessedAuctionHook(ctx, hookstage.ModuleInvocationContext{}, hookstage.ProcessedAuctionRequestPayload{Request: wrapper})
		if err != nil {
			t.Fatalf("processed hook: %v", err)
		}
	})
	if processedRes.ModuleContext == nil {
		t.Fatal("expected processed hook to set ModuleContext with the async holder")
	}
	// The hook ctx passed above is now cancelled. If the module rooted
	// its fan-out ctx in it, the fan-out is dead. The response hook is
	// where we discover that.

	// Stage 2: HandleAuctionResponseHook — waits on the fan-out and
	// mutates the response ext.
	bidResp := &openrtb2.BidResponse{ID: "auction-1", SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{ID: "bid-1"}}}}}
	var responseRes hookstage.HookResult[hookstage.AuctionResponsePayload]
	runHook(func(ctx context.Context) {
		miCtx := hookstage.ModuleInvocationContext{ModuleContext: processedRes.ModuleContext}
		var err error
		responseRes, err = m.HandleAuctionResponseHook(ctx, miCtx, hookstage.AuctionResponsePayload{BidResponse: bidResp})
		if err != nil {
			t.Fatalf("response hook: %v", err)
		}
	})

	// Apply the mutation and assert the segment landed on the response ext.
	mutations := responseRes.ChangeSet.Mutations()
	if len(mutations) == 0 {
		t.Fatal("no mutation emitted — fan-out produced no segments (regression #1)")
	}
	payload := hookstage.AuctionResponsePayload{BidResponse: bidResp}
	for _, mut := range mutations {
		next, err := mut.Apply(payload)
		if err != nil {
			t.Fatalf("mutation apply: %v", err)
		}
		payload = next
	}
	if len(payload.BidResponse.Ext) == 0 {
		t.Fatal("response ext still empty after applying mutation")
	}
	// The exact JSON path is `adcp.segments`; a substring search is
	// enough — the strict assertion is "some segment survived the
	// hook plumbing", which is the invariant that broke.
	if !bytesContains(payload.BidResponse.Ext, "prov_package=pkg-a") {
		t.Errorf("expected prov_package=pkg-a in response ext; got %s", string(payload.BidResponse.Ext))
	}
}

func TestHooks_NoOpWhenPlacementMissing(t *testing.T) {
	m, cleanup := newHooksFixtureModule(t)
	defer cleanup()

	bidReq := &openrtb2.BidRequest{
		Site: &openrtb2.Site{Domain: "publisher.example"},
		Imp:  []openrtb2.Imp{{ID: "imp-1"}}, // no TagID
	}
	wrapper := &openrtb_ext.RequestWrapper{BidRequest: bidReq}
	res, err := m.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, hookstage.ProcessedAuctionRequestPayload{Request: wrapper})
	if err != nil {
		t.Fatalf("processed hook: %v", err)
	}
	if res.ModuleContext != nil {
		t.Error("expected no ModuleContext when there is nothing to fan out")
	}

	// Response hook should short-circuit cleanly when there's no holder.
	rres, err := m.HandleAuctionResponseHook(context.Background(), hookstage.ModuleInvocationContext{}, hookstage.AuctionResponsePayload{})
	if err != nil {
		t.Fatalf("response hook: %v", err)
	}
	if len(rres.ChangeSet.Mutations()) != 0 {
		t.Errorf("expected no mutation on empty ModuleContext; got %d", len(rres.ChangeSet.Mutations()))
	}
}

// bytesContains is a helper because JSON path assertions on []byte are
// noisy — we just want to see the segment string appear somewhere.
func bytesContains(hay []byte, needle string) bool {
	if len(needle) == 0 {
		return true
	}
	nb := []byte(needle)
	for i := 0; i+len(nb) <= len(hay); i++ {
		if string(hay[i:i+len(nb)]) == needle {
			return true
		}
	}
	return false
}
