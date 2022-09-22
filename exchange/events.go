package exchange

import (
	"encoding/json"
	"time"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/endpoints/events"
	"github.com/prebid/prebid-server/openrtb_ext"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
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
		enabledForAccount:  account.EventsEnabled,
		enabledForRequest:  requestExtPrebid != nil && requestExtPrebid.Events != nil,
		auctionTimestampMs: ts.UnixNano() / 1e+6,
		integrationType:    requestExtPrebid.Integration,
		bidderInfos:        bidderInfos,
		externalURL:        externalURL,
	}
}

// modifyBidsForEvents adds bidEvents and modifies VAST AdM if necessary.
func (ev *eventTracking) modifyBidsForEvents(seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, req *openrtb2.BidRequest, trackerURL string) map[openrtb_ext.BidderName]*pbsOrtbSeatBid {
	for bidderName, seatBid := range seatBids {
		for _, pbsBid := range seatBid.bids {
			ev.modifyBidVAST(pbsBid, bidderName, seatBid.bidderCoreName, req, trackerURL)
			pbsBid.bidEvents = ev.makeBidExtEvents(pbsBid, bidderName)
		}
	}
	return seatBids
}

// isModifyingVASTXMLAllowed returns true if this bidder config allows modifying VAST XML for event tracking
func (ev *eventTracking) isModifyingVASTXMLAllowed(bidderName string) bool {
	return ev.bidderInfos[bidderName].ModifyingVastXmlAllowed && (ev.enabledForAccount || ev.enabledForRequest)
}

// modifyBidVAST injects event Impression url if needed, otherwise returns original VAST string
func (ev *eventTracking) modifyBidVAST(pbsBid *pbsOrtbBid, bidderName openrtb_ext.BidderName, bidderCoreName openrtb_ext.BidderName, req *openrtb2.BidRequest, trackerURL string) {
	bid := pbsBid.bid
	if pbsBid.bidType != openrtb_ext.BidTypeVideo || len(bid.AdM) == 0 && len(bid.NURL) == 0 {
		return
	}
	vastXML := makeVAST(bid)
	bidID := bid.ID
	if len(pbsBid.generatedBidID) > 0 {
		bidID = pbsBid.generatedBidID
	}

	if ev.isModifyingVASTXMLAllowed(bidderName.String()) { // condition added for ow fork
		if newVastXML, ok := events.ModifyVastXmlString(ev.externalURL, vastXML, bidID, bidderName.String(), ev.accountID, ev.auctionTimestampMs, ev.integrationType); ok {
			bid.AdM = newVastXML
		}
	}

	// always inject event  trackers without checkign isModifyingVASTXMLAllowed
	if newVastXML, injected, _ := events.InjectVideoEventTrackers(trackerURL, vastXML, bid, bidID, bidderName.String(), bidderCoreName.String(), ev.accountID, ev.auctionTimestampMs, req); injected {
		bid.AdM = string(newVastXML)
	}
}

// modifyBidJSON injects "wurl" (win) event url if needed, otherwise returns original json
func (ev *eventTracking) modifyBidJSON(pbsBid *pbsOrtbBid, bidderName openrtb_ext.BidderName, jsonBytes []byte) ([]byte, error) {
	if !(ev.enabledForAccount || ev.enabledForRequest) || pbsBid.bidType == openrtb_ext.BidTypeVideo {
		return jsonBytes, nil
	}
	var winEventURL string
	if pbsBid.bidEvents != nil { // All bids should have already been updated with win/imp event URLs
		winEventURL = pbsBid.bidEvents.Win
	} else {
		winEventURL = ev.makeEventURL(analytics.Win, pbsBid, bidderName)
	}
	// wurl attribute is not in the schema, so we have to patch
	patch, err := json.Marshal(map[string]string{"wurl": winEventURL})
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
func (ev *eventTracking) makeBidExtEvents(pbsBid *pbsOrtbBid, bidderName openrtb_ext.BidderName) *openrtb_ext.ExtBidPrebidEvents {
	if !(ev.enabledForAccount || ev.enabledForRequest) || pbsBid.bidType == openrtb_ext.BidTypeVideo {
		return nil
	}
	return &openrtb_ext.ExtBidPrebidEvents{
		Win: ev.makeEventURL(analytics.Win, pbsBid, bidderName),
		Imp: ev.makeEventURL(analytics.Imp, pbsBid, bidderName),
	}
}

// makeEventURL returns an analytics event url for the requested type (win or imp)
func (ev *eventTracking) makeEventURL(evType analytics.EventType, pbsBid *pbsOrtbBid, bidderName openrtb_ext.BidderName) string {
	bidId := pbsBid.bid.ID
	if len(pbsBid.generatedBidID) > 0 {
		bidId = pbsBid.generatedBidID
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
