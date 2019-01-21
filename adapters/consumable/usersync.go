package consumable

import (
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/url"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/usersync"
)

var VENDOR_ID uint16 = 65535 // TODO: What is the correct value

func NewConsumableSyncer(cfg *config.Configuration) usersync.Usersyncer {
	userSyncURL := cfg.Adapters[string(openrtb_ext.BidderConsumable)].UserSyncURL

	externalURL := strings.TrimRight(cfg.ExternalURL, "/")
	redirectURL := url.QueryEscape(externalURL) +
		"%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D"
		// i.e. /setuid?bidder=consumable&gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&uid=
		// serverbid will just append uid to the end

	return adapters.NewSyncer(
		"consumable",
		VENDOR_ID,
		adapters.ResolveMacros(userSyncURL+redirectURL),
		adapters.SyncTypeRedirect)
}
