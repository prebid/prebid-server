package bidadjustment

import (
	"math"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func Validate(bidAdjustments *openrtb_ext.ExtRequestPrebidBidAdjustments) bool {
	if bidAdjustments == nil {
		return true
	}
	if bidAdjustments.MediaType.Banner != nil && !findAndValidateAdjustment(bidAdjustments.MediaType.Banner) {
		return false
	}
	if bidAdjustments.MediaType.Audio != nil && !findAndValidateAdjustment(bidAdjustments.MediaType.Audio) {
		return false
	}
	if bidAdjustments.MediaType.Video != nil && !findAndValidateAdjustment(bidAdjustments.MediaType.Video) {
		return false
	}
	if bidAdjustments.MediaType.Native != nil && !findAndValidateAdjustment(bidAdjustments.MediaType.Native) {
		return false
	}
	if bidAdjustments.MediaType.WildCard != nil && !findAndValidateAdjustment(bidAdjustments.MediaType.WildCard) {
		return false
	}
	return true
}

func findAndValidateAdjustment(bidAdjMap map[openrtb_ext.BidderName]openrtb_ext.AdjusmentsByDealID) bool {
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
