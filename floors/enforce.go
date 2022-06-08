package floors

import (
	"github.com/prebid/prebid-server/openrtb_ext"
)

func ShouldEnforceFloors(requestExt *openrtb_ext.PriceFloorRules, configEnforceRate int, f func(int) int) bool {

	if requestExt != nil && requestExt.Skipped != nil && *requestExt.Skipped {
		return false
	}

	if requestExt.Enforcement != nil && !requestExt.Enforcement.EnforcePBS {
		return requestExt.Enforcement.EnforcePBS
	}

	if requestExt.Enforcement != nil && requestExt.Enforcement.EnforceRate > 0 {
		configEnforceRate = requestExt.Enforcement.EnforceRate
	}

	shouldEnforce := configEnforceRate > f(ENFORCE_RATE_MAX+1)
	if requestExt.Enforcement == nil {
		requestExt.Enforcement = new(openrtb_ext.PriceFloorEnforcement)
	}
	requestExt.Enforcement.EnforcePBS = shouldEnforce

	return shouldEnforce
}
