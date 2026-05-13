package ctv_vast_enrichment_test

// End-to-end test suite for the ctv_vast_enrichment module.
//
// These tests exercise the full hook path — from HandleRawBidderResponseHook through
// config merging, enrichVastDocument, and model.Marshal — using real sub-package
// implementations (enrich.NewEnricher, format.NewFormatter, select.NewSelector)
// rather than mocks, so regressions in integration points are caught.
//
// Test matrix:
//  A. Hook path correctness
//     A1  Video bid enriched — <Pricing> and <Advertiser> injected
//     A2  Banner bid passes through untouched (BidType guard)
//     A3  Native bid passes through untouched (BidType guard)
//     A4  BidMeta preserved on enriched TypedBid
//     A5  BidderResponse.Currency used in <Pricing>, not DefaultCurrency from config
//     A6  Fallback to DefaultCurrency when BidderResponse.Currency is empty
//     A7  VAST_WINS: existing <Pricing> not overwritten
//     A8  Unknown/DSP-specific VAST extensions preserved after marshal
//     A9  Mixed bid types — only video bids modified
//
//  B. Config correctness
//     B1  VAST_WINS CollisionPolicy round-trips through account config
//     B2  Account config overrides host config
//
//  C. Pipeline end-to-end (BuildVastFromBidResponse with real components)
//     C1  Single video bid → enriched VAST with <Pricing> and <Advertiser>
//     C2  Ad pod — multiple bids → correct sequence attributes
//     C3  No bids → NoAd VAST returned
//     C4  All-invalid VAST, skeleton disabled → NoAd
//     C5  All-invalid VAST, skeleton enabled → VAST with warnings
//     C6  Duration from bid meta injected into <Linear><Duration>
//     C7  IAB categories injected as extension
//     C8  Debug extension enabled → <BidID>/<Seat> present in output
//
//  D. Regression — previously reported bugs (see CTV.md / ctv-bugs-and-resolve.md)
//     D1  Non-USD DSP currency preserved in <Pricing> (BUG 1)
//     D2  VAST_WINS policy not silently converted to Reject (BUG 2)
//     D3  Hook uses enrich subpackage — debug extension added when debug=true (BUG 3)
//     D4  clearInnerXML does not drop <MediaFiles> content (BUG 4)
//     D5  BidMeta fields (NetworkID, AdvertiserID, BrandID) survive hook (BUG 6)
//     D6  Only first ADomain used in <Advertiser>, not comma-joined list (BUG 7)

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	ctv "github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment"
	"github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment/enrich"
	"github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment/format"
	bidselect "github.com/prebid/prebid-server/v4/modules/prebid/ctv_vast_enrichment/select"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Shared VAST fixtures
// ---------------------------------------------------------------------------

const (
	// minimalVAST is a well-formed VAST 3.0 with one InLine ad containing
	// a video MediaFile. Used as a baseline "clean" input for most tests.
	minimalVAST = `<VAST version="3.0"><Ad id="ad1"><InLine>` +
		`<AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle>` +
		`<Creatives><Creative><Linear>` +
		`<Duration>00:00:30</Duration>` +
		`<MediaFiles><MediaFile delivery="progressive" type="video/mp4" width="1920" height="1080">` +
		`<![CDATA[https://example.com/video.mp4]]></MediaFile></MediaFiles>` +
		`</Linear></Creative></Creatives>` +
		`</InLine></Ad></VAST>`

	// vastWithPricing already contains <Pricing currency="GBP">3.00</Pricing>.
	// Used to verify VAST_WINS collision policy.
	vastWithPricing = `<VAST version="3.0"><Ad id="ad1"><InLine>` +
		`<AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle>` +
		`<Pricing model="CPM" currency="GBP">3.00</Pricing>` +
		`<Creatives></Creatives>` +
		`</InLine></Ad></VAST>`

	// vastWithExtensions contains a DSP-specific <Extension type="dsp_custom">.
	// Used to verify clearInnerXML does not drop unknown elements.
	vastWithExtensions = `<VAST version="3.0"><Ad id="ad1"><InLine>` +
		`<AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle>` +
		`<Creatives><Creative><Linear>` +
		`<Duration>00:00:15</Duration>` +
		`<MediaFiles><MediaFile delivery="progressive" type="video/mp4">` +
		`<![CDATA[https://cdn.dsp.example/ad.mp4]]></MediaFile></MediaFiles>` +
		`</Linear></Creative></Creatives>` +
		`<Extensions>` +
		`<Extension type="dsp_custom"><DspTracker>https://tracker.dsp.example/ping</DspTracker></Extension>` +
		`</Extensions>` +
		`</InLine></Ad></VAST>`
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// hookHandler is the subset of the Module interface we need.
type hookHandler interface {
	HandleRawBidderResponseHook(
		ctx context.Context,
		miCtx hookstage.ModuleInvocationContext,
		payload hookstage.RawBidderResponsePayload,
	) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error)
}

