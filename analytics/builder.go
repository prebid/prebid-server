package analytics

import (
	agma "github.com/prebid/prebid-server/v3/analytics/modules/agma"
	filelogger "github.com/prebid/prebid-server/v3/analytics/modules/filelogger"
	pubstack "github.com/prebid/prebid-server/v3/analytics/modules/pubstack"
)

// builders returns mapping between analytics module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() AnalyticsModuleBuilders {
	return AnalyticsModuleBuilders{
		"agma":       agma.Builder,
		"filelogger": filelogger.Builder,
		"pubstack":   pubstack.Builder,
	}
}

func Builders() AnalyticsModuleBuilders {
	return builders()
}
