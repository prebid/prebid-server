package hookstage

import (
	"context"

	"github.com/prebid/openrtb/v19/openrtb2"
)

// BidderRequest hooks are invoked for each bidder participating in auction.
//
// At this stage, account config is available,
// so it can be configured at the account-level execution plan,
// the account-level module config is passed to hooks.
//
// Rejection results in skipping the bidder's request.
type BidderRequest interface {
	HandleBidderRequestHook(
		context.Context,
		ModuleInvocationContext,
		BidderRequestPayload,
	) (HookResult[BidderRequestPayload], error)
}

// BidderRequestPayload consists of the openrtb2.BidRequest object
// distilled for the particular bidder.
// Hooks are allowed to modify openrtb2.BidRequest using mutations.
type BidderRequestPayload struct {
	BidRequest *openrtb2.BidRequest
	Bidder     string
}

func (brp *BidderRequestPayload) GetBidderRequestPayload() *openrtb2.BidRequest {
	return brp.BidRequest
}

func (brp *BidderRequestPayload) SetBidderRequestPayload(br *openrtb2.BidRequest) {
	brp.BidRequest = br
}

// PayloadBidderRequest indicated of hook carries a bid request.
// used for activities, name can be better
type PayloadBidderRequest interface {
	GetBidderRequestPayload() *openrtb2.BidRequest
	SetBidderRequestPayload(br *openrtb2.BidRequest)
}