// buildModule creates a Module via the public Builder with given JSON host config.
func buildModule(t *testing.T, hostCfgJSON string) hookHandler {
	t.Helper()
	m, err := ctv.Builder(json.RawMessage(hostCfgJSON), moduledeps.ModuleDeps{})
	require.NoError(t, err)
	h, ok := m.(hookHandler)
	require.True(t, ok, "Builder result must implement HandleRawBidderResponseHook")
	return h
}

// applyMutations replays all ChangeSet mutations onto payload and returns the result.
func applyMutations(
	t *testing.T,
	result hookstage.HookResult[hookstage.RawBidderResponsePayload],
	payload hookstage.RawBidderResponsePayload,
) hookstage.RawBidderResponsePayload {
	t.Helper()
	for _, mut := range result.ChangeSet.Mutations() {
		var err error
		payload, err = mut.Apply(payload)
		require.NoError(t, err)
	}
	return payload
}

// enabledCtx returns a minimal account config that enables the module.
func enabledCtx() hookstage.ModuleInvocationContext {
	return hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled":true}`),
	}
}

// videoTypedBid wraps a Bid in a TypedBid with BidTypeVideo.
func videoTypedBid(bid *openrtb2.Bid) *adapters.TypedBid {
	return &adapters.TypedBid{Bid: bid, BidType: openrtb_ext.BidTypeVideo}
}

// newRealComponents returns real (non-mock) selector/enricher/formatter.
func newRealComponents(strategy ctv.SelectionStrategy) (ctv.BidSelector, ctv.Enricher, ctv.Formatter) {
	return bidselect.NewSelector(strategy), enrich.NewEnricher(), format.NewFormatter()
}

// ---------------------------------------------------------------------------
// A. Hook path correctness
// ---------------------------------------------------------------------------

// A1 — video bid is enriched; <Pricing> and <Advertiser> injected.
func TestE2E_A1_VideoBidEnriched(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "rubicon",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{
					ID:      "b1",
					Price:   2.50,
					ADomain: []string{"brand.com"},
					AdM:     minimalVAST,
				}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, "<Pricing", "expected <Pricing> to be injected")
	assert.Contains(t, adm, "2.5", "expected price value in VAST")
	assert.Contains(t, adm, `currency="USD"`)
	assert.Contains(t, adm, "brand.com", "expected <Advertiser> with domain")
}

// A2 — banner bid must pass through unchanged.
func TestE2E_A2_BannerBidNotTouched(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	bannerAdM := `<html><body>banner content</body></html>`
	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{{
				Bid:     &openrtb2.Bid{ID: "b1", Price: 1.0, AdM: bannerAdM},
				BidType: openrtb_ext.BidTypeBanner,
			}},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	assert.Equal(t, bannerAdM, payload.BidderResponse.Bids[0].Bid.AdM,
		"banner AdM must not be modified")
}

