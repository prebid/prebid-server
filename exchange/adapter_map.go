package exchange

import (
	"errors"
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
	"github.com/prebid/prebid-server/adapters/adprime"
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
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/ix"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterBuildersMap() map[openrtb_ext.BidderName]adapters.Builder {
	return map[openrtb_ext.BidderName]adapters.Builder{
		openrtb_ext.Bidder33Across:     ttx.Builder,
		openrtb_ext.BidderAdform:       adform.Builder,
		openrtb_ext.BidderAdgeneration: adgeneration.Builder,
		openrtb_ext.BidderAdhese:       adhese.Builder,
		openrtb_ext.BidderAdkernel:     adkernel.Builder,
		openrtb_ext.BidderAdkernelAdn:  adkernelAdn.Builder,
		openrtb_ext.BidderAdman:        adman.Builder,
		openrtb_ext.BidderAdmixer:      admixer.Builder,
		openrtb_ext.BidderAdOcean:      adocean.Builder,
		openrtb_ext.BidderAdoppler:     adoppler.Builder,
		openrtb_ext.BidderAdpone:       adpone.Builder,
		openrtb_ext.BidderAdprime:      adprime.Builder,
		openrtb_ext.BidderAdtarget:     adtarget.Builder,
		openrtb_ext.BidderAdtelligent:  adtelligent.Builder,
		openrtb_ext.BidderAdvangelists: advangelists.Builder,
		openrtb_ext.BidderAJA:          aja.Builder,
		openrtb_ext.BidderApplogy:      applogy.Builder,
		openrtb_ext.BidderAppnexus:     appnexus.Builder,
		openrtb_ext.BidderFacebook:     audienceNetwork.Builder,
		openrtb_ext.BidderAvocet:       avocet.Builder,
		openrtb_ext.BidderBeachfront:   beachfront.Builder,
		openrtb_ext.BidderBeintoo:      beintoo.Builder, // 22 done
	}
}

// 54 left
// 	openrtb_ext.BidderBrightroll:   brightroll.NewBrightrollBidder(cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderBrightroll)].ExtraAdapterInfo),
// 	openrtb_ext.BidderConsumable:   consumable.NewConsumableBidder(cfg.Adapters[string(openrtb_ext.BidderConsumable)].Endpoint),
// 	openrtb_ext.BidderCpmstar:      cpmstar.NewCpmstarBidder(cfg.Adapters[string(openrtb_ext.BidderCpmstar)].Endpoint),
// 	openrtb_ext.BidderDatablocks:   datablocks.NewDatablocksBidder(cfg.Adapters[string(openrtb_ext.BidderDatablocks)].Endpoint),
// 	openrtb_ext.BidderDmx:          dmx.NewDmxBidder(cfg.Adapters[string(openrtb_ext.BidderDmx)].Endpoint),
// 	openrtb_ext.BidderEmxDigital:   emx_digital.NewEmxDigitalBidder(cfg.Adapters[string(openrtb_ext.BidderEmxDigital)].Endpoint),
// 	openrtb_ext.BidderEngageBDR:    engagebdr.NewEngageBDRBidder(client, cfg.Adapters[string(openrtb_ext.BidderEngageBDR)].Endpoint),
// 	openrtb_ext.BidderEPlanning:    eplanning.NewEPlanningBidder(client, cfg.Adapters[string(openrtb_ext.BidderEPlanning)].Endpoint)
// 	openrtb_ext.BidderGamma:           gamma.NewGammaBidder(cfg.Adapters[string(openrtb_ext.BidderGamma)].Endpoint),
// 	openrtb_ext.BidderGamoshi:         gamoshi.NewGamoshiBidder(cfg.Adapters[string(openrtb_ext.BidderGamoshi)].Endpoint),
// 	openrtb_ext.BidderGrid:            grid.NewGridBidder(cfg.Adapters[string(openrtb_ext.BidderGrid)].Endpoint),
// 	openrtb_ext.BidderGumGum:          gumgum.NewGumGumBidder(cfg.Adapters[string(openrtb_ext.BidderGumGum)].Endpoint),
// 	openrtb_ext.BidderImprovedigital:  improvedigital.NewImprovedigitalBidder(cfg.Adapters[string(openrtb_ext.BidderImprovedigital)].Endpoint),
// 	openrtb_ext.BidderKidoz:           kidoz.NewKidozBidder(cfg.Adapters[string(openrtb_ext.BidderKidoz)].Endpoint),
// 	openrtb_ext.BidderKubient:         kubient.NewKubientBidder(cfg.Adapters[string(openrtb_ext.BidderKubient)].Endpoint),
// 	openrtb_ext.BidderLockerDome:      lockerdome.NewLockerDomeBidder(cfg.Adapters[string(openrtb_ext.BidderLockerDome)].Endpoint),
// 	openrtb_ext.BidderLunaMedia:       lunamedia.NewLunaMediaBidder(cfg.Adapters[string(openrtb_ext.BidderLunaMedia)].Endpoint),
// 	openrtb_ext.BidderLogicad:         logicad.NewLogicadBidder(cfg.Adapters[string(openrtb_ext.BidderLogicad)].Endpoint),
// 	openrtb_ext.BidderMarsmedia:       marsmedia.NewMarsmediaBidder(cfg.Adapters[string(openrtb_ext.BidderMarsmedia)].Endpoint),
// 	openrtb_ext.BidderMgid:            mgid.NewMgidBidder(cfg.Adapters[string(openrtb_ext.BidderMgid)].Endpoint),
// 	openrtb_ext.BidderMobileFuse:      mobilefuse.NewMobileFuseBidder(cfg.Adapters[string(openrtb_ext.BidderMobileFuse)].Endpoint),
// 	openrtb_ext.BidderNanoInteractive: nanointeractive.NewNanoIneractiveBidder(cfg.Adapters[string(openrtb_ext.BidderNanoInteractive)].Endpoint),
// 	openrtb_ext.BidderNinthDecimal:    ninthdecimal.NewNinthDecimalBidder(cfg.Adapters[string(openrtb_ext.BidderNinthDecimal)].Endpoint),
// 	openrtb_ext.BidderOrbidder:        orbidder.NewOrbidderBidder(cfg.Adapters[string(openrtb_ext.BidderOrbidder)].Endpoint),
// 	openrtb_ext.BidderOpenx:           openx.NewOpenxBidder(cfg.Adapters[string(openrtb_ext.BidderOpenx)].Endpoint),
// 	openrtb_ext.BidderPubmatic:        pubmatic.NewPubmaticBidder(client, cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint),
// 	openrtb_ext.BidderPubnative:       pubnative.NewPubnativeBidder(cfg.Adapters[string(openrtb_ext.BidderPubnative)].Endpoint),
// 	openrtb_ext.BidderRhythmone:       rhythmone.NewRhythmoneBidder(cfg.Adapters[string(openrtb_ext.BidderRhythmone)].Endpoint),
// 	openrtb_ext.BidderRTBHouse:        rtbhouse.NewRTBHouseBidder(cfg.Adapters[string(openrtb_ext.BidderRTBHouse)].Endpoint),
// 	openrtb_ext.BidderRubicon: rubicon.NewRubiconBidder(
// 		client,
// 		cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
// 		cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username,
// 		cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password,
// 		cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),

