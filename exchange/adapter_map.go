package exchange

import (
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/adtelligent"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/beachfront"
	"github.com/prebid/prebid-server/adapters/brightroll"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/eplanning"
	"github.com/prebid/prebid-server/adapters/indexExchange"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/openx"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/adapters/somoaudience"
	"github.com/prebid/prebid-server/adapters/sovrn"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client, cfg *config.Configuration, infos adapters.BidderInfos) map[openrtb_ext.BidderName]adaptedBidder {
	return map[openrtb_ext.BidderName]adaptedBidder{
		openrtb_ext.BidderAdform:      adaptBidder(adapters.EnforceBidderInfo(adform.NewAdformBidder(client, cfg.Adapters[string(openrtb_ext.BidderAdform)].Endpoint), infos[string(openrtb_ext.BidderAdform)]), client),
		openrtb_ext.BidderAdtelligent: adaptBidder(adapters.EnforceBidderInfo(adtelligent.NewAdtelligentBidder(cfg.Adapters[string(openrtb_ext.BidderAdtelligent)].Endpoint), infos[string(openrtb_ext.BidderAdtelligent)]), client),
		openrtb_ext.BidderAppnexus:    adaptBidder(adapters.EnforceBidderInfo(appnexus.NewAppNexusBidder(client, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint), infos[string(openrtb_ext.BidderAppnexus)]), client),
		// TODO #615: Update the config setup so that the Beachfront URLs can be configured, and use those in TestRaceIntegration in exchange_test.go
		openrtb_ext.BidderBeachfront: adaptBidder(adapters.EnforceBidderInfo(beachfront.NewBeachfrontBidder(), infos[string(openrtb_ext.BidderBeachfront)]), client),
		openrtb_ext.BidderBrightroll: adaptBidder(adapters.EnforceBidderInfo(brightroll.NewBrightrollBidder(cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint), infos[string(openrtb_ext.BidderBrightroll)]), client),
		// TODO #267: Upgrade the Conversant adapter
		openrtb_ext.BidderConversant: adaptLegacyAdapter(conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderConversant)].Endpoint)),
		openrtb_ext.BidderEPlanning:  adaptBidder(adapters.EnforceBidderInfo(eplanning.NewEPlanningBidder(client, cfg.Adapters[string(openrtb_ext.BidderEPlanning)].Endpoint), infos[string(openrtb_ext.BidderEPlanning)]), client),
		// TODO #211: Upgrade the Facebook adapter
		openrtb_ext.BidderFacebook: adaptLegacyAdapter(audienceNetwork.NewAdapterFromFacebook(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].PlatformID)),
		// TODO #212: Upgrade the Index adapter
		openrtb_ext.BidderIndex: adaptLegacyAdapter(indexExchange.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIndex))].Endpoint)),
		// TODO #213: Upgrade the Lifestreet adapter
		openrtb_ext.BidderLifestreet: adaptLegacyAdapter(lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderLifestreet)].Endpoint)),
		openrtb_ext.BidderOpenx:      adaptBidder(adapters.EnforceBidderInfo(openx.NewOpenxBidder(cfg.Adapters[string(openrtb_ext.BidderOpenx)].Endpoint), infos[string(openrtb_ext.BidderOpenx)]), client),
		// TODO #214: Upgrade the Pubmatic adapter
		openrtb_ext.BidderPubmatic: adaptLegacyAdapter(pubmatic.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint)),
		// TODO #215: Upgrade the Pulsepoint adapter
		openrtb_ext.BidderPulsepoint: adaptLegacyAdapter(pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPulsepoint)].Endpoint)),
		openrtb_ext.BidderRubicon: adaptBidder(adapters.EnforceBidderInfo(
			rubicon.NewRubiconBidder(
				client,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password,
				cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),
			infos[string(openrtb_ext.BidderRubicon)]), client),
		openrtb_ext.BidderSomoaudience: adaptBidder(adapters.EnforceBidderInfo(somoaudience.NewSomoaudienceBidder(cfg.Adapters[string(openrtb_ext.BidderSomoaudience)].Endpoint), infos[string(openrtb_ext.BidderSomoaudience)]), client),
		openrtb_ext.BidderSovrn:        adaptBidder(adapters.EnforceBidderInfo(sovrn.NewSovrnBidder(client, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint), infos[string(openrtb_ext.BidderSovrn)]), client),
	}
}
