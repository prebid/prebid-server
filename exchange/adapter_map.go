package exchange

import (
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/adkernelAdn"
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
	ortbBidders := map[openrtb_ext.BidderName]adapters.Bidder{
		openrtb_ext.BidderAdform:      adform.NewAdformBidder(client, cfg.Adapters[string(openrtb_ext.BidderAdform)].Endpoint),
		openrtb_ext.BidderAdkernelAdn: adkernelAdn.NewAdkernelAdnAdapter(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].Endpoint),
		openrtb_ext.BidderAdtelligent: adtelligent.NewAdtelligentBidder(cfg.Adapters[string(openrtb_ext.BidderAdtelligent)].Endpoint),
		openrtb_ext.BidderAppnexus:    appnexus.NewAppNexusBidder(client, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint),
		// TODO #615: Update the config setup so that the Beachfront URLs can be configured, and use those in TestRaceIntegration in exchange_test.go
		openrtb_ext.BidderBeachfront: beachfront.NewBeachfrontBidder(),
		openrtb_ext.BidderBrightroll: brightroll.NewBrightrollBidder(cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint),
		openrtb_ext.BidderEPlanning:  eplanning.NewEPlanningBidder(client, cfg.Adapters[string(openrtb_ext.BidderEPlanning)].Endpoint),
		openrtb_ext.BidderOpenx:      openx.NewOpenxBidder(cfg.Adapters[string(openrtb_ext.BidderOpenx)].Endpoint),
		openrtb_ext.BidderPubmatic:   pubmatic.NewPubmaticBidder(client, cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint),
		openrtb_ext.BidderRubicon: rubicon.NewRubiconBidder(
			client,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),
		openrtb_ext.BidderSomoaudience: somoaudience.NewSomoaudienceBidder(cfg.Adapters[string(openrtb_ext.BidderSomoaudience)].Endpoint),
		openrtb_ext.BidderSovrn:        sovrn.NewSovrnBidder(client, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint),
	}

	legacyBidders := map[openrtb_ext.BidderName]adapters.Adapter{
		// TODO #267: Upgrade the Conversant adapter
		openrtb_ext.BidderConversant: conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderConversant)].Endpoint),
		// TODO #211: Upgrade the Facebook adapter
		openrtb_ext.BidderFacebook: audienceNetwork.NewAdapterFromFacebook(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].PlatformID),
		// TODO #212: Upgrade the Index adapter
		openrtb_ext.BidderIndex: indexExchange.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIndex))].Endpoint),
		// TODO #213: Upgrade the Lifestreet adapter
		openrtb_ext.BidderLifestreet: lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderLifestreet)].Endpoint),
		// TODO #215: Upgrade the Pulsepoint adapter
		openrtb_ext.BidderPulsepoint: pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPulsepoint)].Endpoint),
	}

	allBidders := make(map[openrtb_ext.BidderName]adaptedBidder, len(ortbBidders)+len(legacyBidders))
	for name, bidder := range legacyBidders {
		allBidders[name] = adaptLegacyAdapter(bidder)
	}
	for name, bidder := range ortbBidders {
		allBidders[name] = adaptBidder(adapters.EnforceBidderInfo(bidder, infos[string(name)]), client)
	}
	return allBidders
}
