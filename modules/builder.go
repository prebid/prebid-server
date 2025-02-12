package modules

import (
	prebidOrtb2blocking "github.com/prebid/prebid-server/v3/modules/prebid/ortb2blocking"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	ret := ModuleBuilders{
		"prebid": {
			"ortb2blocking": prebidOrtb2blocking.Builder,
		},
	}
	addFiftyonedegreesDevicedetection(ret)
	return ret
}
