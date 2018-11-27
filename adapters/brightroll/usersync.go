package brightroll

import (
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func NewBrightrollSyncer(cfg *config.Configuration) usersync.Usersyncer {
	userSyncURL := cfg.Adapters[string(openrtb_ext.BidderBrightroll)].UserSyncURL
	externalURL := strings.TrimRight(cfg.ExternalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"
	return adapters.NewSyncer("brightroll", 25, adapters.ResolveMacros(userSyncURL+redirectURL), adapters.SyncTypeRedirect)
}
