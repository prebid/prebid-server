package identity

import (
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

// Shared constants used across the enrich and impression hooks.
const (
	// iiqSource is the eid source of the IntentIQ cookie id.
	iiqSource = "intentiq.com"
	// sourcePBSGo identifies the request source to the IntentIQ S2S API as prebid-server-go.
	sourcePBSGo = "pbgo"
	// biddingPlatformOpenRTB is the biddingPlatformId reported for OpenRTB impressions.
	biddingPlatformOpenRTB = "4"
	// defaultCurrency is used when the bid response omits a currency.
	defaultCurrency = "USD"
)

// flowContextKey is the module-context key under which the enrich hook stashes state for the
// impression hook within one auction.
const flowContextKey = "intentiq.identity.flow"

// flowContext carries state from the processed-auction-request (enrich) hook to the auction-response
// (impression report) hook via hookstage.ModuleContext within one auction.
//
// The Go auction-response payload exposes only the BidResponse (no request), so the request-derived
// fields the impression report needs (ref/ip/ua/auction id) are captured here by the enrich hook,
// which does have the request.
type flowContext struct {
	// start is when the enrich hook was entered, used for the enrich-response flow latency.
	start time.Time

	traceEnabled bool
	trace        []string

	// abTestUUID is the IIQ A/B test id returned by the resolution response, echoed on the report.
	abTestUUID string
	// terminationCause is the IIQ tc from the resolution response, if any.
	terminationCause *int64
	// auctionID is the bid request id (reported as prebidAuctionId / partnerAuctionId).
	auctionID string
	// ref is the site domain/page or app bundle/name (reported as vrref).
	ref string
	// ip is device.ip or device.ipv6.
	ip string
	// ua is device.ua.
	ua string
}

// setFlowContext stores fc under flowContextKey in a fresh module context and returns it for
// HookResult.ModuleContext.
func setFlowContext(fc flowContext) *hookstage.ModuleContext {
	mctx := hookstage.NewModuleContext()
	mctx.Set(flowContextKey, fc)
	return mctx
}

// getFlowContext retrieves the flow context stashed by the enrich hook, if present.
func getFlowContext(mctx *hookstage.ModuleContext) (flowContext, bool) {
	if mctx == nil {
		return flowContext{}, false
	}
	v, ok := mctx.Get(flowContextKey)
	if !ok {
		return flowContext{}, false
	}
	fc, ok := v.(flowContext)
	return fc, ok
}

// resolution is the outcome of identity resolution (from cache or a live call), carrying the eids to
// merge plus the fields threaded to the impression hook.
type resolution struct {
	eids              []openrtb2.EID
	abTestUUID        string
	terminationCause  *int64
	notEnrichedReason string
}
