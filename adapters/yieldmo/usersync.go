package yieldmo

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
	"net/url"
)

func NewYieldmoSyncer(cfg *config.Configuration) usersync.Usersyncer {
	usersyncURL := cfg.Adapters[string(openrtb_ext.BidderYieldmo)].UserSyncURL + "gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&redirectUri="
	redirectURI := url.QueryEscape(cfg.ExternalURL) + "%2Fsetuid%3Fbidder%3Dyieldmo%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24UID"
	return adapters.NewSyncer("yieldmo", 0, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeRedirect)
}
