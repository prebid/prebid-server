package floors

import (
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func RequestHasFloors(bidRequest *openrtb2.BidRequest) bool {
	for i := range bidRequest.Imp {
		if bidRequest.Imp[i].BidFloor > 0 {
			return true
		}
	}
	return false
}

func ShouldEnforce(bidRequest *openrtb2.BidRequest, floorExt *openrtb_ext.PriceFloorRules, configEnforceRate int, f func(int) int) bool {

	if floorExt != nil && floorExt.Skipped != nil && *floorExt.Skipped {
		return !*floorExt.Skipped
	}

	if floorExt != nil && floorExt.Enforcement != nil && floorExt.Enforcement.EnforcePBS != nil && !*floorExt.Enforcement.EnforcePBS {
		return *floorExt.Enforcement.EnforcePBS
	}

	if floorExt != nil && floorExt.Enforcement != nil && floorExt.Enforcement.EnforceRate > 0 {
		configEnforceRate = floorExt.Enforcement.EnforceRate
	}

	shouldEnforce := configEnforceRate > f(enforceRateMax)
	if floorExt == nil {
		floorExt = new(openrtb_ext.PriceFloorRules)
	}

	if floorExt.Enforcement == nil {
		floorExt.Enforcement = new(openrtb_ext.PriceFloorEnforcement)
	}

	if floorExt.Enforcement.EnforcePBS == nil {
		floorExt.Enforcement.EnforcePBS = new(bool)
	}
	*floorExt.Enforcement.EnforcePBS = shouldEnforce
	return shouldEnforce
}
