package analytics

import (
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/privacy"
)

type Runner interface {
	LogAuctionObject(*AuctionObject, privacy.ActivityControl, gdpr.PrivacyPolicy)
	LogVideoObject(*VideoObject, privacy.ActivityControl, gdpr.PrivacyPolicy)
	LogCookieSyncObject(*CookieSyncObject, privacy.ActivityControl, gdpr.PrivacyPolicy)
	LogSetUIDObject(*SetUIDObject, privacy.ActivityControl, gdpr.PrivacyPolicy)
	LogAmpObject(*AmpObject, privacy.ActivityControl, gdpr.PrivacyPolicy)
	LogNotificationEventObject(*NotificationEvent, privacy.ActivityControl)
	Shutdown()
}
