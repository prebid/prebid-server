package usersyncers

import (
	"github.com/prebid/prebid-server/adapters/operaads"
	"strings"
	"text/template"

	"github.com/golang/glog"
	ttx "github.com/prebid/prebid-server/adapters/33across"
	"github.com/prebid/prebid-server/adapters/acuityads"
	"github.com/prebid/prebid-server/adapters/adagio"
	"github.com/prebid/prebid-server/adapters/adf"
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/adkernel"
	"github.com/prebid/prebid-server/adapters/adkernelAdn"
	"github.com/prebid/prebid-server/adapters/adman"
	"github.com/prebid/prebid-server/adapters/admixer"
	"github.com/prebid/prebid-server/adapters/adocean"
	"github.com/prebid/prebid-server/adapters/adpone"
	"github.com/prebid/prebid-server/adapters/adtarget"
	"github.com/prebid/prebid-server/adapters/adtelligent"
	"github.com/prebid/prebid-server/adapters/advangelists"
	"github.com/prebid/prebid-server/adapters/adxcg"
	"github.com/prebid/prebid-server/adapters/adyoulike"
	"github.com/prebid/prebid-server/adapters/aja"
	"github.com/prebid/prebid-server/adapters/amx"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/avocet"
	"github.com/prebid/prebid-server/adapters/beachfront"
	"github.com/prebid/prebid-server/adapters/beintoo"
	"github.com/prebid/prebid-server/adapters/between"
	"github.com/prebid/prebid-server/adapters/bidmyadz"
	"github.com/prebid/prebid-server/adapters/bmtm"
	"github.com/prebid/prebid-server/adapters/brightroll"
	"github.com/prebid/prebid-server/adapters/colossus"
	"github.com/prebid/prebid-server/adapters/connectad"
	"github.com/prebid/prebid-server/adapters/consumable"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/cpmstar"
	"github.com/prebid/prebid-server/adapters/criteo"
	"github.com/prebid/prebid-server/adapters/datablocks"
	"github.com/prebid/prebid-server/adapters/deepintent"
	"github.com/prebid/prebid-server/adapters/dmx"
	"github.com/prebid/prebid-server/adapters/e_volution"
	"github.com/prebid/prebid-server/adapters/emx_digital"
	"github.com/prebid/prebid-server/adapters/engagebdr"
	"github.com/prebid/prebid-server/adapters/eplanning"
	"github.com/prebid/prebid-server/adapters/gamma"
	"github.com/prebid/prebid-server/adapters/gamoshi"
	"github.com/prebid/prebid-server/adapters/grid"
	"github.com/prebid/prebid-server/adapters/gumgum"
	"github.com/prebid/prebid-server/adapters/improvedigital"
	"github.com/prebid/prebid-server/adapters/inmobi"
	"github.com/prebid/prebid-server/adapters/invibes"
	"github.com/prebid/prebid-server/adapters/ix"
	"github.com/prebid/prebid-server/adapters/jixie"
	"github.com/prebid/prebid-server/adapters/krushmedia"
	"github.com/prebid/prebid-server/adapters/lockerdome"
	"github.com/prebid/prebid-server/adapters/logicad"
	"github.com/prebid/prebid-server/adapters/lunamedia"
	"github.com/prebid/prebid-server/adapters/marsmedia"
	"github.com/prebid/prebid-server/adapters/mediafuse"
	"github.com/prebid/prebid-server/adapters/mgid"
	"github.com/prebid/prebid-server/adapters/nanointeractive"
	"github.com/prebid/prebid-server/adapters/ninthdecimal"
	"github.com/prebid/prebid-server/adapters/nobid"
	"github.com/prebid/prebid-server/adapters/onetag"
	"github.com/prebid/prebid-server/adapters/openx"
	"github.com/prebid/prebid-server/adapters/outbrain"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rhythmone"
	"github.com/prebid/prebid-server/adapters/rtbhouse"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/adapters/sa_lunamedia"
	"github.com/prebid/prebid-server/adapters/sharethrough"
	"github.com/prebid/prebid-server/adapters/smartadserver"
	"github.com/prebid/prebid-server/adapters/smarthub"
	"github.com/prebid/prebid-server/adapters/smartrtb"
	"github.com/prebid/prebid-server/adapters/smartyads"
	"github.com/prebid/prebid-server/adapters/smilewanted"
	"github.com/prebid/prebid-server/adapters/somoaudience"
	"github.com/prebid/prebid-server/adapters/sonobi"
	"github.com/prebid/prebid-server/adapters/sovrn"
	"github.com/prebid/prebid-server/adapters/synacormedia"
	"github.com/prebid/prebid-server/adapters/tappx"
	"github.com/prebid/prebid-server/adapters/telaria"
	"github.com/prebid/prebid-server/adapters/triplelift"
	"github.com/prebid/prebid-server/adapters/triplelift_native"
	"github.com/prebid/prebid-server/adapters/trustx"
	"github.com/prebid/prebid-server/adapters/ucfunnel"
	"github.com/prebid/prebid-server/adapters/unruly"
	"github.com/prebid/prebid-server/adapters/valueimpression"
	"github.com/prebid/prebid-server/adapters/verizonmedia"
	"github.com/prebid/prebid-server/adapters/viewdeos"
	"github.com/prebid/prebid-server/adapters/visx"
	"github.com/prebid/prebid-server/adapters/vrtcal"
	"github.com/prebid/prebid-server/adapters/yieldlab"
	"github.com/prebid/prebid-server/adapters/yieldmo"
	"github.com/prebid/prebid-server/adapters/yieldone"
	"github.com/prebid/prebid-server/adapters/zeroclickfraud"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

// NewSyncerMap returns a map of all the usersyncer objects.
// The same keys should exist in this map as in the exchanges map.
// Static syncer map will be removed when adapter isolation is complete.
func NewSyncerMap(cfg *config.Configuration) map[openrtb_ext.BidderName]usersync.Usersyncer {
	syncers := make(map[openrtb_ext.BidderName]usersync.Usersyncer, len(cfg.Adapters))

	insertIntoMap(cfg, syncers, openrtb_ext.Bidder33Across, ttx.New33AcrossSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAcuityAds, acuityads.NewAcuityAdsSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdagio, adagio.NewAdagioSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdf, adf.NewAdfSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdform, adform.NewAdformSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdkernel, adkernel.NewAdkernelSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdkernelAdn, adkernelAdn.NewAdkernelAdnSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdman, adman.NewAdmanSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdmixer, admixer.NewAdmixerSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdOcean, adocean.NewAdOceanSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdpone, adpone.NewadponeSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdtarget, adtarget.NewAdtargetSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdtelligent, adtelligent.NewAdtelligentSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdvangelists, advangelists.NewAdvangelistsSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdxcg, adxcg.NewAdxcgSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdyoulike, adyoulike.NewAdyoulikeSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAJA, aja.NewAJASyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAMX, amx.NewAMXSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAppnexus, appnexus.NewAppnexusSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAvocet, avocet.NewAvocetSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBeachfront, beachfront.NewBeachfrontSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBeintoo, beintoo.NewBeintooSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBmtm, bmtm.NewBmtmSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBrightroll, brightroll.NewBrightrollSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBidmyadz, bidmyadz.NewBidmyadzSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderColossus, colossus.NewColossusSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderConnectAd, connectad.NewConnectAdSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderConsumable, consumable.NewConsumableSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderConversant, conversant.NewConversantSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderCriteo, criteo.NewCriteoSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderCpmstar, cpmstar.NewCpmstarSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderDatablocks, datablocks.NewDatablocksSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderDeepintent, deepintent.NewDeepintentSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderDmx, dmx.NewDmxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderEmxDigital, emx_digital.NewEMXDigitalSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderEVolution, evolution.NewEvolutionSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderEngageBDR, engagebdr.NewEngageBDRSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderEPlanning, eplanning.NewEPlanningSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAudienceNetwork, audienceNetwork.NewFacebookSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGamma, gamma.NewGammaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGamoshi, gamoshi.NewGamoshiSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGrid, grid.NewGridSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGumGum, gumgum.NewGumGumSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderImprovedigital, improvedigital.NewImprovedigitalSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderInMobi, inmobi.NewInmobiSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderInvibes, invibes.NewInvibesSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderIx, ix.NewIxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderJixie, jixie.NewJixieSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderKrushmedia, krushmedia.NewKrushmediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderLockerDome, lockerdome.NewLockerDomeSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderLogicad, logicad.NewLogicadSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderLunaMedia, lunamedia.NewLunaMediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSaLunaMedia, salunamedia.NewSaLunamediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderMarsmedia, marsmedia.NewMarsmediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderMediafuse, mediafuse.NewMediafuseSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderMgid, mgid.NewMgidSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderNanoInteractive, nanointeractive.NewNanoInteractiveSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderNinthDecimal, ninthdecimal.NewNinthDecimalSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderNoBid, nobid.NewNoBidSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderOneTag, onetag.NewSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderOutbrain, outbrain.NewOutbrainSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderOpenx, openx.NewOpenxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderOperaads, operaads.NewOperaadsSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderPubmatic, pubmatic.NewPubmaticSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderPulsepoint, pulsepoint.NewPulsepointSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderRhythmone, rhythmone.NewRhythmoneSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderRTBHouse, rtbhouse.NewRTBHouseSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderRubicon, rubicon.NewRubiconSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSharethrough, sharethrough.NewSharethroughSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSomoaudience, somoaudience.NewSomoaudienceSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSonobi, sonobi.NewSonobiSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSovrn, sovrn.NewSovrnSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSmartAdserver, smartadserver.NewSmartadserverSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSmartHub, smarthub.NewSmartHubSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSmartRTB, smartrtb.NewSmartRTBSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSmartyAds, smartyads.NewSmartyAdsSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSmileWanted, smilewanted.NewSmileWantedSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSynacormedia, synacormedia.NewSynacorMediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTappx, tappx.NewTappxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTelaria, telaria.NewTelariaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTriplelift, triplelift.NewTripleliftSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTripleliftNative, triplelift_native.NewTripleliftSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTrustX, trustx.NewTrustXSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderUcfunnel, ucfunnel.NewUcfunnelSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderUnruly, unruly.NewUnrulySyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderValueImpression, valueimpression.NewValueImpressionSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderVerizonMedia, verizonmedia.NewVerizonMediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderViewdeos, viewdeos.NewViewdeosSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderVisx, visx.NewVisxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderVrtcal, vrtcal.NewVrtcalSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderYieldlab, yieldlab.NewYieldlabSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderYieldmo, yieldmo.NewYieldmoSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderYieldone, yieldone.NewYieldoneSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderZeroClickFraud, zeroclickfraud.NewZeroClickFraudSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBetween, between.NewBetweenSyncer)

	return syncers
}

func insertIntoMap(cfg *config.Configuration, syncers map[openrtb_ext.BidderName]usersync.Usersyncer, bidder openrtb_ext.BidderName, syncerFactory func(*template.Template) usersync.Usersyncer) {
	lowercased := strings.ToLower(string(bidder))
	urlString := cfg.Adapters[lowercased].UserSyncURL
	if urlString == "" {
		glog.Warningf("adapters." + string(bidder) + ".usersync_url was not defined, and their usersync API isn't flexible enough for Prebid Server to choose a good default. No usersyncs will be performed with " + string(bidder))
		return
	}
	syncers[bidder] = syncerFactory(template.Must(template.New(lowercased + "_usersync_url").Parse(urlString)))
}
