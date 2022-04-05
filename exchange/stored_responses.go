package exchange

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type StoredResponses struct {
	bidResponses       map[openrtb_ext.BidderName]BidderRequest
	storedBidResponses map[string]map[string]json.RawMessage
	aliases            map[string]string
}

func (sr *StoredResponses) initStoredBidResponses(req *openrtb2.BidRequest) {
	sr.removeImpsWithStoredResponses(req)
	sr.bidResponses = sr.buildStoredBidResponses(req)
}

// removeImpsWithStoredResponses deletes imps with stored bid resp
func (sr *StoredResponses) removeImpsWithStoredResponses(req *openrtb2.BidRequest) {
	imps := req.Imp
	req.Imp = nil
	for _, imp := range imps {
		if _, ok := sr.storedBidResponses[imp.ID]; !ok {
			//add real imp back to request
			req.Imp = append(req.Imp, imp)
		}
	}
}

func (sr *StoredResponses) buildStoredBidResponses(req *openrtb2.BidRequest) map[openrtb_ext.BidderName]BidderRequest {
	// bidder -> imp id -> stored bid resp
	bidderToBidderResponse := make(map[openrtb_ext.BidderName]BidderRequest)
	for impID, storedData := range sr.storedBidResponses {
		for bidderName, storedResp := range storedData {
			if _, ok := bidderToBidderResponse[openrtb_ext.BidderName(bidderName)]; !ok {
				//new bidder with stored bid responses
				bidderStoredResp := make(map[string]json.RawMessage)
				bidderStoredResp[impID] = storedResp
				resolvedBidder := resolveBidder(bidderName, sr.aliases)
				bidderToBidderResponse[openrtb_ext.BidderName(bidderName)] = BidderRequest{
					BidRequest:            req,
					BidderCoreName:        resolvedBidder,
					BidderName:            openrtb_ext.BidderName(bidderName),
					BidderStoredResponses: bidderStoredResp,
					BidderLabels:          metrics.AdapterLabels{Adapter: resolvedBidder},
				}
			} else {
				bidderToBidderResponse[openrtb_ext.BidderName(bidderName)].BidderStoredResponses[impID] = storedResp
			}
		}
	}
	return bidderToBidderResponse
}

func (sr *StoredResponses) removeBidRequestsWithRealRequests(bidderRequest *BidderRequest) {
	if bidderWithStoredBidResponses, ok := sr.bidResponses[bidderRequest.BidderName]; ok {
		//this bidder has real imps and imps with stored bid response
		bidderRequest.BidderStoredResponses = bidderWithStoredBidResponses.BidderStoredResponses
		delete(sr.bidResponses, bidderRequest.BidderCoreName)
	}
}

//getAllRemaining checks if any bidders with storedBidResponses only left
func (sr *StoredResponses) getAllRemaining() []BidderRequest {
	bidderRequests := make([]BidderRequest, 0)
	if len(sr.bidResponses) > 0 {
		for _, bidResp := range sr.bidResponses {
			bidResp.BidRequest.Imp = nil //to indicate this bidder doesn't have real requests
			bidderRequests = append(bidderRequests, bidResp)
		}
	}
	return bidderRequests
}
