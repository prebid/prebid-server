package openx

import (
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewOpenxSyncer(cfg *config.Configuration) usersync.Usersyncer {
	externalURL := strings.TrimRight(cfg.ExternalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"
	return adapters.NewSyncer("openx", 69, adapters.ResolveMacros("https://rtb.openx.net/sync/prebid?r="+redirectURL), adapters.SyncTypeRedirect)
}
