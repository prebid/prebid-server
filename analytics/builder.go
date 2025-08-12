package analytics

import (
	prebidAgma "github.com/prebid/prebid-server/v3/analytics/prebid/agma"
	prebidFilelogger "github.com/prebid/prebid-server/v3/analytics/prebid/filelogger"
	prebidPubstack "github.com/prebid/prebid-server/v3/analytics/prebid/pubstack"
)

// builders returns mapping between analytics module name and its builder
// vendor and module names are chosen based on the module directory name
func builders() AnalyticsModuleBuilders {
	return AnalyticsModuleBuilders{
		"prebid": {
			"agma":       prebidAgma.Builder,
			"filelogger": prebidFilelogger.Builder,
			"pubstack":   prebidPubstack.Builder,
		},
	}
}
