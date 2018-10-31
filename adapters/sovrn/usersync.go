package sovrn

import (
	"net/url"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func NewSovrnSyncer(cfg *config.Configuration) usersync.Usersyncer {
	externalURL := cfg.ExternalURL
	usersyncURL := cfg.Adapters[string(openrtb_ext.BidderSovrn)].UserSyncURL
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"
	return adapters.NewSyncer("sovrn", 13, adapters.ResolveMacros(usersyncURL+"redir="+redirectURI), adapters.SyncTypeRedirect)
}
