package exchange

import (
	"net/http"
	"strings"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adform"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adtelligent"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/appnexus"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/audienceNetwork"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/beachfront"
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
		openrtb_ext.BidderAdform:      adaptBidder(adform.NewAdformBidder(client, cfg.Adapters[string(openrtb_ext.BidderAdform)].Endpoint), client),
		openrtb_ext.BidderAdtelligent: adaptBidder(adtelligent.NewAdtelligentBidder(cfg.Adapters[string(openrtb_ext.BidderAdtelligent)].Endpoint), client),
		openrtb_ext.BidderAppnexus:    adaptBidder(appnexus.NewAppNexusBidder(client, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint), client),
		// TODO #615: Update the config setup so that the Beachfront URLs can be configured, and use those in TestRaceIntegration in exchange_test.go
		openrtb_ext.BidderBeachfront: adaptBidder(beachfront.NewBeachfrontBidder(), client),
		openrtb_ext.BidderBrightroll: adaptBidder(brightroll.NewBrightrollBidder(cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint), client),
		// TODO #267: Upgrade the Conversant adapter
		openrtb_ext.BidderConversant: adaptLegacyAdapter(conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderConversant)].Endpoint)),
		openrtb_ext.BidderEPlanning:  adaptBidder(eplanning.NewEPlanningBidder(client, cfg.Adapters[string(openrtb_ext.BidderEPlanning)].Endpoint), client),
		// TODO #211: Upgrade the Facebook adapter
		openrtb_ext.BidderFacebook: adaptLegacyAdapter(audienceNetwork.NewAdapterFromFacebook(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].PlatformID)),
		// TODO #212: Upgrade the Index adapter
		openrtb_ext.BidderIndex: adaptLegacyAdapter(indexExchange.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIndex))].Endpoint)),
		// TODO #213: Upgrade the Lifestreet adapter
		openrtb_ext.BidderLifestreet: adaptLegacyAdapter(lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderLifestreet)].Endpoint)),
		openrtb_ext.BidderOpenx:      adaptBidder(openx.NewOpenxBidder(cfg.Adapters[string(openrtb_ext.BidderOpenx)].Endpoint), client),
		// TODO #214: Upgrade the Pubmatic adapter
		openrtb_ext.BidderPubmatic: adaptBidder(pubmatic.NewPubmaticBidder(client, cfg.Adapters["pubmatic"].Endpoint), client),
		// TODO #215: Upgrade the Pulsepoint adapter
		openrtb_ext.BidderPulsepoint: adaptLegacyAdapter(pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPulsepoint)].Endpoint)),
		openrtb_ext.BidderRubicon: adaptBidder(
			rubicon.NewRubiconBidder(
				client,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),
			client),
		openrtb_ext.BidderSomoaudience: adaptBidder(somoaudience.NewSomoaudienceBidder(cfg.Adapters[string(openrtb_ext.BidderSomoaudience)].Endpoint), client),
		openrtb_ext.BidderSovrn:        adaptBidder(sovrn.NewSovrnBidder(client, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint), client),
	}
}
