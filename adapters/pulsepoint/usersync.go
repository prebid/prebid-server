package pulsepoint

import (
	"net/url"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

func NewPulsepointSyncer(cfg *config.Configuration) usersync.Usersyncer {
	redirectURI := url.QueryEscape(cfg.ExternalURL) + "%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%25%25VGUID%25%25"
	usersyncURL := "//bh.contextweb.com/rtset?pid=561205&ev=1&rurl="
	return adapters.NewSyncer("pulsepoint", 81, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeRedirect)
}
