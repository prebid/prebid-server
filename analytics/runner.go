package analytics

import (
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/privacy"
)

type Runner interface {
	LogAuctionObject(*AuctionObject, privacy.ActivityControl, config.AccountPrivacy)
	LogVideoObject(*VideoObject, privacy.ActivityControl, config.AccountPrivacy)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject, privacy.ActivityControl, config.AccountPrivacy)
	LogNotificationEventObject(*NotificationEvent, privacy.ActivityControl)
}
