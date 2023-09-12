package modules

import (
	"github.com/prebid/prebid-server/modules/mytest/mymodule"
	"github.com/prebid/prebid-server/modules/mytest2/mymodule2"
	prebidOrtb2blocking "github.com/prebid/prebid-server/modules/prebid/ortb2blocking"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	return ModuleBuilders{
		"prebid": {
			"ortb2blocking": prebidOrtb2blocking.Builder,
		},
		"mytest": {
			"mymodule": mymodule.Builder,
		},
		"mytest2": {
			"mymodule2": mymodule2.Builder,
		},
	}
}