// A3 — native bid must pass through unchanged.
func TestE2E_A3_NativeBidNotTouched(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	nativeAdM := `{"ver":"1.1","assets":[]}`
	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{{
				Bid:     &openrtb2.Bid{ID: "b1", Price: 0.5, AdM: nativeAdM},
				BidType: openrtb_ext.BidTypeNative,
			}},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	assert.Equal(t, nativeAdM, payload.BidderResponse.Bids[0].Bid.AdM,
		"native AdM must not be modified")
}

// A4 — BidMeta must be preserved on the enriched TypedBid.
func TestE2E_A4_BidMetaPreserved(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{{
				Bid:     &openrtb2.Bid{ID: "b1", Price: 1.0, AdM: minimalVAST},
				BidType: openrtb_ext.BidTypeVideo,
				BidMeta: &openrtb_ext.ExtBidPrebidMeta{
					NetworkID:    42,
					AdvertiserID: 99,
					BrandID:      7,
				},
			}},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	meta := payload.BidderResponse.Bids[0].BidMeta
	require.NotNil(t, meta, "BidMeta must not be nil after hook")
	assert.Equal(t, 42, meta.NetworkID)
	assert.Equal(t, 99, meta.AdvertiserID)
	assert.Equal(t, 7, meta.BrandID)
}

// A5 — BidderResponse.Currency must be used in <Pricing>, not DefaultCurrency from host config.
func TestE2E_A5_BidderResponseCurrencyUsedInPricing(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`) // host says USD

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder_eur",
		BidderResponse: &adapters.BidderResponse{
			Currency: "EUR", // DSP responds in EUR
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 1.80, AdM: minimalVAST}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, `currency="EUR"`, "EUR from BidderResponse must appear in <Pricing>")
	assert.NotContains(t, adm, `currency="USD"`, "USD from host config must NOT override DSP currency")
}

// A6 — fallback to DefaultCurrency when BidderResponse.Currency is empty.
func TestE2E_A6_FallbackToDefaultCurrencyWhenBidderCurrencyEmpty(t *testing.T) {
	module := buildModule(t, `{"default_currency":"GBP"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "", // DSP omits currency
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 3.00, AdM: minimalVAST}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, `currency="GBP"`, "should fallback to host DefaultCurrency GBP")
}

// A7 — VAST_WINS: existing <Pricing> in VAST must not be overwritten.
func TestE2E_A7_VastWinsPreservesExistingPricing(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD","collision_policy":"VAST_WINS"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{
					ID:    "b1",
					Price: 9.99, // bidder price
					AdM:   vastWithPricing, // already has GBP 3.00
				}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, "GBP", "original currency GBP must be preserved")
	assert.Contains(t, adm, "3.00", "original price 3.00 must be preserved")
	assert.NotContains(t, adm, "9.99", "bidder price must NOT overwrite existing VAST pricing")
}

// A8 — DSP-specific VAST extensions must survive the marshal round-trip.
func TestE2E_A8_UnknownVastExtensionsPreserved(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 2.00, AdM: vastWithExtensions}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, "dsp_custom", "DSP extension type must survive marshal")
	assert.Contains(t, adm, "https://tracker.dsp.example/ping", "DSP tracker URL must survive marshal")
}

