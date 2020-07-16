package exchange

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/prebid-server/pbsmetrics"

	"github.com/prebid/prebid-server/adapters"
	ttx "github.com/prebid/prebid-server/adapters/33across"
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/adgeneration"
	"github.com/prebid/prebid-server/adapters/adhese"
	"github.com/prebid/prebid-server/adapters/adkernel"
	"github.com/prebid/prebid-server/adapters/adkernelAdn"
	"github.com/prebid/prebid-server/adapters/adman"
	"github.com/prebid/prebid-server/adapters/admixer"
	"github.com/prebid/prebid-server/adapters/adocean"
	"github.com/prebid/prebid-server/adapters/adoppler"
	"github.com/prebid/prebid-server/adapters/adpone"
	"github.com/prebid/prebid-server/adapters/adtarget"
	"github.com/prebid/prebid-server/adapters/adtelligent"
	"github.com/prebid/prebid-server/adapters/advangelists"
	"github.com/prebid/prebid-server/adapters/aja"
	"github.com/prebid/prebid-server/adapters/applogy"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/avocet"
	"github.com/prebid/prebid-server/adapters/beachfront"
	"github.com/prebid/prebid-server/adapters/beintoo"
	"github.com/prebid/prebid-server/adapters/brightroll"
	"github.com/prebid/prebid-server/adapters/consumable"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/cpmstar"
	"github.com/prebid/prebid-server/adapters/datablocks"
	"github.com/prebid/prebid-server/adapters/dmx"
	"github.com/prebid/prebid-server/adapters/emx_digital"
	"github.com/prebid/prebid-server/adapters/engagebdr"
	"github.com/prebid/prebid-server/adapters/eplanning"
	"github.com/prebid/prebid-server/adapters/gamma"
	"github.com/prebid/prebid-server/adapters/gamoshi"
	"github.com/prebid/prebid-server/adapters/grid"
	"github.com/prebid/prebid-server/adapters/gumgum"
	"github.com/prebid/prebid-server/adapters/improvedigital"
	"github.com/prebid/prebid-server/adapters/ix"
	"github.com/prebid/prebid-server/adapters/kidoz"
	"github.com/prebid/prebid-server/adapters/kubient"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/lockerdome"
	"github.com/prebid/prebid-server/adapters/logicad"
	"github.com/prebid/prebid-server/adapters/lunamedia"
	"github.com/prebid/prebid-server/adapters/marsmedia"
	"github.com/prebid/prebid-server/adapters/mgid"
	"github.com/prebid/prebid-server/adapters/mobilefuse"
	"github.com/prebid/prebid-server/adapters/nanointeractive"
	"github.com/prebid/prebid-server/adapters/ninthdecimal"
	"github.com/prebid/prebid-server/adapters/openx"
	"github.com/prebid/prebid-server/adapters/orbidder"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pubnative"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rhythmone"
	"github.com/prebid/prebid-server/adapters/rtbhouse"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/adapters/sharethrough"
	"github.com/prebid/prebid-server/adapters/smartadserver"
	"github.com/prebid/prebid-server/adapters/smartrtb"
	"github.com/prebid/prebid-server/adapters/somoaudience"
	"github.com/prebid/prebid-server/adapters/sonobi"
	"github.com/prebid/prebid-server/adapters/sovrn"
	"github.com/prebid/prebid-server/adapters/synacormedia"
	"github.com/prebid/prebid-server/adapters/tappx"
	"github.com/prebid/prebid-server/adapters/telaria"
	"github.com/prebid/prebid-server/adapters/triplelift"
	"github.com/prebid/prebid-server/adapters/triplelift_native"
	"github.com/prebid/prebid-server/adapters/ucfunnel"
	"github.com/prebid/prebid-server/adapters/unruly"
	"github.com/prebid/prebid-server/adapters/valueimpression"
	"github.com/prebid/prebid-server/adapters/verizonmedia"
	"github.com/prebid/prebid-server/adapters/visx"
	"github.com/prebid/prebid-server/adapters/vrtcal"
	"github.com/prebid/prebid-server/adapters/yeahmobi"
	"github.com/prebid/prebid-server/adapters/yieldlab"
	"github.com/prebid/prebid-server/adapters/yieldmo"
	"github.com/prebid/prebid-server/adapters/yieldone"
	"github.com/prebid/prebid-server/adapters/zeroclickfraud"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterMap(client *http.Client, cfg *config.Configuration, infos adapters.BidderInfos, me pbsmetrics.MetricsEngine) map[openrtb_ext.BidderName]adaptedBidder {
	ortbBidders := map[openrtb_ext.BidderName]adapters.Bidder{
		openrtb_ext.Bidder33Across:     ttx.New33AcrossBidder(cfg.Adapters[string(openrtb_ext.Bidder33Across)].Endpoint),
		openrtb_ext.BidderAdform:       adform.NewAdformBidder(client, cfg.Adapters[string(openrtb_ext.BidderAdform)].Endpoint),
		openrtb_ext.BidderAdgeneration: adgeneration.NewAdgenerationAdapter(cfg.Adapters[string(openrtb_ext.BidderAdgeneration)].Endpoint),
		openrtb_ext.BidderAdhese:       adhese.NewAdheseBidder(cfg.Adapters[string(openrtb_ext.BidderAdhese)].Endpoint),
		openrtb_ext.BidderAdkernel:     adkernel.NewAdkernelAdapter(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernel))].Endpoint),
		openrtb_ext.BidderAdkernelAdn:  adkernelAdn.NewAdkernelAdnAdapter(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].Endpoint),
		openrtb_ext.BidderAdman:        adman.NewAdmanBidder(cfg.Adapters[string(openrtb_ext.BidderAdman)].Endpoint),
		openrtb_ext.BidderAdmixer:      admixer.NewAdmixerBidder(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdmixer))].Endpoint),
		openrtb_ext.BidderAdOcean:      adocean.NewAdOceanBidder(client, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdOcean))].Endpoint),
		openrtb_ext.BidderAdoppler:     adoppler.NewAdopplerBidder(cfg.Adapters[string(openrtb_ext.BidderAdoppler)].Endpoint),
		openrtb_ext.BidderAdpone:       adpone.NewAdponeBidder(cfg.Adapters[string(openrtb_ext.BidderAdpone)].Endpoint),
		openrtb_ext.BidderAdtarget:     adtarget.NewAdtargetBidder(cfg.Adapters[string(openrtb_ext.BidderAdtarget)].Endpoint),
		openrtb_ext.BidderAdtelligent:  adtelligent.NewAdtelligentBidder(cfg.Adapters[string(openrtb_ext.BidderAdtelligent)].Endpoint),
		openrtb_ext.BidderAdvangelists: advangelists.NewAdvangelistsBidder(cfg.Adapters[string(openrtb_ext.BidderAdvangelists)].Endpoint),
		openrtb_ext.BidderAJA:          aja.NewAJABidder(cfg.Adapters[string(openrtb_ext.BidderAJA)].Endpoint),
		openrtb_ext.BidderApplogy:      applogy.NewApplogyBidder(cfg.Adapters[string(openrtb_ext.BidderApplogy)].Endpoint),
		openrtb_ext.BidderAppnexus:     appnexus.NewAppNexusBidder(client, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].PlatformID),
		openrtb_ext.BidderAvocet:       avocet.NewAvocetAdapter(cfg.Adapters[string(openrtb_ext.BidderAvocet)].Endpoint),
		openrtb_ext.BidderBeachfront:   beachfront.NewBeachfrontBidder(cfg.Adapters[string(openrtb_ext.BidderBeachfront)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderBeachfront)].ExtraAdapterInfo),
		openrtb_ext.BidderBeintoo:      beintoo.NewBeintooBidder(cfg.Adapters[string(openrtb_ext.BidderBeintoo)].Endpoint),
		openrtb_ext.BidderBrightroll:   brightroll.NewBrightrollBidder(cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint),
		openrtb_ext.BidderConsumable:   consumable.NewConsumableBidder(cfg.Adapters[string(openrtb_ext.BidderConsumable)].Endpoint),
		openrtb_ext.BidderCpmstar:      cpmstar.NewCpmstarBidder(cfg.Adapters[string(openrtb_ext.BidderCpmstar)].Endpoint),
		openrtb_ext.BidderDatablocks:   datablocks.NewDatablocksBidder(cfg.Adapters[string(openrtb_ext.BidderDatablocks)].Endpoint),
		openrtb_ext.BidderDmx:          dmx.NewDmxBidder(cfg.Adapters[string(openrtb_ext.BidderDmx)].Endpoint),
		openrtb_ext.BidderEmxDigital:   emx_digital.NewEmxDigitalBidder(cfg.Adapters[string(openrtb_ext.BidderEmxDigital)].Endpoint),
		openrtb_ext.BidderEngageBDR:    engagebdr.NewEngageBDRBidder(client, cfg.Adapters[string(openrtb_ext.BidderEngageBDR)].Endpoint),
		openrtb_ext.BidderEPlanning:    eplanning.NewEPlanningBidder(client, cfg.Adapters[string(openrtb_ext.BidderEPlanning)].Endpoint),
		openrtb_ext.BidderFacebook: audienceNetwork.NewFacebookBidder(
			cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].PlatformID,
			cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].AppSecret),
		openrtb_ext.BidderGamma:           gamma.NewGammaBidder(cfg.Adapters[string(openrtb_ext.BidderGamma)].Endpoint),
		openrtb_ext.BidderGamoshi:         gamoshi.NewGamoshiBidder(cfg.Adapters[string(openrtb_ext.BidderGamoshi)].Endpoint),
		openrtb_ext.BidderGrid:            grid.NewGridBidder(cfg.Adapters[string(openrtb_ext.BidderGrid)].Endpoint),
		openrtb_ext.BidderGumGum:          gumgum.NewGumGumBidder(cfg.Adapters[string(openrtb_ext.BidderGumGum)].Endpoint),
		openrtb_ext.BidderImprovedigital:  improvedigital.NewImprovedigitalBidder(cfg.Adapters[string(openrtb_ext.BidderImprovedigital)].Endpoint),
		openrtb_ext.BidderKidoz:           kidoz.NewKidozBidder(cfg.Adapters[string(openrtb_ext.BidderKidoz)].Endpoint),
		openrtb_ext.BidderKubient:         kubient.NewKubientBidder(cfg.Adapters[string(openrtb_ext.BidderKubient)].Endpoint),
		openrtb_ext.BidderLockerDome:      lockerdome.NewLockerDomeBidder(cfg.Adapters[string(openrtb_ext.BidderLockerDome)].Endpoint),
		openrtb_ext.BidderLunaMedia:       lunamedia.NewLunaMediaBidder(cfg.Adapters[string(openrtb_ext.BidderLunaMedia)].Endpoint),
		openrtb_ext.BidderLogicad:         logicad.NewLogicadBidder(cfg.Adapters[string(openrtb_ext.BidderLogicad)].Endpoint),
		openrtb_ext.BidderMarsmedia:       marsmedia.NewMarsmediaBidder(cfg.Adapters[string(openrtb_ext.BidderMarsmedia)].Endpoint),
		openrtb_ext.BidderMgid:            mgid.NewMgidBidder(cfg.Adapters[string(openrtb_ext.BidderMgid)].Endpoint),
		openrtb_ext.BidderMobileFuse:      mobilefuse.NewMobileFuseBidder(cfg.Adapters[string(openrtb_ext.BidderMobileFuse)].Endpoint),
		openrtb_ext.BidderNanoInteractive: nanointeractive.NewNanoIneractiveBidder(cfg.Adapters[string(openrtb_ext.BidderNanoInteractive)].Endpoint),
		openrtb_ext.BidderNinthDecimal:    ninthdecimal.NewNinthDecimalBidder(cfg.Adapters[string(openrtb_ext.BidderNinthDecimal)].Endpoint),
		openrtb_ext.BidderOrbidder:        orbidder.NewOrbidderBidder(cfg.Adapters[string(openrtb_ext.BidderOrbidder)].Endpoint),
		openrtb_ext.BidderOpenx:           openx.NewOpenxBidder(cfg.Adapters[string(openrtb_ext.BidderOpenx)].Endpoint),
		openrtb_ext.BidderPubmatic:        pubmatic.NewPubmaticBidder(client, cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint),
		openrtb_ext.BidderPubnative:       pubnative.NewPubnativeBidder(cfg.Adapters[string(openrtb_ext.BidderPubnative)].Endpoint),
		openrtb_ext.BidderRhythmone:       rhythmone.NewRhythmoneBidder(cfg.Adapters[string(openrtb_ext.BidderRhythmone)].Endpoint),
		openrtb_ext.BidderRTBHouse:        rtbhouse.NewRTBHouseBidder(cfg.Adapters[string(openrtb_ext.BidderRTBHouse)].Endpoint),
		openrtb_ext.BidderRubicon: rubicon.NewRubiconBidder(
			client,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),

		openrtb_ext.BidderSharethrough:     sharethrough.NewSharethroughBidder(cfg.Adapters[string(openrtb_ext.BidderSharethrough)].Endpoint),
		openrtb_ext.BidderSmartadserver:    smartadserver.NewSmartadserverBidder(cfg.Adapters[string(openrtb_ext.BidderSmartadserver)].Endpoint),
		openrtb_ext.BidderSmartRTB:         smartrtb.NewSmartRTBBidder(cfg.Adapters[string(openrtb_ext.BidderSmartRTB)].Endpoint),
		openrtb_ext.BidderSomoaudience:     somoaudience.NewSomoaudienceBidder(cfg.Adapters[string(openrtb_ext.BidderSomoaudience)].Endpoint),
		openrtb_ext.BidderSonobi:           sonobi.NewSonobiBidder(client, cfg.Adapters[string(openrtb_ext.BidderSonobi)].Endpoint),
		openrtb_ext.BidderSovrn:            sovrn.NewSovrnBidder(client, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint),
		openrtb_ext.BidderSynacormedia:     synacormedia.NewSynacorMediaBidder(cfg.Adapters[string(openrtb_ext.BidderSynacormedia)].Endpoint),
		openrtb_ext.BidderTappx:            tappx.NewTappxBidder(client, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderTappx))].Endpoint),
		openrtb_ext.BidderTelaria:          telaria.NewTelariaBidder(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderTelaria))].Endpoint),
		openrtb_ext.BidderTriplelift:       triplelift.NewTripleliftBidder(client, cfg.Adapters[string(openrtb_ext.BidderTriplelift)].Endpoint),
		openrtb_ext.BidderTripleliftNative: triplelift_native.NewTripleliftNativeBidder(client, cfg.Adapters[string(openrtb_ext.BidderTripleliftNative)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderTripleliftNative)].ExtraAdapterInfo),
		openrtb_ext.BidderUcfunnel:         ucfunnel.NewUcfunnelBidder(cfg.Adapters[string(openrtb_ext.BidderUcfunnel)].Endpoint),
		openrtb_ext.BidderUnruly:           unruly.NewUnrulyBidder(client, cfg.Adapters[string(openrtb_ext.BidderUnruly)].Endpoint),
		openrtb_ext.BidderValueImpression:  valueimpression.NewValueImpressionBidder(cfg.Adapters[string(openrtb_ext.BidderValueImpression)].Endpoint),
		openrtb_ext.BidderYieldlab:         yieldlab.NewYieldlabBidder(cfg.Adapters[string(openrtb_ext.BidderYieldlab)].Endpoint),
		openrtb_ext.BidderVerizonMedia:     verizonmedia.NewVerizonMediaBidder(client, cfg.Adapters[string(openrtb_ext.BidderVerizonMedia)].Endpoint),
		openrtb_ext.BidderVisx:             visx.NewVisxBidder(cfg.Adapters[string(openrtb_ext.BidderVisx)].Endpoint),
		openrtb_ext.BidderVrtcal:           vrtcal.NewVrtcalBidder(cfg.Adapters[string(openrtb_ext.BidderVrtcal)].Endpoint),
		openrtb_ext.BidderYeahmobi:         yeahmobi.NewYeahmobiBidder(cfg.Adapters[string(openrtb_ext.BidderYeahmobi)].Endpoint),
		openrtb_ext.BidderYieldmo:          yieldmo.NewYieldmoBidder(cfg.Adapters[string(openrtb_ext.BidderYieldmo)].Endpoint),
		openrtb_ext.BidderYieldone:         yieldone.NewYieldoneBidder(cfg.Adapters[string(openrtb_ext.BidderYieldone)].Endpoint),
		openrtb_ext.BidderZeroClickFraud:   zeroclickfraud.NewZeroClickFraudBidder(cfg.Adapters[string(openrtb_ext.BidderZeroClickFraud)].Endpoint),
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
			allBidders[name] = adaptBidder(adapters.EnforceBidderInfo(bidder, infos[string(name)]), client, cfg, me, name)
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
