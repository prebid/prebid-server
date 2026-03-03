package exchange

import (
	"time"

	"github.com/prebid/prebid-server/v3/exchange/entities"
	jsonpatch "gopkg.in/evanphx/json-patch.v5"

	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/endpoints/events"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// eventTracking has configuration fields needed for adding event tracking to an auction response
type eventTracking struct {
	accountID          string
	enabledForAccount  bool
	enabledForRequest  bool
	auctionTimestampMs int64
	integrationType    string
	bidderInfos        config.BidderInfos
	externalURL        string
}

// getEventTracking creates an eventTracking object from the different configuration sources
func getEventTracking(requestExtPrebid *openrtb_ext.ExtRequestPrebid, ts time.Time, account *config.Account, bidderInfos config.BidderInfos, externalURL string) *eventTracking {
	return &eventTracking{
		accountID:          account.ID,
		enabledForAccount:  account.Events.Enabled,
		enabledForRequest:  requestExtPrebid != nil && requestExtPrebid.Events != nil,
		auctionTimestampMs: ts.UnixNano() / 1e+6,
		integrationType:    getIntegrationType(requestExtPrebid),
		bidderInfos:        bidderInfos,
		externalURL:        externalURL,
	}
}

func getIntegrationType(requestExtPrebid *openrtb_ext.ExtRequestPrebid) string {
	if requestExtPrebid != nil {
		return requestExtPrebid.Integration
	}
	return ""
}

// modifyBidsForEvents adds bidEvents and modifies VAST AdM if necessary.
func (ev *eventTracking) modifyBidsForEvents(seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid) map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid {
	for bidderName, seatBid := range seatBids {
		modifyingVastXMLAllowed := ev.isModifyingVASTXMLAllowed(bidderName.String())
		for _, pbsBid := range seatBid.Bids {
			if modifyingVastXMLAllowed {
				ev.modifyBidVAST(pbsBid, bidderName)
			}
			pbsBid.BidEvents = ev.makeBidExtEvents(pbsBid, bidderName)
		}
	}
	return seatBids
}

// isModifyingVASTXMLAllowed returns true if this bidder config allows modifying VAST XML for event tracking
func (ev *eventTracking) isModifyingVASTXMLAllowed(bidderName string) bool {
	return ev.bidderInfos[bidderName].ModifyingVastXmlAllowed && ev.isEventAllowed()
}

// modifyBidVAST injects event Impression url if needed, otherwise returns original VAST string
func (ev *eventTracking) modifyBidVAST(pbsBid *entities.PbsOrtbBid, bidderName openrtb_ext.BidderName) {
	bid := pbsBid.Bid
	if pbsBid.BidType != openrtb_ext.BidTypeVideo || len(bid.AdM) == 0 && len(bid.NURL) == 0 {
		return
	}
	vastXML := makeVAST(bid)
	bidID := bid.ID
	if len(pbsBid.GeneratedBidID) > 0 {
		bidID = pbsBid.GeneratedBidID
	}
	if newVastXML, ok := events.ModifyVastXmlString(ev.externalURL, vastXML, bidID, bidderName.String(), ev.accountID, ev.auctionTimestampMs, ev.integrationType); ok {
		bid.AdM = newVastXML
	}
}

// modifyBidJSON injects "wurl" (win) event url if needed, otherwise returns original json
func (ev *eventTracking) modifyBidJSON(pbsBid *entities.PbsOrtbBid, bidderName openrtb_ext.BidderName, jsonBytes []byte) ([]byte, error) {
	if !ev.isEventAllowed() || pbsBid.BidType == openrtb_ext.BidTypeVideo {
		return jsonBytes, nil
	}
	var winEventURL string
	if pbsBid.BidEvents != nil { // All bids should have already been updated with win/imp event URLs
		winEventURL = pbsBid.BidEvents.Win
	} else {
		winEventURL = ev.makeEventURL(analytics.Win, pbsBid, bidderName)
	}
	// wurl attribute is not in the schema, so we have to patch
	patch, err := jsonutil.Marshal(map[string]string{"wurl": winEventURL})
	if err != nil {
		return jsonBytes, err
	}
	modifiedJSON, err := jsonpatch.MergePatch(jsonBytes, patch)
	if err != nil {
		return jsonBytes, err
	}
	return modifiedJSON, nil
}

// makeBidExtEvents make the data for bid.ext.prebid.events if needed, otherwise returns nil
func (ev *eventTracking) makeBidExtEvents(pbsBid *entities.PbsOrtbBid, bidderName openrtb_ext.BidderName) *openrtb_ext.ExtBidPrebidEvents {
	if !ev.isEventAllowed() || pbsBid.BidType == openrtb_ext.BidTypeVideo {
		return nil
	}
	return &openrtb_ext.ExtBidPrebidEvents{
		Win: ev.makeEventURL(analytics.Win, pbsBid, bidderName),
		Imp: ev.makeEventURL(analytics.Imp, pbsBid, bidderName),
	}
}

// makeEventURL returns an analytics event url for the requested type (win or imp)
func (ev *eventTracking) makeEventURL(evType analytics.EventType, pbsBid *entities.PbsOrtbBid, bidderName openrtb_ext.BidderName) string {
	bidId := pbsBid.Bid.ID
	if len(pbsBid.GeneratedBidID) > 0 {
		bidId = pbsBid.GeneratedBidID
	}
	return events.EventRequestToUrl(ev.externalURL,
		&analytics.EventRequest{
			Type:        evType,
			BidID:       bidId,
			Bidder:      string(bidderName),
			AccountID:   ev.accountID,
			Timestamp:   ev.auctionTimestampMs,
			Integration: ev.integrationType,
		})
}

// isEventAllowed checks if events are enabled by default or on account/request level
func (ev *eventTracking) isEventAllowed() bool {
	return ev.enabledForAccount || ev.enabledForRequest
}