// 	openrtb_ext.BidderSharethrough:     sharethrough.NewSharethroughBidder(cfg.Adapters[string(openrtb_ext.BidderSharethrough)].Endpoint),
// 	openrtb_ext.BidderSmaato:           smaato.NewSmaatoBidder(cfg.Adapters[string(openrtb_ext.BidderSmaato)].Endpoint),
// 	openrtb_ext.BidderSmartadserver:    smartadserver.NewSmartadserverBidder(cfg.Adapters[string(openrtb_ext.BidderSmartadserver)].Endpoint),
// 	openrtb_ext.BidderSmartRTB:         smartrtb.NewSmartRTBBidder(cfg.Adapters[string(openrtb_ext.BidderSmartRTB)].Endpoint),
// 	openrtb_ext.BidderSomoaudience:     somoaudience.NewSomoaudienceBidder(cfg.Adapters[string(openrtb_ext.BidderSomoaudience)].Endpoint),
// 	openrtb_ext.BidderSonobi:           sonobi.NewSonobiBidder(client, cfg.Adapters[string(openrtb_ext.BidderSonobi)].Endpoint),
// 	openrtb_ext.BidderSovrn:            sovrn.NewSovrnBidder(client, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint),
// 	openrtb_ext.BidderSynacormedia:     synacormedia.NewSynacorMediaBidder(cfg.Adapters[string(openrtb_ext.BidderSynacormedia)].Endpoint),
// 	openrtb_ext.BidderTappx:            tappx.NewTappxBidder(client, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderTappx))].Endpoint),
// 	openrtb_ext.BidderTelaria:          telaria.NewTelariaBidder(cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderTelaria))].Endpoint),
// 	openrtb_ext.BidderTriplelift:       triplelift.NewTripleliftBidder(client, cfg.Adapters[string(openrtb_ext.BidderTriplelift)].Endpoint),
// 	openrtb_ext.BidderTripleliftNative: triplelift_native.NewTripleliftNativeBidder(client, cfg.Adapters[string(openrtb_ext.BidderTripleliftNative)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderTripleliftNative)].ExtraAdapterInfo),
// 	openrtb_ext.BidderUcfunnel:         ucfunnel.NewUcfunnelBidder(cfg.Adapters[string(openrtb_ext.BidderUcfunnel)].Endpoint),
// 	openrtb_ext.BidderUnruly:           unruly.NewUnrulyBidder(client, cfg.Adapters[string(openrtb_ext.BidderUnruly)].Endpoint),
// 	openrtb_ext.BidderValueImpression:  valueimpression.NewValueImpressionBidder(cfg.Adapters[string(openrtb_ext.BidderValueImpression)].Endpoint),
// 	openrtb_ext.BidderYieldlab:         yieldlab.NewYieldlabBidder(cfg.Adapters[string(openrtb_ext.BidderYieldlab)].Endpoint),
// 	openrtb_ext.BidderVerizonMedia:     verizonmedia.NewVerizonMediaBidder(client, cfg.Adapters[string(openrtb_ext.BidderVerizonMedia)].Endpoint),
// 	openrtb_ext.BidderVisx:             visx.NewVisxBidder(cfg.Adapters[string(openrtb_ext.BidderVisx)].Endpoint),
// 	openrtb_ext.BidderVrtcal:           vrtcal.NewVrtcalBidder(cfg.Adapters[string(openrtb_ext.BidderVrtcal)].Endpoint),
// 	openrtb_ext.BidderYeahmobi:         yeahmobi.NewYeahmobiBidder(cfg.Adapters[string(openrtb_ext.BidderYeahmobi)].Endpoint),
// 	openrtb_ext.BidderYieldmo:          yieldmo.NewYieldmoBidder(cfg.Adapters[string(openrtb_ext.BidderYieldmo)].Endpoint),
// 	openrtb_ext.BidderYieldone:         yieldone.NewYieldoneBidder(cfg.Adapters[string(openrtb_ext.BidderYieldone)].Endpoint),
// 	openrtb_ext.BidderZeroClickFraud:   zeroclickfraud.NewZeroClickFraudBidder(cfg.Adapters[string(openrtb_ext.BidderZeroClickFraud)].Endpoint),
// }

