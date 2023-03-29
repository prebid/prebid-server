package modules

import (
	prebidOrtb2blocking "github.com/prebid/prebid-server/modules/prebid/ortb2blocking"
	prebid_seat_moduleSeat "github.com/prebid/prebid-server/modules/prebid_seat_module/seat"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	return ModuleBuilders{
		"prebid": {
			"ortb2blocking": prebidOrtb2blocking.Builder,
		},
		"prebid_seat_module": {
			"seat": prebid_seat_moduleSeat.Builder,
		},
	}
}