// A9 — mixed bid types: only video bids enriched, others unchanged.
func TestE2E_A9_MixedBidTypesOnlyVideoEnriched(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	bannerAdM := `<html>banner</html>`
	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				{
					Bid:     &openrtb2.Bid{ID: "banner-bid", Price: 1.0, AdM: bannerAdM},
					BidType: openrtb_ext.BidTypeBanner,
				},
				videoTypedBid(&openrtb2.Bid{ID: "video-bid", Price: 2.5, AdM: minimalVAST}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	assert.Equal(t, bannerAdM, payload.BidderResponse.Bids[0].Bid.AdM,
		"banner bid must not be modified")
	assert.Contains(t, payload.BidderResponse.Bids[1].Bid.AdM, "<Pricing",
		"video bid must be enriched")
}

// ---------------------------------------------------------------------------
// B. Config correctness
// ---------------------------------------------------------------------------

// B1 — VAST_WINS collision policy set via account config round-trips correctly
// (not silently dropped to Reject). Verified through observable behaviour: existing
// <Pricing> in VAST is preserved when VAST_WINS is set.
func TestE2E_B1_VastWinsCollisionPolicyRoundTrip(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`) // host has no collision_policy

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 9.99, AdM: vastWithPricing}),
			},
		},
	}

	// Account overrides collision_policy to VAST_WINS
	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled":true,"collision_policy":"VAST_WINS"}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, "GBP", "VAST_WINS: original GBP currency must survive")
	assert.Contains(t, adm, "3.00", "VAST_WINS: original price must survive")
	assert.NotContains(t, adm, "9.99", "VAST_WINS: bidder price must not replace existing pricing")
}

// B2 — account config overrides host config; host fields not in account are preserved.
func TestE2E_B2_AccountConfigOverridesHost(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD","receiver":"GENERIC"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "", // empty — falls back to config currency
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 1.0, AdM: minimalVAST}),
			},
		},
	}

	// Account overrides currency to EUR
	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled":true,"default_currency":"EUR"}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, `currency="EUR"`, "account currency EUR must override host USD")
}

// ---------------------------------------------------------------------------
// C. Pipeline end-to-end (BuildVastFromBidResponse with real components)
// ---------------------------------------------------------------------------

// C1 — single video bid → enriched VAST with <Pricing> and <Advertiser>.
func TestE2E_C1_PipelineSingleBidEnriched(t *testing.T) {
	cfg := ctv.DefaultConfig()
	cfg.SelectionStrategy = ctv.SelectionSingle
	cfg.DefaultCurrency = "USD"

	req := &openrtb2.BidRequest{ID: "req-1"}
	resp := &openrtb2.BidResponse{
		ID:  "resp-1",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "rubicon",
			Bid: []openrtb2.Bid{{
				ID:      "bid-1",
				ImpID:   "imp-1",
				Price:   5.00,
				AdM:     minimalVAST,
				ADomain: []string{"brand.example.com"},
			}},
		}},
	}

	selector, enricher, formatter := newRealComponents(cfg.SelectionStrategy)
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	require.False(t, result.NoAd)

	xmlStr := string(result.VastXML)
	assert.Contains(t, xmlStr, "<Pricing")
	assert.Contains(t, xmlStr, "5")
	assert.Contains(t, xmlStr, "brand.example.com")
}

// C2 — ad pod: multiple bids → sequence attributes set correctly.
func TestE2E_C2_PipelineAdPodSequenceAttributes(t *testing.T) {
	cfg := ctv.DefaultConfig()
	cfg.SelectionStrategy = ctv.SelectionTopN
	cfg.MaxAdsInPod = 3
	cfg.DefaultCurrency = "USD"

	makeVAST := func(id string) string {
		return strings.ReplaceAll(minimalVAST, `id="ad1"`, `id="`+id+`"`)
	}

	req := &openrtb2.BidRequest{ID: "req-pod"}
	resp := &openrtb2.BidResponse{
		ID:  "resp-pod",
		Cur: "USD",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "bidder1",
			Bid: []openrtb2.Bid{
				{ID: "b1", ImpID: "imp-1", Price: 10.0, AdM: makeVAST("ad-1")},
				{ID: "b2", ImpID: "imp-2", Price: 8.0, AdM: makeVAST("ad-2")},
				{ID: "b3", ImpID: "imp-3", Price: 6.0, AdM: makeVAST("ad-3")},
			},
		}},
	}

	selector, enricher, formatter := newRealComponents(cfg.SelectionStrategy)
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	require.False(t, result.NoAd)
	assert.Len(t, result.Selected, 3)

	xmlStr := string(result.VastXML)
	assert.Contains(t, xmlStr, `sequence="1"`)
	assert.Contains(t, xmlStr, `sequence="2"`)
	assert.Contains(t, xmlStr, `sequence="3"`)
}

// C3 — no bids → NoAd VAST returned.
func TestE2E_C3_PipelineNoBidsReturnsNoAd(t *testing.T) {
	cfg := ctv.DefaultConfig()

	req := &openrtb2.BidRequest{ID: "req-empty"}
	resp := &openrtb2.BidResponse{ID: "resp-empty"}

	selector, enricher, formatter := newRealComponents(ctv.SelectionSingle)
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	assert.True(t, result.NoAd)
	assert.NotEmpty(t, result.VastXML)
	assert.Contains(t, string(result.VastXML), "<VAST")
}

// C4 — invalid VAST, skeleton disabled → NoAd.
func TestE2E_C4_InvalidVastNoSkeletonReturnsNoAd(t *testing.T) {
	cfg := ctv.DefaultConfig()
	cfg.AllowSkeletonVast = false

	req := &openrtb2.BidRequest{ID: "req-invalid"}
	resp := &openrtb2.BidResponse{
		ID: "resp-invalid",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "bidder1",
			Bid:  []openrtb2.Bid{{ID: "b1", Price: 5.0, AdM: "not-xml-at-all"}},
		}},
	}

	selector, enricher, formatter := newRealComponents(ctv.SelectionSingle)
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	assert.True(t, result.NoAd)
}

// C5 — invalid VAST, skeleton enabled → returns VAST with warning.
func TestE2E_C5_InvalidVastWithSkeletonReturnsVast(t *testing.T) {
	cfg := ctv.DefaultConfig()
	cfg.AllowSkeletonVast = true

	req := &openrtb2.BidRequest{ID: "req-skel"}
	resp := &openrtb2.BidResponse{
		ID: "resp-skel",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "bidder1",
			Bid:  []openrtb2.Bid{{ID: "b1", Price: 5.0, AdM: "not-xml-at-all"}},
		}},
	}

	selector, enricher, formatter := newRealComponents(ctv.SelectionSingle)
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	assert.False(t, result.NoAd, "skeleton=true must return a VAST even for invalid AdM")
	assert.NotEmpty(t, result.Warnings, "skeleton VAST must produce at least one warning")
}

// C6 — DurSec from CanonicalMeta injected into <Linear><Duration>.
func TestE2E_C6_DurationInjectedFromMeta(t *testing.T) {
	cfg := ctv.DefaultConfig()
	cfg.DefaultCurrency = "USD"

	vastNoDuration := `<VAST version="3.0"><Ad id="ad1"><InLine>` +
		`<AdSystem>Test</AdSystem><AdTitle>No Duration</AdTitle>` +
		`<Creatives><Creative><Linear>` +
		`<MediaFiles><MediaFile type="video/mp4"><![CDATA[https://cdn.example/v.mp4]]></MediaFile></MediaFiles>` +
		`</Linear></Creative></Creatives>` +
		`</InLine></Ad></VAST>`

	req := &openrtb2.BidRequest{ID: "req-dur"}
	resp := &openrtb2.BidResponse{
		ID: "resp-dur",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "bidder1",
			Bid:  []openrtb2.Bid{{ID: "b1", Price: 2.0, AdM: vastNoDuration}},
		}},
	}

	sel := &durationInjectingSelector{durSec: 45}
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, sel, enrich.NewEnricher(), format.NewFormatter())
	require.NoError(t, err)
	require.False(t, result.NoAd)

	xmlStr := string(result.VastXML)
	assert.Contains(t, xmlStr, "00:00:45", "duration 45s must appear as 00:00:45")
}

