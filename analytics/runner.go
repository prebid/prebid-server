package analytics

import (
	"github.com/prebid/prebid-server/v3/privacy"
)

type Runner interface {
	LogAuctionObject(*AuctionObject, privacy.ActivityControl)
	LogVideoObject(*VideoObject, privacy.ActivityControl)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject, privacy.ActivityControl)
	LogNotificationEventObject(*NotificationEvent, privacy.ActivityControl)
	Shutdown()
}
