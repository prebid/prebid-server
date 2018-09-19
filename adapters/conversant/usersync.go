package conversant

import (
	"net/url"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func NewConversantSyncer(cfg *config.Configuration) usersync.Usersyncer {
	usersyncURL := cfg.Adapters[string(openrtb_ext.BidderConversant)].UserSyncURL
	redirectURI := url.QueryEscape(cfg.ExternalURL) + "%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D"
	return adapters.NewSyncer("conversant", 24, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeRedirect)
}
