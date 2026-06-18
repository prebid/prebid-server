package doohcreativeapproval

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/prebid/prebid-server/v4/exchange/entities"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

const creativeApprovalIDVersion = "v1:"

func creativeApprovalID(accountID string, bidder openrtb_ext.BidderName, creativeID string) string {
	hash := sha256.Sum256([]byte(accountID + "\x1f" + bidder.String() + "\x1f" + creativeID))
	return creativeApprovalIDVersion + hex.EncodeToString(hash[:])
}

func newCreativeApproval(accountID string, bidder openrtb_ext.BidderName, pbsBid *entities.PbsOrtbBid) (creativeApproval, bool) {
	if pbsBid == nil || pbsBid.Bid == nil || pbsBid.Bid.CrID == "" {
		return creativeApproval{}, false
	}

	bid := pbsBid.Bid
	return creativeApproval{
		CreativeApprovalID: creativeApprovalID(accountID, bidder, bid.CrID),
		Bidder:             bidder.String(),
		CreativeID:         bid.CrID,
		AdID:               bid.AdID,
		CampaignID:         bid.CID,
		AdvertiserDomains:  bid.ADomain,
		Categories:         bid.Cat,
		CategoryTaxonomy:   int(bid.CatTax),
		MediaType:          string(pbsBid.BidType),
		Width:              bid.W,
		Height:             bid.H,
		Duration:           bid.Dur,
		DealID:             bid.DealID,
		IURL:               bid.IURL,
	}, true
}
