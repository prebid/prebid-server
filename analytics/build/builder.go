package build

import (
	modulesAgma "github.com/prebid/prebid-server/v3/analytics/modules/agma"
	modulesFilelogger "github.com/prebid/prebid-server/v3/analytics/modules/filelogger"
	modulesPubstack "github.com/prebid/prebid-server/v3/analytics/modules/pubstack"
)

// builders returns mapping between analytics module name and its builder.
func builders() AnalyticsModuleBuilders {
	return AnalyticsModuleBuilders{
		"agma":       modulesAgma.Builder,
		"filelogger": modulesFilelogger.Builder,
		"pubstack":   modulesPubstack.Builder,
	}
}
