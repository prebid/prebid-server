package somoaudience

import (
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewSomoaudienceSyncer(cfg *config.Configuration) usersync.Usersyncer {
	externalURL := strings.TrimRight(cfg.ExternalURL, "/")
	redirectURL := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%24%7BUID%7D"
	usersyncURL := "//publisher-east.mobileadtrading.com/usersync?ru="
	return adapters.NewSyncer("somoaudience", 341, adapters.ResolveMacros(usersyncURL+redirectURL), adapters.SyncTypeRedirect)
}
