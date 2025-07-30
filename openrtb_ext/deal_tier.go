package openrtb_ext

import (
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// DealTier defines the configuration of a deal tier.
type DealTier struct {
	// Prefix specifies the beginning of the hb_pb_cat_dur targeting key value. Must be non-empty.
	Prefix string `json:"prefix"`

	// MinDealTier specifies the minimum deal priority value (inclusive) that must be met for the targeting
	// key value to be modified. Must be greater than 0.
	MinDealTier int `json:"minDealTier"`
}

// DealTierBidderMap defines a correlation between bidders and deal tiers.
type DealTierBidderMap map[BidderName]DealTier

// ReadDealTiersFromImp returns a map of bidder deal tiers read from the impression of an original request (not split / cleaned).
func ReadDealTiersFromImp(imp openrtb2.Imp) (DealTierBidderMap, error) {
	dealTiers := make(DealTierBidderMap)

	if len(imp.Ext) == 0 {
		return dealTiers, nil
	}

	var impPrebidExt struct {
		Prebid struct {
			Bidders map[string]struct {
				DealTier *DealTier `json:"dealTier"`
			} `json:"bidder"`
		} `json:"prebid"`
	}
	if err := jsonutil.Unmarshal(imp.Ext, &impPrebidExt); err != nil {
		return nil, err
	}
	for bidder, param := range impPrebidExt.Prebid.Bidders {
		if param.DealTier != nil {
			if bidderNormalized, bidderFound := NormalizeBidderName(bidder); bidderFound {
				dealTiers[bidderNormalized] = *param.DealTier
			} else {
				dealTiers[BidderName(bidder)] = *param.DealTier
			}
		}
	}

	return dealTiers, nil
}
