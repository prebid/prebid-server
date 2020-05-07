package usersyncers

import (
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/telaria"
	"strings"
	"text/template"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adpone"

	"github.com/golang/glog"
	ttx "github.com/PubMatic-OpenWrap/prebid-server/adapters/33across"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adform"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adkernel"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adkernelAdn"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adtelligent"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/advangelists"
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
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pulsepoint"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rhythmone"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rtbhouse"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rubicon"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sharethrough"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/somoaudience"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sonobi"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sovrn"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/synacormedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/triplelift"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/triplelift_native"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/unruly"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/verizonmedia"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/visx"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/vrtcal"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/yieldmo"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

// NewSyncerMap returns a map of all the usersyncer objects.
// The same keys should exist in this map as in the exchanges map.
// Static syncer map will be removed when adapter isolation is complete.
func NewSyncerMap(cfg *config.Configuration) map[openrtb_ext.BidderName]usersync.Usersyncer {
	syncers := make(map[openrtb_ext.BidderName]usersync.Usersyncer, len(cfg.Adapters))

	insertIntoMap(cfg, syncers, openrtb_ext.Bidder33Across, ttx.New33AcrossSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdform, adform.NewAdformSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdkernel, adkernel.NewAdkernelSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdkernelAdn, adkernelAdn.NewAdkernelAdnSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdpone, adpone.NewadponeSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdtelligent, adtelligent.NewAdtelligentSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAdvangelists, advangelists.NewAdvangelistsSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderAppnexus, appnexus.NewAppnexusSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBeachfront, beachfront.NewBeachfrontSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderBrightroll, brightroll.NewBrightrollSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderConsumable, consumable.NewConsumableSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderConversant, conversant.NewConversantSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderDatablocks, datablocks.NewDatablocksSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderEmxDigital, emx_digital.NewEMXDigitalSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderEngageBDR, engagebdr.NewEngageBDRSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderEPlanning, eplanning.NewEPlanningSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderFacebook, audienceNetwork.NewFacebookSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGamma, gamma.NewGammaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGamoshi, gamoshi.NewGamoshiSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGrid, grid.NewGridSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderGumGum, gumgum.NewGumGumSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderImprovedigital, improvedigital.NewImprovedigitalSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderIx, ix.NewIxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderLifestreet, lifestreet.NewLifestreetSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderLockerDome, lockerdome.NewLockerDomeSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderMarsmedia, marsmedia.NewMarsmediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderMgid, mgid.NewMgidSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderOpenx, openx.NewOpenxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderPubmatic, pubmatic.NewPubmaticSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderPulsepoint, pulsepoint.NewPulsepointSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderRhythmone, rhythmone.NewRhythmoneSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderRTBHouse, rtbhouse.NewRTBHouseSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderRubicon, rubicon.NewRubiconSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSharethrough, sharethrough.NewSharethroughSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSomoaudience, somoaudience.NewSomoaudienceSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSonobi, sonobi.NewSonobiSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSovrn, sovrn.NewSovrnSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderSynacormedia, synacormedia.NewSynacorMediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTelaria, telaria.NewTelariaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTriplelift, triplelift.NewTripleliftSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderTripleliftNative, triplelift_native.NewTripleliftSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderUnruly, unruly.NewUnrulySyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderVerizonMedia, verizonmedia.NewVerizonMediaSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderVisx, visx.NewVisxSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderVrtcal, vrtcal.NewVrtcalSyncer)
	insertIntoMap(cfg, syncers, openrtb_ext.BidderYieldmo, yieldmo.NewYieldmoSyncer)

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