func newAdapterMap(client *http.Client, cfg *config.Configuration, infos adapters.BidderInfos, me pbsmetrics.MetricsEngine) (map[openrtb_ext.BidderName]adaptedBidder, []error) {
	adapterConfig := cfg.Adapters

	builders := newAdapterBuildersMap()
	bidders := make(map[openrtb_ext.BidderName]adapters.Bidder)

	var errs []error
	for bidder, cfg := range adapterConfig {
		bidderName := openrtb_ext.BidderName(strings.ToLower(bidder))

		// get builder, if error report it

		// build, if error report it

		if builder, ok := builders[bidderName]; ok {
			if adapter, err := builder(bidderName, cfg); err != nil {
				bidders[bidderName] = &adapters.MisconfiguredBidder{bidderName, err}
			} else {
				bidders[bidderName] = adapter
			}
		} else {
			errs = append(errs, errors.New("it failed"))
		}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	legacyBidders := map[openrtb_ext.BidderName]adapters.Adapter{
		// TODO #267: Upgrade the Conversant adapter
		openrtb_ext.BidderConversant: conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderConversant)].Endpoint),
		// TODO #212: Upgrade the Index adapter
		openrtb_ext.BidderIx: ix.NewIxAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderIx)].Endpoint),
		// TODO #213: Upgrade the Lifestreet adapter
		openrtb_ext.BidderLifestreet: lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderLifestreet)].Endpoint),
		// TODO #215: Upgrade the Pulsepoint adapter
		openrtb_ext.BidderPulsepoint: pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderPulsepoint)].Endpoint),
	}

	allBidders := make(map[openrtb_ext.BidderName]adaptedBidder, len(bidders)+len(legacyBidders))

	// Wrap legacy and openrtb Bidders behind a common interface, so that the Exchange doesn't need to concern
	// itself with the differences.
	for name, bidder := range legacyBidders {
		// Clean out any disabled bidders
		if infos[string(name)].Status == adapters.StatusActive {
			allBidders[name] = adaptLegacyAdapter(bidder)
		}
	}
	for name, bidder := range bidders {
		// Clean out any disabled bidders
		if infos[string(name)].Status == adapters.StatusActive {
			allBidders[name] = adaptBidder(adapters.EnforceBidderInfo(bidder, infos[string(name)]), client, cfg, me, name)
		}
	}

	// Apply any middleware used for global Bidder logic.
	for name, bidder := range allBidders {
		allBidders[name] = ensureValidBids(bidder)
	}

	return allBidders, nil
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
