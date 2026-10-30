package doohcreativeapproval

import (
	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

type approvalCandidate struct {
	CreativeApprovalID string
	Exempt             bool
}

func collectCreativeApprovals(responses map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, cfg moduleConfig, accountID string) (map[string]creativeApproval, []string) {
	creatives := make(map[string]creativeApproval)
	warnings := make([]string, 0)

	for bidder, seatBid := range responses {
		if seatBid == nil || isBidderExempt(cfg, bidder.String()) {
			continue
		}
		for _, pbsBid := range seatBid.Bids {
			creative, ok := newCreativeApproval(accountID, bidder, pbsBid)
			if !ok {
				warnings = append(warnings, "bid skipped from approval lookup because it is missing creative id")
				continue
			}
			if _, exists := creatives[creative.CreativeApprovalID]; exists {
				continue
			}
			creatives[creative.CreativeApprovalID] = creative
		}
	}

	return creatives, warnings
}

func needsApprovalFilter(responses map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, cfg moduleConfig, accountID string, statuses map[string]approvalStatus) bool {
	for bidder, seatBid := range responses {
		if seatBid == nil {
			continue
		}
		for _, pbsBid := range seatBid.Bids {
			candidate := approvalCandidateForBid(accountID, bidder, pbsBid, cfg)
			if !candidate.Exempt && statuses[candidate.CreativeApprovalID] != approvalStatusApproved {
				return true
			}
		}
	}
	return false
}

func filterResponsesByApproval(responses map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, cfg moduleConfig, accountID string, statuses map[string]approvalStatus) int {
	removed := 0
	for bidder, seatBid := range responses {
		if seatBid == nil {
			delete(responses, bidder)
			continue
		}

		keptBids := make([]*entities.PbsOrtbBid, 0, len(seatBid.Bids))
		for _, pbsBid := range seatBid.Bids {
			candidate := approvalCandidateForBid(accountID, bidder, pbsBid, cfg)
			if candidate.Exempt || statuses[candidate.CreativeApprovalID] == approvalStatusApproved {
				keptBids = append(keptBids, pbsBid)
				continue
			}
			removed++
		}

		if len(keptBids) == 0 {
			delete(responses, bidder)
			continue
		}
		seatBid.Bids = keptBids
	}

	return removed
}

func approvalCandidateForBid(accountID string, bidder openrtb_ext.BidderName, pbsBid *entities.PbsOrtbBid, cfg moduleConfig) approvalCandidate {
	if isBidderExempt(cfg, bidder.String()) {
		return approvalCandidate{Exempt: true}
	}
	if pbsBid == nil || pbsBid.Bid == nil || pbsBid.Bid.CrID == "" {
		return approvalCandidate{}
	}
	return approvalCandidate{
		CreativeApprovalID: creativeApprovalID(accountID, bidder, pbsBid.Bid.CrID),
	}
}
