package rhythmone

import (
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

func NewRhythmoneSyncer(cfg *config.Configuration) usersync.Usersyncer {
	externalURL := strings.TrimRight(cfg.ExternalURL, "/")
	redirectURI := url.QueryEscape(externalURL) + "%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D%5BRX_UUID%5D"
	usersyncURL := cfg.Adapters[string(openrtb_ext.BidderRhythmone)].UserSyncURL
	return adapters.NewSyncer("rhythmone", 36, adapters.ResolveMacros(usersyncURL+redirectURI), adapters.SyncTypeRedirect)

}
