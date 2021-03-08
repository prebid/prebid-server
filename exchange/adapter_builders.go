package exchange

import (
	"github.com/prebid/prebid-server/adapters"
	ttx "github.com/prebid/prebid-server/adapters/33across"
	"github.com/prebid/prebid-server/adapters/acuityads"
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/adgeneration"
	"github.com/prebid/prebid-server/adapters/adhese"
	"github.com/prebid/prebid-server/adapters/adkernel"
	"github.com/prebid/prebid-server/adapters/adkernelAdn"
	"github.com/prebid/prebid-server/adapters/adman"
	"github.com/prebid/prebid-server/adapters/admixer"
	"github.com/prebid/prebid-server/adapters/adocean"
	"github.com/prebid/prebid-server/adapters/adoppler"
	"github.com/prebid/prebid-server/adapters/adot"
	"github.com/prebid/prebid-server/adapters/adpone"
	"github.com/prebid/prebid-server/adapters/adprime"
	"github.com/prebid/prebid-server/adapters/adtarget"
	"github.com/prebid/prebid-server/adapters/adtelligent"
	"github.com/prebid/prebid-server/adapters/advangelists"
	"github.com/prebid/prebid-server/adapters/adyoulike"
	"github.com/prebid/prebid-server/adapters/aja"
	"github.com/prebid/prebid-server/adapters/amx"
	"github.com/prebid/prebid-server/adapters/applogy"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/avocet"
	"github.com/prebid/prebid-server/adapters/beachfront"
	"github.com/prebid/prebid-server/adapters/beintoo"
	"github.com/prebid/prebid-server/adapters/between"
	"github.com/prebid/prebid-server/adapters/brightroll"
	"github.com/prebid/prebid-server/adapters/colossus"
	"github.com/prebid/prebid-server/adapters/connectad"
	"github.com/prebid/prebid-server/adapters/consumable"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/cpmstar"
	"github.com/prebid/prebid-server/adapters/datablocks"
	"github.com/prebid/prebid-server/adapters/decenterads"
	"github.com/prebid/prebid-server/adapters/deepintent"
	"github.com/prebid/prebid-server/adapters/dmx"
	"github.com/prebid/prebid-server/adapters/emx_digital"
	"github.com/prebid/prebid-server/adapters/engagebdr"
	"github.com/prebid/prebid-server/adapters/eplanning"
	"github.com/prebid/prebid-server/adapters/epom"
	"github.com/prebid/prebid-server/adapters/gamma"
	"github.com/prebid/prebid-server/adapters/gamoshi"
	"github.com/prebid/prebid-server/adapters/grid"
	"github.com/prebid/prebid-server/adapters/gumgum"
	"github.com/prebid/prebid-server/adapters/improvedigital"
	"github.com/prebid/prebid-server/adapters/inmobi"
	"github.com/prebid/prebid-server/adapters/invibes"
	"github.com/prebid/prebid-server/adapters/ix"
	"github.com/prebid/prebid-server/adapters/jixie"
	"github.com/prebid/prebid-server/adapters/kidoz"
	"github.com/prebid/prebid-server/adapters/krushmedia"
	"github.com/prebid/prebid-server/adapters/kubient"
	"github.com/prebid/prebid-server/adapters/lockerdome"
	"github.com/prebid/prebid-server/adapters/logicad"
	"github.com/prebid/prebid-server/adapters/lunamedia"
	"github.com/prebid/prebid-server/adapters/marsmedia"
	"github.com/prebid/prebid-server/adapters/mgid"
	"github.com/prebid/prebid-server/adapters/mobfoxpb"
	"github.com/prebid/prebid-server/adapters/mobilefuse"
	"github.com/prebid/prebid-server/adapters/nanointeractive"
	"github.com/prebid/prebid-server/adapters/ninthdecimal"
	"github.com/prebid/prebid-server/adapters/nobid"
	"github.com/prebid/prebid-server/adapters/onetag"
	"github.com/prebid/prebid-server/adapters/openx"
	"github.com/prebid/prebid-server/adapters/orbidder"
	"github.com/prebid/prebid-server/adapters/pangle"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pubnative"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/revcontent"
	"github.com/prebid/prebid-server/adapters/rhythmone"
	"github.com/prebid/prebid-server/adapters/rtbhouse"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/adapters/sharethrough"
	"github.com/prebid/prebid-server/adapters/silvermob"
	"github.com/prebid/prebid-server/adapters/smaato"
	"github.com/prebid/prebid-server/adapters/smartadserver"
	"github.com/prebid/prebid-server/adapters/smartrtb"
	"github.com/prebid/prebid-server/adapters/smartyads"
	"github.com/prebid/prebid-server/adapters/somoaudience"
	"github.com/prebid/prebid-server/adapters/sonobi"
	"github.com/prebid/prebid-server/adapters/sovrn"
	"github.com/prebid/prebid-server/adapters/synacormedia"
	"github.com/prebid/prebid-server/adapters/tappx"
	"github.com/prebid/prebid-server/adapters/telaria"
	"github.com/prebid/prebid-server/adapters/triplelift"
	"github.com/prebid/prebid-server/adapters/triplelift_native"
	"github.com/prebid/prebid-server/adapters/ucfunnel"
	"github.com/prebid/prebid-server/adapters/unicorn"
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
	"github.com/prebid/prebid-server/openrtb_ext"
)

