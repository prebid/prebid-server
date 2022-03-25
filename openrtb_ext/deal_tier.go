package openrtb_ext

import (
	"encoding/json"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
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

	// imp.ext.{bidder}
	var impExt map[string]struct {
		DealTier *DealTier `json:"dealTier"`
	}
	if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
		return nil, err
	}
	for bidder, param := range impExt {
		if param.DealTier != nil {
			dealTiers[BidderName(bidder)] = *param.DealTier
		}
	}

	// imp.ext.prebid.{bidder}
	var impPrebidExt struct {
		Prebid struct {
			Bidders map[string]struct {
				DealTier *DealTier `json:"dealTier"`
			} `json:"bidder"`
		} `json:"prebid"`
	}
	if err := json.Unmarshal(imp.Ext, &impPrebidExt); err != nil {
		return nil, err
	}
	for bidder, param := range impPrebidExt.Prebid.Bidders {
		if param.DealTier != nil {
			dealTiers[BidderName(bidder)] = *param.DealTier
		}
	}

	return dealTiers, nil
}
