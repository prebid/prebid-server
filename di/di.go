package di

import (
	"github.com/prebid/prebid-server/v3/di/interfaces"
	"github.com/prebid/prebid-server/v3/di/providers/log"
)

var Log interfaces.ILogger = log.ProvideLogger()
