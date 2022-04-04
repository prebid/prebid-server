package exchange

import (
	"encoding/json"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/metrics"
	"github.com/prebid/prebid-server/openrtb_ext"
)

func processStoredBidResponses(req AuctionRequest, aliases map[string]string) map[openrtb_ext.BidderName]BidderRequest {
	if len(req.StoredBidResponses) <= 0 {
		return nil
	}
	removeImpsWithStoredResponses(req)
	return buildStoredBidResponses(req, aliases)
}

// removeImpsWithStoredResponses deletes imps with stored bid resp
func removeImpsWithStoredResponses(req AuctionRequest) {
	imps := req.BidRequest.Imp
	req.BidRequest.Imp = nil
	for _, imp := range imps {
		if _, ok := req.StoredBidResponses[imp.ID]; !ok {
			//add real imp back to request
			req.BidRequest.Imp = append(req.BidRequest.Imp, imp)
		}
	}
}

func buildStoredBidResponses(req AuctionRequest, aliases map[string]string) map[openrtb_ext.BidderName]BidderRequest {
	// bidder -> imp id -> stored bid resp
	bidderToBidderResponse := make(map[openrtb_ext.BidderName]BidderRequest)
	for impID, storedData := range req.StoredBidResponses {
		for bidderName, storedResp := range storedData {
			if _, ok := bidderToBidderResponse[openrtb_ext.BidderName(bidderName)]; !ok {
				//new bidder with stored bid responses
				bidderStoredResp := make(map[string]json.RawMessage)
				bidderStoredResp[impID] = storedResp
				resolvedBidder := resolveBidder(bidderName, aliases)
				bidderToBidderResponse[openrtb_ext.BidderName(bidderName)] = BidderRequest{
					BidRequest:            req.BidRequest,
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

func prepareStoredResponse(impId string, bidResp json.RawMessage) *httpCallInfo {
	//always one element in reqData because stored response is mapped to single imp
	reqDataForStoredResp := adapters.RequestData{
		Method: "POST",
		Uri:    "",
		Body:   []byte(impId), //use it to pass imp id for stored resp
	}
	respData := &httpCallInfo{
		request: &reqDataForStoredResp,
		response: &adapters.ResponseData{
			StatusCode: 200,
			Body:       bidResp,
		},
		err: nil,
	}
	return respData
}
