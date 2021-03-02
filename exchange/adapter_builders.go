package exchange

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	ttx "github.com/PubMatic-OpenWrap/prebid-server/adapters/33across"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/acuityads"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adform"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adgeneration"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adhese"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adkernel"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adkernelAdn"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adman"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/admixer"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adocean"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adoppler"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adot"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adpone"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adprime"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adtarget"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adtelligent"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/advangelists"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/aja"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/amx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/applogy"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/appnexus"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/audienceNetwork"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/avocet"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/beachfront"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/beintoo"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/between"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/brightroll"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/colossus"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/connectad"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/consumable"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/conversant"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/cpmstar"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/datablocks"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/decenterads"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/deepintent"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/dmx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/emx_digital"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/engagebdr"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/eplanning"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/gamma"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/gamoshi"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/grid"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/gumgum"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/improvedigital"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/inmobi"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/invibes"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/ix"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/kidoz"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/krushmedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/kubient"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/lockerdome"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/logicad"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/lunamedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/marsmedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/mgid"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/mobfoxpb"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/mobilefuse"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/nanointeractive"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/ninthdecimal"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/nobid"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/openx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/orbidder"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pubmatic"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pubnative"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pulsepoint"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/revcontent"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rhythmone"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rtbhouse"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rubicon"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sharethrough"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/silvermob"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/smaato"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/smartadserver"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/smartrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/smartyads"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/somoaudience"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sonobi"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sovrn"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/spotx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/synacormedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/tappx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/telaria"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/triplelift"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/triplelift_native"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/ucfunnel"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/unruly"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/valueimpression"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/verizonmedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/visx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/vrtcal"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/yeahmobi"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/yieldlab"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/yieldmo"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/yieldone"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/zeroclickfraud"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
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
		openrtb_ext.BidderGamma:            gamma.Builder,
		openrtb_ext.BidderGamoshi:          gamoshi.Builder,
		openrtb_ext.BidderGrid:             grid.Builder,
		openrtb_ext.BidderGumGum:           gumgum.Builder,
		openrtb_ext.BidderImprovedigital:   improvedigital.Builder,
		openrtb_ext.BidderInMobi:           inmobi.Builder,
		openrtb_ext.BidderInvibes:          invibes.Builder,
		openrtb_ext.BidderIx:               ix.Builder,
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
		openrtb_ext.BidderOpenx:            openx.Builder,
		openrtb_ext.BidderOrbidder:         orbidder.Builder,
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
		openrtb_ext.BidderSpotX:            spotx.Builder,
		openrtb_ext.BidderSynacormedia:     synacormedia.Builder,
		openrtb_ext.BidderTappx:            tappx.Builder,
		openrtb_ext.BidderTelaria:          telaria.Builder,
		openrtb_ext.BidderTriplelift:       triplelift.Builder,
		openrtb_ext.BidderTripleliftNative: triplelift_native.Builder,
		openrtb_ext.BidderUcfunnel:         ucfunnel.Builder,
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
