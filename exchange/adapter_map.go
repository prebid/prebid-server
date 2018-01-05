package exchange

import (
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/facebook"
	"github.com/prebid/prebid-server/adapters/index"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client, cfg *config.Configuration) map[openrtb_ext.BidderName]adaptedBidder {
	return map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAppnexus: adaptBidder(appnexus.NewAppNexusBidder(client, cfg.ExternalURL), client),
		// TODO #267: Upgrade the Conversant adapter
		openrtb_ext.BidderConversant: adaptLegacyAdapter(conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["conversant"].Endpoint, cfg.Adapters["conversant"].UserSyncURL, cfg.ExternalURL)),
		// TODO #211: Upgrade the Facebook adapter
		openrtb_ext.BidderFacebook: adaptLegacyAdapter(facebook.NewFacebookAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["facebook"].PlatformID, cfg.Adapters["facebook"].UserSyncURL)),
		// TODO #212: Upgrade the Index adapter
		openrtb_ext.BidderIndex: adaptLegacyAdapter(index.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["indexexchange"].Endpoint, cfg.Adapters["indexexchange"].UserSyncURL)),
		// TODO #213: Upgrade the Lifestreet adapter
		openrtb_ext.BidderLifestreet: adaptLegacyAdapter(lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL)),
		// TODO #214: Upgrade the Pubmatic adapter
		openrtb_ext.BidderPubmatic: adaptLegacyAdapter(pubmatic.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pubmatic"].Endpoint, cfg.ExternalURL)),
		// TODO #215: Upgrade the Pulsepoint adapter
		openrtb_ext.BidderPulsepoint: adaptLegacyAdapter(pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pulsepoint"].Endpoint, cfg.ExternalURL)),
		openrtb_ext.BidderRubicon: adaptBidder(rubicon.NewRubiconBidder(client, cfg.Adapters["rubicon"].Endpoint, cfg.Adapters["rubicon"].XAPI.Username,
			cfg.Adapters["rubicon"].XAPI.Password, cfg.Adapters["rubicon"].XAPI.Tracker, cfg.Adapters["rubicon"].UserSyncURL), client),
	}
}
