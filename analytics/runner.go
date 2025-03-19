package analytics

import (
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/privacy"
)

type Runner interface {
	LogAuctionObject(*AuctionObject, privacy.ActivityControl, gdpr.PrivacyPolicy)
	LogVideoObject(*VideoObject, privacy.ActivityControl)
	LogCookieSyncObject(*CookieSyncObject)
	LogSetUIDObject(*SetUIDObject)
	LogAmpObject(*AmpObject, privacy.ActivityControl)
	LogNotificationEventObject(*NotificationEvent, privacy.ActivityControl)
	Shutdown()
}
