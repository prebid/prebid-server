package doohcreativeapproval

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func testModuleConfig() moduleConfig {
	return moduleConfig{
		Enabled:            true,
		Platforms:          []string{defaultPlatformDOOH},
		Endpoint:           "http://approval.example.com/creative-approval",
		TimeoutMS:          defaultTimeoutMS,
		CacheSizeBytes:     defaultCacheSizeBytes,
		ApprovedTTLSeconds: defaultApprovedTTLSeconds,
		RejectedTTLSeconds: defaultRejectedTTLSeconds,
		PendingTTLSeconds:  defaultPendingTTLSeconds,
	}
}

func testBid(crid string) *entities.PbsOrtbBid {
	return &entities.PbsOrtbBid{
		Bid: &openrtb2.Bid{
			ID:      "bid-" + crid,
			ImpID:   "imp-1",
			Price:   1.23,
			CrID:    crid,
			AdID:    "ad-" + crid,
			CID:     "campaign-" + crid,
			ADomain: []string{"advertiser.example"},
			Cat:     []string{"IAB1"},
			W:       1920,
			H:       1080,
			Dur:     15,
			DealID:  "deal-" + crid,
			IURL:    "https://example.com/" + crid + ".jpg",
		},
		BidType: openrtb_ext.BidTypeVideo,
	}
}

func testResponses(bidder openrtb_ext.BidderName, bids ...*entities.PbsOrtbBid) map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid {
	return map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		bidder: {
			Bids:     bids,
			Currency: "USD",
			Seat:     bidder.String(),
		},
	}
}

func testActiveModuleContext() *hookstage.ModuleContext {
	moduleContext := hookstage.NewModuleContext()
	moduleContext.Set(activeContextKey, true)
	return moduleContext
}

func applyAllProcessedMutations(payload hookstage.AllProcessedBidResponsesPayload, result hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]) hookstage.AllProcessedBidResponsesPayload {
	for _, mutation := range result.ChangeSet.Mutations() {
		payload, _ = mutation.Apply(payload)
	}
	return payload
}

type fakeApprovalProvider struct {
	statuses  map[string]approvalStatus
	warnings  []string
	err       error
	calls     int
	creatives []creativeApproval
}

func (p *fakeApprovalProvider) Lookup(_ context.Context, _ moduleConfig, _ string, creatives []creativeApproval) (map[string]approvalStatus, []string, error) {
	p.calls++
	p.creatives = append([]creativeApproval(nil), creatives...)
	if p.err != nil {
		return nil, p.warnings, p.err
	}
	statuses := make(map[string]approvalStatus, len(p.statuses))
	for id, status := range p.statuses {
		statuses[id] = status
	}
	return statuses, p.warnings, nil
}

func testAccountConfig(t interface{ Helper() }, value string) json.RawMessage {
	t.Helper()
	return json.RawMessage(value)
}

var errApprovalProvider = errors.New("approval service unavailable")
