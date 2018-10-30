package usersyncers

import (
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/adkernelAdn"
	"github.com/prebid/prebid-server/adapters/adtelligent"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/beachfront"
	"github.com/prebid/prebid-server/adapters/brightroll"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/eplanning"
	"github.com/prebid/prebid-server/adapters/ix"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/openx"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rhythmone"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/adapters/somoaudience"
	"github.com/prebid/prebid-server/adapters/sovrn"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/usersync"
)

// NewSyncerMap returns a map of all the usersyncer objects.
// The same keys should exist in this map as in the exchanges map.
// Static syncer map will be removed when adapter isolation is complete.
func NewSyncerMap(cfg *config.Configuration) map[openrtb_ext.BidderName]usersync.Usersyncer {
	return map[openrtb_ext.BidderName]usersync.Usersyncer{
		openrtb_ext.BidderAdform:       adform.NewAdformSyncer(cfg),
		openrtb_ext.BidderAdkernelAdn:  adkernelAdn.NewAdkernelAdnSyncer(cfg),
		openrtb_ext.BidderAdtelligent:  adtelligent.NewAdtelligentSyncer(cfg),
		openrtb_ext.BidderAppnexus:     appnexus.NewAppnexusSyncer(cfg),
		openrtb_ext.BidderBeachfront:   beachfront.NewBeachfrontSyncer(cfg),
		openrtb_ext.BidderFacebook:     audienceNetwork.NewFacebookSyncer(cfg),
		openrtb_ext.BidderBrightroll:   brightroll.NewBrightrollSyncer(cfg),
		openrtb_ext.BidderConversant:   conversant.NewConversantSyncer(cfg),
		openrtb_ext.BidderEPlanning:    eplanning.NewEPlanningSyncer(cfg),
		openrtb_ext.BidderIx:           ix.NewIxSyncer(cfg),
		openrtb_ext.BidderLifestreet:   lifestreet.NewLifestreetSyncer(cfg),
		openrtb_ext.BidderOpenx:        openx.NewOpenxSyncer(cfg),
		openrtb_ext.BidderPubmatic:     pubmatic.NewPubmaticSyncer(cfg),
		openrtb_ext.BidderPulsepoint:   pulsepoint.NewPulsepointSyncer(cfg),
		openrtb_ext.BidderRhythmone:    rhythmone.NewRhythmoneSyncer(cfg),
		openrtb_ext.BidderRubicon:      rubicon.NewRubiconSyncer(cfg),
		openrtb_ext.BidderSomoaudience: somoaudience.NewSomoaudienceSyncer(cfg),
		openrtb_ext.BidderSovrn:        sovrn.NewSovrnSyncer(cfg),
	}
}
