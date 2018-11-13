package gumgum

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"

	"net/url"
	"strings"
)

func NewGumGumSyncer(cfg *config.Configuration) usersync.Usersyncer {
	externalURL := strings.TrimRight(cfg.ExternalURL, "/")
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dgumgum%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D"
	usersyncURL := cfg.Adapters[string(openrtb_ext.BidderGumGum)].UserSyncURL
	return adapters.NewSyncer("gumgum", 61, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeIframe)
}
