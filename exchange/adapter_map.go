package exchange

import (
	"fmt"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/telaria"
	"net/http"
	"strings"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/kubient"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	ttx "github.com/PubMatic-OpenWrap/prebid-server/adapters/33across"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adform"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adkernel"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adkernelAdn"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adpone"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adtelligent"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/advangelists"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/applogy"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/appnexus"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/audienceNetwork"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/beachfront"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/brightroll"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/consumable"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/conversant"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/datablocks"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/emx_digital"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/engagebdr"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/eplanning"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/gamma"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/gamoshi"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/grid"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/gumgum"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/improvedigital"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/ix"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/lifestreet"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/lockerdome"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/marsmedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/mgid"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/openx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pubmatic"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pubnative"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pulsepoint"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rhythmone"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rtbhouse"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rubicon"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sharethrough"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/somoaudience"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sonobi"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sovrn"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/spotx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/synacormedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/tappx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/triplelift"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/triplelift_native"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/unruly"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/verizonmedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/visx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/vrtcal"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/yieldmo"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client, cfg *config.Configuration, infos adapters.BidderInfos) map[openrtb_ext.BidderName]adaptedBidder {
	ortbBidders := map[openrtb_ext.BidderName]adapters.Bidder{
		openrtb_ext.Bidder33Across:     ttx.New33AcrossBidder(cfg.Adapters[string(openrtb_ext.Bidder33Across)].Endpoint),
		openrtb_ext.BidderAdform:       adform.NewAdformBidder(client, cfg.Adapters[string(openrtb_ext.BidderAdform)].Endpoint),
		openrtb_ext.BidderAdkernel:     adkernel.NewAdkernelAdapter(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernel))].Endpoint),
		openrtb_ext.BidderAdkernelAdn:  adkernelAdn.NewAdkernelAdnAdapter(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].Endpoint),
		openrtb_ext.BidderAdpone:       adpone.NewAdponeBidder(cfg.Adapters[string(openrtb_ext.BidderAdpone)].Endpoint),
		openrtb_ext.BidderAdtelligent:  adtelligent.NewAdtelligentBidder(cfg.Adapters[string(openrtb_ext.BidderAdtelligent)].Endpoint),
		openrtb_ext.BidderAdvangelists: advangelists.NewAdvangelistsBidder(cfg.Adapters[string(openrtb_ext.BidderAdvangelists)].Endpoint),
		openrtb_ext.BidderApplogy:      applogy.NewApplogyBidder(cfg.Adapters[string(openrtb_ext.BidderApplogy)].Endpoint),
		openrtb_ext.BidderAppnexus:     appnexus.NewAppNexusBidder(client, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].PlatformID),
		// TODO #615: Update the config setup so that the Beachfront URLs can be configured, and use those in TestRaceIntegration in exchange_test.go
		openrtb_ext.BidderBeachfront: beachfront.NewBeachfrontBidder(),
		openrtb_ext.BidderBrightroll: brightroll.NewBrightrollBidder(cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint),
		openrtb_ext.BidderConsumable: consumable.NewConsumableBidder(cfg.Adapters[string(openrtb_ext.BidderConsumable)].Endpoint),
		openrtb_ext.BidderDatablocks: datablocks.NewDatablocksBidder(cfg.Adapters[string(openrtb_ext.BidderDatablocks)].Endpoint),
		openrtb_ext.BidderEmxDigital: emx_digital.NewEmxDigitalBidder(cfg.Adapters[string(openrtb_ext.BidderEmxDigital)].Endpoint),
		openrtb_ext.BidderEngageBDR:  engagebdr.NewEngageBDRBidder(client, cfg.Adapters[string(openrtb_ext.BidderEngageBDR)].Endpoint),
		openrtb_ext.BidderEPlanning:  eplanning.NewEPlanningBidder(client, cfg.Adapters[string(openrtb_ext.BidderEPlanning)].Endpoint),
		openrtb_ext.BidderFacebook: audienceNetwork.NewFacebookBidder(
			client,
			cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].PlatformID,
			cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].AppSecret),
		openrtb_ext.BidderGamma:          gamma.NewGammaBidder(cfg.Adapters[string(openrtb_ext.BidderGamma)].Endpoint),
		openrtb_ext.BidderGamoshi:        gamoshi.NewGamoshiBidder(cfg.Adapters[string(openrtb_ext.BidderGamoshi)].Endpoint),
		openrtb_ext.BidderGrid:           grid.NewGridBidder(cfg.Adapters[string(openrtb_ext.BidderGrid)].Endpoint),
		openrtb_ext.BidderGumGum:         gumgum.NewGumGumBidder(cfg.Adapters[string(openrtb_ext.BidderGumGum)].Endpoint),
		openrtb_ext.BidderImprovedigital: improvedigital.NewImprovedigitalBidder(cfg.Adapters[string(openrtb_ext.BidderImprovedigital)].Endpoint),
		openrtb_ext.BidderKubient:        kubient.NewKubientBidder(cfg.Adapters[string(openrtb_ext.BidderKubient)].Endpoint),
		openrtb_ext.BidderLockerDome:     lockerdome.NewLockerDomeBidder(cfg.Adapters[string(openrtb_ext.BidderLockerDome)].Endpoint),
		openrtb_ext.BidderMarsmedia:      marsmedia.NewMarsmediaBidder(cfg.Adapters[string(openrtb_ext.BidderMarsmedia)].Endpoint),
		openrtb_ext.BidderMgid:           mgid.NewMgidBidder(cfg.Adapters[string(openrtb_ext.BidderMgid)].Endpoint),
		openrtb_ext.BidderOpenx:          openx.NewOpenxBidder(cfg.Adapters[string(openrtb_ext.BidderOpenx)].Endpoint),
		openrtb_ext.BidderPubmatic:       pubmatic.NewPubmaticBidder(client, cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint),
		openrtb_ext.BidderPubnative:      pubnative.NewPubnativeBidder(cfg.Adapters[string(openrtb_ext.BidderPubnative)].Endpoint),
		openrtb_ext.BidderRhythmone:      rhythmone.NewRhythmoneBidder(cfg.Adapters[string(openrtb_ext.BidderRhythmone)].Endpoint),
		openrtb_ext.BidderRTBHouse:       rtbhouse.NewRTBHouseBidder(cfg.Adapters[string(openrtb_ext.BidderRTBHouse)].Endpoint),
		openrtb_ext.BidderRubicon: rubicon.NewRubiconBidder(
			client,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),

		openrtb_ext.BidderSharethrough:     sharethrough.NewSharethroughBidder(cfg.Adapters[string(openrtb_ext.BidderSharethrough)].Endpoint),
		openrtb_ext.BidderSomoaudience:     somoaudience.NewSomoaudienceBidder(cfg.Adapters[string(openrtb_ext.BidderSomoaudience)].Endpoint),
		openrtb_ext.BidderSonobi:           sonobi.NewSonobiBidder(client, cfg.Adapters[string(openrtb_ext.BidderSonobi)].Endpoint),
		openrtb_ext.BidderSovrn:            sovrn.NewSovrnBidder(client, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint),
		openrtb_ext.BidderSynacormedia:     synacormedia.NewSynacorMediaBidder(cfg.Adapters[string(openrtb_ext.BidderSynacormedia)].Endpoint),
		openrtb_ext.BidderSpotX:            spotx.NewSpotxBidder(cfg.Adapters[string(openrtb_ext.BidderSpotX)].Endpoint),
		openrtb_ext.BidderTappx:            tappx.NewTappxBidder(client, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderTappx))].Endpoint),
		openrtb_ext.BidderTelaria:          telaria.NewTelariaBidder(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderTelaria))].Endpoint),
		openrtb_ext.BidderTriplelift:       triplelift.NewTripleliftBidder(client, cfg.Adapters[string(openrtb_ext.BidderTriplelift)].Endpoint),
		openrtb_ext.BidderTripleliftNative: triplelift_native.NewTripleliftNativeBidder(client, cfg.Adapters[string(openrtb_ext.BidderTripleliftNative)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderTripleliftNative)].ExtraAdapterInfo),
		openrtb_ext.BidderUnruly:           unruly.NewUnrulyBidder(client, cfg.Adapters[string(openrtb_ext.BidderUnruly)].Endpoint),
		openrtb_ext.BidderVerizonMedia:     verizonmedia.NewVerizonMediaBidder(client, cfg.Adapters[string(openrtb_ext.BidderVerizonMedia)].Endpoint),
		openrtb_ext.BidderVisx:             visx.NewVisxBidder(cfg.Adapters[string(openrtb_ext.BidderVisx)].Endpoint),
		openrtb_ext.BidderVrtcal:           vrtcal.NewVrtcalBidder(cfg.Adapters[string(openrtb_ext.BidderVrtcal)].Endpoint),
		openrtb_ext.BidderYieldmo:          yieldmo.NewYieldmoBidder(cfg.Adapters[string(openrtb_ext.BidderYieldmo)].Endpoint),
	}

	legacyBidders := map[openrtb_ext.BidderName]adapters.Adapter{
		// TODO #267: Upgrade the Conversant adapter
		openrtb_ext.BidderConversant: conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderConversant)].Endpoint),
		// TODO #212: Upgrade the Index adapter
		openrtb_ext.BidderIx: ix.NewIxAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIx))].Endpoint),
		// TODO #213: Upgrade the Lifestreet adapter
		openrtb_ext.BidderLifestreet: lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderLifestreet)].Endpoint),
		// TODO #215: Upgrade the Pulsepoint adapter
		openrtb_ext.BidderPulsepoint: pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPulsepoint)].Endpoint),
	}

	allBidders := make(map[openrtb_ext.BidderName]adaptedBidder, len(ortbBidders)+len(legacyBidders))

	// Wrap legacy and openrtb Bidders behind a common interface, so that the Exchange doesn't need to concern
	// itself with the differences.
	for name, bidder := range legacyBidders {
		// Clean out any disabled bidders
		if infos[string(name)].Status == adapters.StatusActive {
			allBidders[name] = adaptLegacyAdapter(bidder)
		}
	}
	for name, bidder := range ortbBidders {
		// Clean out any disabled bidders
		if infos[string(name)].Status == adapters.StatusActive {
			allBidders[name] = adaptBidder(adapters.EnforceBidderInfo(bidder, infos[string(name)]), client)
		}
	}

	// Apply any middleware used for global Bidder logic.
	for name, bidder := range allBidders {
		allBidders[name] = ensureValidBids(bidder)
	}

	return allBidders
}

// DisableBidders get all bidders but disabled ones
func DisableBidders(biddersInfo adapters.BidderInfos, disabledBidders map[string]string) (bidderMap map[string]openrtb_ext.BidderName) {
	bidderMap = make(map[string]openrtb_ext.BidderName)

	// Set up error messages for disabled bidders
	for name, infos := range biddersInfo {
		if infos.Status == adapters.StatusDisabled {
			disabledBidders[name] = fmt.Sprintf("Bidder \"%s\" has been disabled on this instance of Prebid Server. Please work with the PBS host to enable this bidder again.", name)
		} else {
			bidderMap[name] = openrtb_ext.BidderName(name)
		}
	}

	return bidderMap
}
