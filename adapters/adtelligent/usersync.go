package adtelligent

import (
	"net/url"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewAdtelligentSyncer(cfg *config.Configuration) usersync.Usersyncer {
	redirectURI := url.QueryEscape(cfg.ExternalURL) + "%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%7Buid%7D"
	usersyncURL := "//sync.adtelligent.com/csync?t=p&ep=0&redir="
	return adapters.NewSyncer("adtelligent", 0, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeRedirect)
}
