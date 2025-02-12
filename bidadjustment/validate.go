package bidadjustment

import (
	"math"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Validate checks whether all provided bid adjustments are valid or not against the requirements defined in the issue
func Validate(bidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments) bool {
	if bidAdjustments == nil {
		return true
	}
	if !validateForMediaType(bidAdjustments.MediaType.Banner) {
		return false
	}
	if !validateForMediaType(bidAdjustments.MediaType.Audio) {
		return false
	}
	if !validateForMediaType(bidAdjustments.MediaType.VideoInstream) {
		return false
	}
	if !validateForMediaType(bidAdjustments.MediaType.VideoOutstream) {
		return false
	}
	if !validateForMediaType(bidAdjustments.MediaType.Native) {
		return false
	}
	if !validateForMediaType(bidAdjustments.MediaType.WildCard) {
		return false
	}
	return true
}

func validateForMediaType(bidAdj map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID) bool {
	for bidderName := range bidAdj {
		if bidAdj[bidderName] == nil {
			return false
		}
		for dealID := range bidAdj[bidderName] {
			if bidAdj[bidderName][dealID] == nil {
				return false
			}
			for _, adjustment := range bidAdj[bidderName][dealID] {
				if !validateAdjustment(adjustment) {
					return false
				}
			}
		}
	}
	return true
}

func validateAdjustment(adjustment openrtb_ext.Adjustment) bool {
	switch adjustment.Type {
	case AdjustmentTypeCPM:
		if adjustment.Currency != "" && adjustment.Value >= 0 && adjustment.Value < math.MaxFloat64 {
			return true
		}
	case AdjustmentTypeMultiplier:
		if adjustment.Value >= 0 && adjustment.Value < 100 {
			return true
		}
	case AdjustmentTypeStatic:
		if adjustment.Currency != "" && adjustment.Value >= 0 && adjustment.Value < math.MaxFloat64 {
			return true
		}
	}
	return false
}
