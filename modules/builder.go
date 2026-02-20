package modules

import (
	fiftyonedegreesDevicedetection "github.com/prebid/prebid-server/v3/modules/fiftyonedegrees/devicedetection"
	prebidOrtb2blocking "github.com/prebid/prebid-server/v3/modules/prebid/ortb2blocking"
	prebidRulesengine "github.com/prebid/prebid-server/v3/modules/prebid/rulesengine"
	wurflDevicedetection "github.com/prebid/prebid-server/v3/modules/scientiamobile/wurfl_devicedetection"
	scope3Rtd "github.com/prebid/prebid-server/v3/modules/scope3/rtd"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	return ModuleBuilders{
		"fiftyonedegrees": {
			"devicedetection": fiftyonedegreesDevicedetection.Builder,
		},
		"prebid": {
			"ortb2blocking": prebidOrtb2blocking.Builder,
			"rulesengine":   prebidRulesengine.Builder,
		},
		"scientiamobile": {
			"wurfl_devicedetection": wurflDevicedetection.Builder,
		},
		"scope3": {
			"rtd": scope3Rtd.Builder,
		},
	}
}
