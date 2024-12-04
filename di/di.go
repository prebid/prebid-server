package di

import (
	"github.com/prebid/prebid-server/v3/di/interfaces"
	"github.com/prebid/prebid-server/v3/di/providers"
)

var Log interfaces.ILogger = providers.ProvideLogger()