// C7 — IAB categories injected as VAST extension.
func TestE2E_C7_IABCategoriesInjectedAsExtension(t *testing.T) {
	cfg := ctv.DefaultConfig()
	cfg.DefaultCurrency = "USD"

	req := &openrtb2.BidRequest{ID: "req-cat"}
	resp := &openrtb2.BidResponse{
		ID: "resp-cat",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "bidder1",
			Bid: []openrtb2.Bid{{
				ID:    "b1",
				Price: 3.0,
				AdM:   minimalVAST,
				Cat:   []string{"IAB1", "IAB2-3"},
			}},
		}},
	}

	selector, enricher, formatter := newRealComponents(ctv.SelectionSingle)
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	require.False(t, result.NoAd)

	xmlStr := string(result.VastXML)
	assert.Contains(t, xmlStr, "IAB1")
	assert.Contains(t, xmlStr, "IAB2-3")
	assert.Contains(t, xmlStr, "iab_category")
}

// C8 — debug extension enabled → <BidID> and <Seat> in output.
func TestE2E_C8_DebugExtensionIncludesBidIDAndSeat(t *testing.T) {
	cfg := ctv.DefaultConfig()
	cfg.DefaultCurrency = "USD"
	cfg.Debug = true

	req := &openrtb2.BidRequest{ID: "req-dbg"}
	resp := &openrtb2.BidResponse{
		ID: "resp-dbg",
		SeatBid: []openrtb2.SeatBid{{
			Seat: "rubicon",
			Bid:  []openrtb2.Bid{{ID: "debug-bid-123", Price: 4.0, AdM: minimalVAST}},
		}},
	}

	selector, enricher, formatter := newRealComponents(ctv.SelectionSingle)
	result, err := ctv.BuildVastFromBidResponse(context.Background(), req, resp, cfg, selector, enricher, formatter)
	require.NoError(t, err)
	require.False(t, result.NoAd)

	xmlStr := string(result.VastXML)
	assert.Contains(t, xmlStr, "<BidID>debug-bid-123</BidID>")
	assert.Contains(t, xmlStr, "<Seat>rubicon</Seat>")
}

