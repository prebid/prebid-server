package stored_responses

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
)

//type ImpsWithAuctionResponseIDs map[string]string
//type ImpsWithAuctionResponses map[string]json.RawMessage
//type ImpBiddersWithBidResponseIDs map[string]map[openrtb_ext.BidderName]string
//type ImpBiddersWithBidResponses map[string]map[openrtb_ext.BidderName]json.RawMessage

type BidderImpsWithBidResponses map[openrtb_ext.BidderName]map[string]json.RawMessage
type ImpsWithBidResponses map[string]json.RawMessage
type ImpBidderStoredResp map[string]map[string]json.RawMessage

type StoredBidResponses struct {
	StoredBidResponses     ImpBidderStoredResp
	BidderToImpToResponses BidderImpsWithBidResponses
}

func (sr *StoredBidResponses) InitStoredBidResponses(req *openrtb2.BidRequest) {
	sr.removeImpsWithStoredResponses(req)
	sr.buildStoredResp()
}

// removeImpsWithStoredResponses deletes imps with stored bid resp
func (sr *StoredBidResponses) removeImpsWithStoredResponses(req *openrtb2.BidRequest) {
	imps := req.Imp
	req.Imp = nil //to indicate this bidder doesn't have real requests
	for _, imp := range imps {
		if _, ok := sr.StoredBidResponses[imp.ID]; !ok {
			//add real imp back to request
			req.Imp = append(req.Imp, imp)
		}
	}
}

func (sr *StoredBidResponses) buildStoredResp() {
	// bidder -> imp id -> stored bid resp
	sr.BidderToImpToResponses = make(map[openrtb_ext.BidderName]map[string]json.RawMessage)
	for impID, storedData := range sr.StoredBidResponses {
		for bidderName, storedResp := range storedData {
			if _, ok := sr.BidderToImpToResponses[openrtb_ext.BidderName(bidderName)]; !ok {
				//new bidder with stored bid responses
				impToStoredResp := make(map[string]json.RawMessage)
				impToStoredResp[impID] = storedResp
				sr.BidderToImpToResponses[openrtb_ext.BidderName(bidderName)] = impToStoredResp
			} else {
				sr.BidderToImpToResponses[openrtb_ext.BidderName(bidderName)][impID] = storedResp
			}
		}
	}
}
