package bidadjustment

import (
	"math"

	"github.com/prebid/prebid-server/openrtb_ext"
)

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
	if !validateForMediaType(bidAdjustments.MediaType.Video) {
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

func validateForMediaType(bidAdjMap map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID) bool {
	for bidderName := range bidAdjMap {
		for dealId := range bidAdjMap[bidderName] {
			for _, adjustment := range bidAdjMap[bidderName][dealId] {
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
	case AdjustmentTypeCpm:
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
