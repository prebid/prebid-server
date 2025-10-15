package modules

import (
	fiftyonedegreesDevicedetection "github.com/prebid/prebid-server/v3/modules/fiftyonedegrees/devicedetection"
	mileTrafficshaping "github.com/prebid/prebid-server/v3/modules/mile/trafficshaping"
	prebidOrtb2blocking "github.com/prebid/prebid-server/v3/modules/prebid/ortb2blocking"
	prebidRulesengine "github.com/prebid/prebid-server/v3/modules/prebid/rulesengine"
	scope3Rtd "github.com/prebid/prebid-server/v3/modules/scope3/rtd"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	return ModuleBuilders{
		"fiftyonedegrees": {
			"devicedetection": fiftyonedegreesDevicedetection.Builder,
		},
		"mile": {
			"trafficshaping": mileTrafficshaping.Builder,
		},
		"prebid": {
			"ortb2blocking": prebidOrtb2blocking.Builder,
			"rulesengine":   prebidRulesengine.Builder,
		},
		"scope3": {
			"rtd": scope3Rtd.Builder,
		},
	}
}
