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
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// The newAdapterMap function is segregated to its own file to make it a simple and clean location for each Adapter
// to register itself. No wading through Exchange code to find it.

func newAdapterBuildersMap() map[openrtb_ext.BidderName]adapters.Builder {
	return map[openrtb_ext.BidderName]adapters.Builder{
		openrtb_ext.Bidder33Across:       ttx.Builder,
		openrtb_ext.BidderAdform:         adform.Builder,
		openrtb_ext.BidderAdgeneration:   adgeneration.Builder,
		openrtb_ext.BidderAdhese:         adhese.Builder,
		openrtb_ext.BidderAdkernel:       adkernel.Builder,
		openrtb_ext.BidderAdkernelAdn:    adkernelAdn.Builder,
		openrtb_ext.BidderAdman:          adman.Builder,
		openrtb_ext.BidderAdmixer:        admixer.Builder,
		openrtb_ext.BidderAdOcean:        adocean.Builder,
		openrtb_ext.BidderAdoppler:       adoppler.Builder,
		openrtb_ext.BidderAdpone:         adpone.Builder,
		openrtb_ext.BidderAdprime:        adprime.Builder,
		openrtb_ext.BidderAdtarget:       adtarget.Builder,
		openrtb_ext.BidderAdtelligent:    adtelligent.Builder,
		openrtb_ext.BidderAdvangelists:   advangelists.Builder,
		openrtb_ext.BidderAJA:            aja.Builder,
		openrtb_ext.BidderApplogy:        applogy.Builder,
		openrtb_ext.BidderAppnexus:       appnexus.Builder,
		openrtb_ext.BidderFacebook:       audienceNetwork.Builder,
		openrtb_ext.BidderAvocet:         avocet.Builder,
		openrtb_ext.BidderBeachfront:     beachfront.Builder,
		openrtb_ext.BidderBeintoo:        beintoo.Builder,
		openrtb_ext.BidderBrightroll:     brightroll.Builder,
		openrtb_ext.BidderConsumable:     consumable.Builder,
		openrtb_ext.BidderCpmstar:        cpmstar.Builder,
		openrtb_ext.BidderDatablocks:     datablocks.Builder,
		openrtb_ext.BidderDmx:            dmx.Builder,
		openrtb_ext.BidderEmxDigital:     emx_digital.Builder,
		openrtb_ext.BidderEngageBDR:      engagebdr.Builder,
		openrtb_ext.BidderEPlanning:      eplanning.Builder,
		openrtb_ext.BidderGamma:          gamma.Builder,
		openrtb_ext.BidderGamoshi:        gamoshi.Builder,
		openrtb_ext.BidderGrid:           grid.Builder,
		openrtb_ext.BidderGumGum:         gumgum.Builder,
		openrtb_ext.BidderImprovedigital: improvedigital.Builder,
		openrtb_ext.BidderKidoz:          kidoz.Builder,
		openrtb_ext.BidderKubient:        kubient.Builder,
		openrtb_ext.BidderLockerDome:     lockerdome.Builder,
		openrtb_ext.BidderLunaMedia:      lunamedia.Builder,
		openrtb_ext.BidderLogicad:        logicad.Builder,
		openrtb_ext.BidderMarsmedia:      marsmedia.Builder,
		openrtb_ext.BidderMgid:           mgid.Builder,

		// 42 done
	}
}

// 34 left

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
	bidders, errs := buildBidders(cfg.Adapters, infos)
	if len(errs) > 0 {
		return nil, errs
	}

	biddersLegacy := buildLegacyBidders(cfg.Adapters, infos)
	for k, v := range biddersLegacy {
		bidders[k] = v
	}

	wrapWithMiddleware(bidders)

	return bidders, nil
}

func buildBidders(adapterConfig map[string]config.Adapter, infos adapters.BidderInfos) (map[openrtb_ext.BidderName]adaptedBidder, []error) {
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

	for name, bidder := range bidders {
		// Clean out any disabled bidders
		if infos[string(name)].Status == adapters.StatusActive {
			allBidders[name] = adaptBidder(adapters.EnforceBidderInfo(bidder, infos[string(name)]), client, cfg, me, name)
		}
	}

	return bidders, errs
}

func buildLegacyBidders(adapterConfig map[string]config.Adapter, infos adapters.BidderInfos) map[openrtb_ext.BidderName]adaptedBidder {
	bidders := make(map[openrtb_ext.BidderName]adaptedBidder, 4)

	// Conversant
	if infos[string(openrtb_ext.BidderConversant)].Status == adapters.StatusActive {
		adapter := conversant.NewConversantLegacyAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderConversant)].Endpoint)
		bidders[openrtb_ext.BidderConversant] = adaptLegacyAdapter(adapter)
	}

	// Index
	if infos[string(openrtb_ext.BidderIx)].Status == adapters.StatusActive {
		adapter := ix.NewIxLegacyAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderIx)].Endpoint)
		bidders[openrtb_ext.BidderIx] = adaptLegacyAdapter(adapter)
	}

	// Lifestreet
	if infos[string(openrtb_ext.BidderLifestreet)].Status == adapters.StatusActive {
		adapter := lifestreet.NewLifestreetLegacyAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderLifestreet)].Endpoint)
		bidders[openrtb_ext.BidderLifestreet] = adaptLegacyAdapter(adapter)
	}

	// Pulsepoint
	if infos[string(openrtb_ext.BidderPulsepoint)].Status == adapters.StatusActive {
		adapter := pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, adapterConfig[string(openrtb_ext.BidderPulsepoint)].Endpoint)
		bidders[openrtb_ext.BidderPulsepoint] = adaptLegacyAdapter(adapter)
	}

	return bidders
}

func wrapWithMiddleware(bidders map[openrtb_ext.BidderName]adaptedBidder) {
	for name, bidder := range bidders {
		bidders[name] = addValidatedBidderMiddleware(bidder)
	}
}

// ActiveBidders get all active bidders
func ActiveBidders(biddersInfo adapters.BidderInfos, disabledBidders map[string]string) (bidderMap map[string]openrtb_ext.BidderName) {
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
