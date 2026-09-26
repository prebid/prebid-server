package vastlint

import (
	"context"
	"fmt"
	"strings"

	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

type finding struct {
	ID string
}

// HandleRawBidderResponseHook checks video adm that looks like VAST.
// Revenue-impact findings are counted. Bids are dropped only when
// reject_revenue is set.
func (m Module) HandleRawBidderResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.RawBidderResponsePayload,
) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	result := hookstage.HookResult[hookstage.RawBidderResponsePayload]{}
	reject, err := m.reject(miCtx.AccountConfig)
	if err != nil {
		return result, err
	}
	if m.validate == nil || payload.BidderResponse == nil {
		return result, nil
	}

	rec := m.record
	if rec == nil {
		rec = currentRecorder()
	}
	caller := payload.Bidder
	if caller == "" {
		caller = "unknown"
	}

	kept := make([]*adapters.TypedBid, 0, len(payload.BidderResponse.Bids))
	dropped := false
	for _, bid := range payload.BidderResponse.Bids {
		if bid == nil || bid.Bid == nil || bid.BidType != openrtb_ext.BidTypeVideo || !looksLikeVAST(bid.Bid.AdM) {
			kept = append(kept, bid)
			continue
		}

		issues, verr := m.validate(bid.Bid.AdM)
		if verr != nil {
			result.Errors = append(result.Errors, verr.Error())
			kept = append(kept, bid)
			continue
		}

		var revenueIDs []string
		for _, issue := range issues {
			if !revenueImpact(issue.ID) {
				continue
			}
			revenueIDs = append(revenueIDs, issue.ID)
			rec.Finding(caller, issue.ID, true)
		}
		if reject && len(revenueIDs) > 0 {
			dropped = true
			result.DebugMessages = append(result.DebugMessages, fmt.Sprintf(
				"openadtech.vastlint dropped bid %s from %s: %s",
				bid.Bid.ID,
				caller,
				strings.Join(revenueIDs, ", "),
			))
			continue
		}
		kept = append(kept, bid)
	}

	if dropped {
		result.ChangeSet.RawBidderResponse().Bids().UpdateBids(kept)
	}
	return result, nil
}

func (m Module) reject(account []byte) (bool, error) {
	if len(account) == 0 {
		return m.cfg.RejectRevenue, nil
	}
	cfg, err := parseConfig(account)
	if err != nil {
		return false, err
	}
	return cfg.RejectRevenue, nil
}
