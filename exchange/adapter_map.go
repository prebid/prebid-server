package exchange

import (
	"net/http"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adform"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adtelligent"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/appnexus"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/audienceNetwork"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/brightroll"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/conversant"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/eplanning"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/indexExchange"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/lifestreet"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/openx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pubmatic"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pulsepoint"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rubicon"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/somoaudience"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sovrn"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client, cfg *config.Configuration) map[openrtb_ext.BidderName]adaptedBidder {
	return map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAdform:      adaptBidder(adform.NewAdformBidder(client, cfg.Adapters["adform"].Endpoint), client),
		openrtb_ext.BidderAdtelligent: adaptBidder(adtelligent.NewAdtelligentBidder(client), client),
		openrtb_ext.BidderAppnexus:    adaptBidder(appnexus.NewAppNexusBidder(client, cfg.Adapters["appnexus"].Endpoint), client),
		openrtb_ext.BidderBrightroll:  adaptBidder(brightroll.NewBrightrollBidder(cfg.Adapters["brightroll"].Endpoint), client),
		// TODO #267: Upgrade the Conversant adapter
		openrtb_ext.BidderConversant: adaptLegacyAdapter(conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["conversant"].Endpoint)),
		openrtb_ext.BidderEPlanning:  adaptBidder(eplanning.NewEPlanningBidder(client, cfg.Adapters["eplanning"].Endpoint), client),
		// TODO #211: Upgrade the Facebook adapter
		openrtb_ext.BidderFacebook: adaptLegacyAdapter(audienceNetwork.NewAdapterFromFacebook(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["facebook"].PlatformID)),
		// TODO #212: Upgrade the Index adapter
		openrtb_ext.BidderIndex: adaptLegacyAdapter(indexExchange.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["indexexchange"].Endpoint)),
		// TODO #213: Upgrade the Lifestreet adapter
		openrtb_ext.BidderLifestreet: adaptLegacyAdapter(lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig)),
		openrtb_ext.BidderOpenx:      adaptBidder(openx.NewOpenxBidder(), client),
		// TODO #214: Upgrade the Pubmatic adapter
		openrtb_ext.BidderPubmatic: adaptBidder(pubmatic.NewPubmaticBidder(client, cfg.Adapters["pubmatic"].Endpoint), client),
		// TODO #215: Upgrade the Pulsepoint adapter
		openrtb_ext.BidderPulsepoint: adaptLegacyAdapter(pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pulsepoint"].Endpoint)),
		openrtb_ext.BidderRubicon: adaptBidder(rubicon.NewRubiconBidder(client, cfg.Adapters["rubicon"].Endpoint, cfg.Adapters["rubicon"].XAPI.Username,
			cfg.Adapters["rubicon"].XAPI.Password, cfg.Adapters["rubicon"].XAPI.Tracker), client),
		openrtb_ext.BidderSomoaudience: adaptBidder(somoaudience.NewSomoaudienceBidder(), client),
		openrtb_ext.BidderSovrn:        adaptBidder(sovrn.NewSovrnBidder(client, cfg.Adapters["sovrn"].Endpoint), client),
	}
}
