package build

import (
	modulesAgma "github.com/prebid/prebid-server/v3/analytics/modules/agma"
	modulesFile "github.com/prebid/prebid-server/v3/analytics/modules/file"
	modulesPubstack "github.com/prebid/prebid-server/v3/analytics/modules/pubstack"
)

// builders returns mapping between analytics module name and its builder.
func builders() AnalyticsModuleBuilders {
	return AnalyticsModuleBuilders{
		"agma":     modulesAgma.Builder,
		"file":     modulesFile.Builder,
		"pubstack": modulesPubstack.Builder,
	}
}
