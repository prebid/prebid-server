package modules

import (
	fiftyonedegreesDevicedetection "github.com/prebid/prebid-server/v2/modules/fiftyonedegrees/devicedetection"
	prebidOrtb2blocking "github.com/prebid/prebid-server/v2/modules/prebid/ortb2blocking"
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
		},
	}
}