// ---------------------------------------------------------------------------
// D. Regression tests — previously reported bugs
// ---------------------------------------------------------------------------

// D1 — (BUG 1) Non-USD DSP currency preserved in <Pricing currency="...">.
func TestE2E_D1_NonUSDCurrencyPreserved(t *testing.T) {
	for _, dspCurrency := range []string{"EUR", "JPY", "BRL", "AUD"} {
		t.Run(dspCurrency, func(t *testing.T) {
			module := buildModule(t, `{"default_currency":"USD"}`)

			payload := hookstage.RawBidderResponsePayload{
				Bidder: "bidder",
				BidderResponse: &adapters.BidderResponse{
					Currency: dspCurrency,
					Bids: []*adapters.TypedBid{
						videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 1.0, AdM: minimalVAST}),
					},
				},
			}

			result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
			require.NoError(t, err)
			payload = applyMutations(t, result, payload)

			adm := payload.BidderResponse.Bids[0].Bid.AdM
			assert.Contains(t, adm, `currency="`+dspCurrency+`"`,
				"DSP currency %s must appear in <Pricing>", dspCurrency)
			assert.NotContains(t, adm, `currency="USD"`,
				"host DefaultCurrency USD must NOT override DSP currency %s", dspCurrency)
		})
	}
}

// D2 — (BUG 2) VAST_WINS collision policy must not silently become CollisionReject.
// Observable effect: existing <Pricing> in VAST is preserved, not replaced.
func TestE2E_D2_VastWinsNotSilentlyDropped(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD","collision_policy":"VAST_WINS"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 9.99, AdM: vastWithPricing}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, "3.00", "VAST_WINS: original price must not be overwritten")
	assert.Contains(t, adm, "GBP", "VAST_WINS: original currency must not be overwritten")
	assert.NotContains(t, adm, "9.99", "VAST_WINS: bidder price must not replace existing")
}

// D3 — (BUG 3) Hook uses enrich subpackage: debug extension added when debug=true via account config.
// This verifies the hook path actually reaches enrich.VastEnricher, not a partial inline impl.
func TestE2E_D3_HookUsesEnrichSubpackage_DebugExtension(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled":true,"debug":true}`),
	}

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "rubicon",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "hook-debug-bid", Price: 2.0, AdM: minimalVAST}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	// enrich.VastEnricher adds <Extension type="openrtb"> with <BidID> when debug=true
	assert.Contains(t, adm, `type="openrtb"`,
		"debug extension must be present — proves hook reached enrich subpackage")
	assert.Contains(t, adm, "hook-debug-bid",
		"BidID must appear in debug extension")
}

