package bidadjustments

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
	return true
}

func findAndValidateAdjustment(bidAdjMap map[string]map[string][]openrtb_ext.Adjustments) bool {
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

func validateAdjustment(adjustment openrtb_ext.Adjustments) bool {
	switch adjustment.AdjType {
	case AdjTypeCpm:
		if adjustment.Currency != "" && adjustment.Value >= 0 && adjustment.Value < math.MaxFloat64 {
			return true
		}
	case AdjTypeMultiplier:
		if adjustment.Value >= 0 && adjustment.Value < 100 {
			return true
		}
		adjustment.Currency = ""
	case AdjTypeStatic:
		if adjustment.Currency != "" && adjustment.Value >= 0 && adjustment.Value < math.MaxFloat64 {
			return true
		}
	}
	return false
}
