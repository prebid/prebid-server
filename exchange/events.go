package exchange

import (
	"encoding/json"
	"time"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/endpoints/events"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
)

// eventsData has configuration fields needed for adding event tracking to an auction response
type eventsData struct {
	accountID          string
	enabledForAccount  bool
	enabledForRequest  bool
	auctionTimestampMs int64
	integration        pbsmetrics.DemandSource // web app amp
	bidderInfos        adapters.BidderInfos
	externalURL        string
}

// getExtEventsData creates an eventsData object from the different configuration sources
func getExtEventsData(requestExtPrebid *openrtb_ext.ExtRequestPrebid, ts time.Time, account *config.Account, bidderInfos adapters.BidderInfos, externalURL string) *eventsData {
	return &eventsData{
		accountID:          account.ID,
		enabledForAccount:  account.EventsEnabled,
		enabledForRequest:  requestExtPrebid != nil && requestExtPrebid.Events != nil,
		auctionTimestampMs: ts.UnixNano() / 1e+6,
		integration:        "", // FIXME
		bidderInfos:        bidderInfos,
		externalURL:        externalURL,
	}
}

// isModifyingVASTXMLAllowed returns true if this bidder config allows modifying VAST XML for event tracking
func (ev *eventsData) isModifyingVASTXMLAllowed(bidderName string) bool {
	return ev.bidderInfos[bidderName].ModifyingVastXmlAllowed && ev.enabledForAccount
}

// modifyVAST injects event Impression url if needed, otherwise returns original VAST string
func (ev *eventsData) modifyVAST(bid *openrtb.Bid, bidderName openrtb_ext.BidderName, vastXML string) string {
	if ev.isModifyingVASTXMLAllowed(bidderName.String()) {
		if newVastXML, ok := events.ModifyVastXmlString(ev.externalURL, vastXML, bid.ID, bidderName.String(), ev.accountID, ev.auctionTimestampMs); ok {
			return newVastXML
		}
	}
	return vastXML
}

// modifyBidJSON injects "wurl" (win) event url if needed, otherwise returns original json
func (ev *eventsData) modifyBidJSON(bid *openrtb.Bid, bidderName openrtb_ext.BidderName, jsonBytes []byte) []byte {
	if !ev.enabledForAccount && !ev.enabledForRequest {
		return jsonBytes
	}
	// wurl attribute is not in the schema, so we have to patch
	if patch, err := json.Marshal(map[string]string{"wurl": ev.makeEventURL(analytics.Win, bid, bidderName)}); err == nil {
		if modifiedJSON, err := jsonpatch.MergePatch(jsonBytes, patch); err == nil {
			jsonBytes = modifiedJSON
		}
	}
	return jsonBytes
}

// makeBidExtEvents make the data for bid.ext.prebid.events if needed, otherwise returns nil
func (ev *eventsData) makeBidExtEvents(bid *openrtb.Bid, bidderName openrtb_ext.BidderName) *openrtb_ext.ExtBidPrebidEvents {
	if !ev.enabledForAccount && !ev.enabledForRequest {
		return nil
	}
	return &openrtb_ext.ExtBidPrebidEvents{
		Win: ev.makeEventURL(analytics.Win, bid, bidderName),
		Imp: ev.makeEventURL(analytics.Imp, bid, bidderName),
	}
}

// makeEventURL returns an analytics event url for the requested type (win or imp)
func (ev *eventsData) makeEventURL(evType analytics.EventType, bid *openrtb.Bid, bidderName openrtb_ext.BidderName) string {
	return events.EventRequestToUrl(ev.externalURL,
		&analytics.EventRequest{
			Type:      evType,
			BidID:     bid.ID,
			Bidder:    string(bidderName),
			AccountID: ev.accountID,
			Timestamp: ev.auctionTimestampMs,
		})
}
