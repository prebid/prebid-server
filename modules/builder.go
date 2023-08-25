package modules

import (
	mytestMymodule "github.com/prebid/prebid-server/modules/mytest/mymodule"
	mytest2Mymodule2 "github.com/prebid/prebid-server/modules/mytest2/mymodule2"
	prebidOrtb2blocking "github.com/prebid/prebid-server/modules/prebid/ortb2blocking"
)

// builders returns mapping between module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() ModuleBuilders {
	return ModuleBuilders{
		"mytest": {
			"mymodule": mytestMymodule.Builder,
		},
		"mytest2": {
			"mymodule2": mytest2Mymodule2.Builder,
		},
		"prebid": {
			"ortb2blocking": prebidOrtb2blocking.Builder,
		},
	}
}