// Adapter registration is kept in this separate file for ease of use and to aid
// in resolving merge conflicts.

func newAdapterBuilders() map[openrtb_ext.BidderName]adapters.Builder {
	return map[openrtb_ext.BidderName]adapters.Builder{
		openrtb_ext.Bidder33Across:         ttx.Builder,
		openrtb_ext.BidderAcuityAds:        acuityads.Builder,
		openrtb_ext.BidderAdform:           adform.Builder,
		openrtb_ext.BidderAdgeneration:     adgeneration.Builder,
		openrtb_ext.BidderAdhese:           adhese.Builder,
		openrtb_ext.BidderAdkernel:         adkernel.Builder,
		openrtb_ext.BidderAdkernelAdn:      adkernelAdn.Builder,
		openrtb_ext.BidderAdman:            adman.Builder,
		openrtb_ext.BidderAdmixer:          admixer.Builder,
		openrtb_ext.BidderAdOcean:          adocean.Builder,
		openrtb_ext.BidderAdoppler:         adoppler.Builder,
		openrtb_ext.BidderAdpone:           adpone.Builder,
		openrtb_ext.BidderAdot:             adot.Builder,
		openrtb_ext.BidderAdprime:          adprime.Builder,
		openrtb_ext.BidderAdtarget:         adtarget.Builder,
		openrtb_ext.BidderAdtelligent:      adtelligent.Builder,
		openrtb_ext.BidderAdvangelists:     advangelists.Builder,
		openrtb_ext.BidderAdyoulike:        adyoulike.Builder,
		openrtb_ext.BidderAJA:              aja.Builder,
		openrtb_ext.BidderAMX:              amx.Builder,
		openrtb_ext.BidderApplogy:          applogy.Builder,
		openrtb_ext.BidderAppnexus:         appnexus.Builder,
		openrtb_ext.BidderAudienceNetwork:  audienceNetwork.Builder,
		openrtb_ext.BidderAvocet:           avocet.Builder,
		openrtb_ext.BidderBeachfront:       beachfront.Builder,
		openrtb_ext.BidderBeintoo:          beintoo.Builder,
		openrtb_ext.BidderBetween:          between.Builder,
		openrtb_ext.BidderBrightroll:       brightroll.Builder,
		openrtb_ext.BidderColossus:         colossus.Builder,
		openrtb_ext.BidderConnectAd:        connectad.Builder,
		openrtb_ext.BidderConsumable:       consumable.Builder,
		openrtb_ext.BidderConversant:       conversant.Builder,
		openrtb_ext.BidderCpmstar:          cpmstar.Builder,
		openrtb_ext.BidderDatablocks:       datablocks.Builder,
		openrtb_ext.BidderDecenterAds:      decenterads.Builder,
		openrtb_ext.BidderDeepintent:       deepintent.Builder,
		openrtb_ext.BidderDmx:              dmx.Builder,
		openrtb_ext.BidderEmxDigital:       emx_digital.Builder,
		openrtb_ext.BidderEngageBDR:        engagebdr.Builder,
		openrtb_ext.BidderEPlanning:        eplanning.Builder,
		openrtb_ext.BidderEpom:             epom.Builder,
		openrtb_ext.BidderGamma:            gamma.Builder,
		openrtb_ext.BidderGamoshi:          gamoshi.Builder,
		openrtb_ext.BidderGrid:             grid.Builder,
		openrtb_ext.BidderGumGum:           gumgum.Builder,
		openrtb_ext.BidderImprovedigital:   improvedigital.Builder,
		openrtb_ext.BidderInMobi:           inmobi.Builder,
		openrtb_ext.BidderInvibes:          invibes.Builder,
		openrtb_ext.BidderIx:               ix.Builder,
		openrtb_ext.BidderJixie:            jixie.Builder,
		openrtb_ext.BidderKidoz:            kidoz.Builder,
		openrtb_ext.BidderKrushmedia:       krushmedia.Builder,
		openrtb_ext.BidderKubient:          kubient.Builder,
		openrtb_ext.BidderLockerDome:       lockerdome.Builder,
		openrtb_ext.BidderLogicad:          logicad.Builder,
		openrtb_ext.BidderLunaMedia:        lunamedia.Builder,
		openrtb_ext.BidderMarsmedia:        marsmedia.Builder,
		openrtb_ext.BidderMediafuse:        adtelligent.Builder,
		openrtb_ext.BidderMgid:             mgid.Builder,
		openrtb_ext.BidderMobfoxpb:         mobfoxpb.Builder,
		openrtb_ext.BidderMobileFuse:       mobilefuse.Builder,
		openrtb_ext.BidderNanoInteractive:  nanointeractive.Builder,
		openrtb_ext.BidderNinthDecimal:     ninthdecimal.Builder,
		openrtb_ext.BidderNoBid:            nobid.Builder,
		openrtb_ext.BidderOneTag:           onetag.Builder,
		openrtb_ext.BidderOpenx:            openx.Builder,
		openrtb_ext.BidderOrbidder:         orbidder.Builder,
		openrtb_ext.BidderPangle:           pangle.Builder,
		openrtb_ext.BidderPubmatic:         pubmatic.Builder,
		openrtb_ext.BidderPubnative:        pubnative.Builder,
		openrtb_ext.BidderPulsepoint:       pulsepoint.Builder,
		openrtb_ext.BidderRevcontent:       revcontent.Builder,
		openrtb_ext.BidderRhythmone:        rhythmone.Builder,
		openrtb_ext.BidderRTBHouse:         rtbhouse.Builder,
		openrtb_ext.BidderRubicon:          rubicon.Builder,
		openrtb_ext.BidderSharethrough:     sharethrough.Builder,
		openrtb_ext.BidderSilverMob:        silvermob.Builder,
		openrtb_ext.BidderSmaato:           smaato.Builder,
		openrtb_ext.BidderSmartAdserver:    smartadserver.Builder,
		openrtb_ext.BidderSmartRTB:         smartrtb.Builder,
		openrtb_ext.BidderSmartyAds:        smartyads.Builder,
		openrtb_ext.BidderSomoaudience:     somoaudience.Builder,
		openrtb_ext.BidderSonobi:           sonobi.Builder,
		openrtb_ext.BidderSovrn:            sovrn.Builder,
		openrtb_ext.BidderSynacormedia:     synacormedia.Builder,
		openrtb_ext.BidderTappx:            tappx.Builder,
		openrtb_ext.BidderTelaria:          telaria.Builder,
		openrtb_ext.BidderTriplelift:       triplelift.Builder,
		openrtb_ext.BidderTripleliftNative: triplelift_native.Builder,
		openrtb_ext.BidderTrustX:           grid.Builder,
		openrtb_ext.BidderUcfunnel:         ucfunnel.Builder,
		openrtb_ext.BidderUnicorn:          unicorn.Builder,
		openrtb_ext.BidderUnruly:           unruly.Builder,
		openrtb_ext.BidderValueImpression:  valueimpression.Builder,
		openrtb_ext.BidderVerizonMedia:     verizonmedia.Builder,
		openrtb_ext.BidderVisx:             visx.Builder,
		openrtb_ext.BidderVrtcal:           vrtcal.Builder,
		openrtb_ext.BidderYeahmobi:         yeahmobi.Builder,
		openrtb_ext.BidderYieldlab:         yieldlab.Builder,
		openrtb_ext.BidderYieldmo:          yieldmo.Builder,
		openrtb_ext.BidderYieldone:         yieldone.Builder,
		openrtb_ext.BidderZeroClickFraud:   zeroclickfraud.Builder,
	}
}