// D4 — (BUG 4) clearInnerXML must not drop <MediaFiles> content.
func TestE2E_D4_MediaFilesPreservedAfterMarshal(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{ID: "b1", Price: 1.0, AdM: minimalVAST}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, "https://example.com/video.mp4",
		"MediaFile URL must survive clearInnerXML + marshal")
	assert.Contains(t, adm, "<MediaFile", "MediaFile element must survive")
	assert.Contains(t, adm, "video/mp4", "media type attribute must survive")
}

// D5 — (BUG 6) BidMeta fields survive hook mutation.
func TestE2E_D5_BidMetaFieldsSurviveHook(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{{
				Bid:     &openrtb2.Bid{ID: "b1", Price: 1.5, AdM: minimalVAST},
				BidType: openrtb_ext.BidTypeVideo,
				BidMeta: &openrtb_ext.ExtBidPrebidMeta{
					NetworkID:         100,
					AdvertiserID:      200,
					BrandID:           300,
					PrimaryCategoryID: "IAB1",
				},
			}},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	meta := payload.BidderResponse.Bids[0].BidMeta
	require.NotNil(t, meta)
	assert.Equal(t, 100, meta.NetworkID)
	assert.Equal(t, 200, meta.AdvertiserID)
	assert.Equal(t, 300, meta.BrandID)
	assert.Equal(t, "IAB1", meta.PrimaryCategoryID)
}

// D6 — (BUG 7) Only first ADomain used in <Advertiser>, not all domains comma-joined.
func TestE2E_D6_OnlyFirstADomainUsedInAdvertiser(t *testing.T) {
	module := buildModule(t, `{"default_currency":"USD"}`)

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "bidder1",
		BidderResponse: &adapters.BidderResponse{
			Currency: "USD",
			Bids: []*adapters.TypedBid{
				videoTypedBid(&openrtb2.Bid{
					ID:      "b1",
					Price:   1.0,
					ADomain: []string{"primary.com", "secondary.com", "tertiary.com"},
					AdM:     minimalVAST,
				}),
			},
		},
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), enabledCtx(), payload)
	require.NoError(t, err)
	payload = applyMutations(t, result, payload)

	adm := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, adm, "primary.com", "first domain must be in <Advertiser>")
	assert.NotContains(t, adm, "primary.com,secondary.com",
		"domains must NOT be comma-joined in <Advertiser>")

	// Parse XML to check <Advertiser> value is exactly first domain
	var vastParsed struct {
		XMLName xml.Name `xml:"VAST"`
		Ad      struct {
			InLine struct {
				Advertiser string `xml:"Advertiser"`
			} `xml:"InLine"`
		} `xml:"Ad"`
	}
	err = xml.Unmarshal([]byte(adm), &vastParsed)
	require.NoError(t, err)
	assert.Equal(t, "primary.com", vastParsed.Ad.InLine.Advertiser,
		"<Advertiser> must contain exactly the first domain")
}

// ---------------------------------------------------------------------------
// Test helpers — custom selector implementations
// ---------------------------------------------------------------------------

// durationInjectingSelector wraps the default selector and injects DurSec into each CanonicalMeta.
type durationInjectingSelector struct {
	durSec int
}

func (s *durationInjectingSelector) Select(
	req *openrtb2.BidRequest,
	resp *openrtb2.BidResponse,
	cfg ctv.ReceiverConfig,
) ([]ctv.SelectedBid, []string, error) {
	base := bidselect.NewSelector(ctv.SelectionSingle)
	selected, warnings, err := base.Select(req, resp, cfg)
	if err != nil {
		return selected, warnings, err
	}
	for i := range selected {
		selected[i].Meta.DurSec = s.durSec
	}
	return selected, warnings, nil
}
