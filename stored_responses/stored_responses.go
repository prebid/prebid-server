package stored_responses

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/buger/jsonparser"
	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/stored_requests"
)

type ImpsWithAuctionResponseIDs map[string]string
type ImpBiddersWithBidResponseIDs map[string]map[string]string
type StoredResponseIDs []string
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

func extractStoredResponsesIds(impInfo []ImpExtPrebidData,
	bidderMap map[string]openrtb_ext.BidderName) (
	StoredResponseIDs,
	ImpBiddersWithBidResponseIDs,
	ImpsWithAuctionResponseIDs, error,
) {

	// 1) all stored responses ids from all imps
	allStoredResponseIDs := make([]string, 0, 0)
	// 2) stored bid responses: imp id to bidder to stored response id
	impBiddersWithBidResponseIDs := make(map[string]map[string]string)
	// 3) imp id to stored resp id
	impAuctionResponseIDs := make(map[string]string)

	for index, impData := range impInfo {
		impId, err := jsonparser.GetString(impData.Imp, "id")
		if err != nil {
			return nil, nil, nil, fmt.Errorf("request.imp[%d] missing required field: \"id\"", index)
		}

		if impData.ImpExtPrebid.StoredAuctionResponse != nil {
			if len(impData.ImpExtPrebid.StoredAuctionResponse.ID) == 0 {
				return nil, nil, nil, fmt.Errorf("request.imp[%d] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ", index)
			}
			allStoredResponseIDs = append(allStoredResponseIDs, impData.ImpExtPrebid.StoredAuctionResponse.ID)

			impAuctionResponseIDs[impId] = impData.ImpExtPrebid.StoredAuctionResponse.ID

		}
		if len(impData.ImpExtPrebid.StoredBidResponse) > 0 {

			bidderStoredRespId := make(map[string]string)
			for _, bidderResp := range impData.ImpExtPrebid.StoredBidResponse {
				if len(bidderResp.ID) == 0 || len(bidderResp.Bidder) == 0 {
					return nil, nil, nil, fmt.Errorf("request.imp[%d] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ", index)
				}
				//check if bidder is valid/exists
				if _, isValid := bidderMap[bidderResp.Bidder]; !isValid {
					return nil, nil, nil, fmt.Errorf("request.imp[impId: %s].ext contains unknown bidder: %s. Did you forget an alias in request.ext.prebid.aliases?", impId, bidderResp.Bidder)
				}
				// bidder is unique per one bid stored response
				// if more than one bidder specified the last defined bidder id will take precedence
				bidderStoredRespId[bidderResp.Bidder] = bidderResp.ID
				impBiddersWithBidResponseIDs[impId] = bidderStoredRespId
				//storedAuctionResponseIds are not unique, but fetch will return single data for repeated ids
				allStoredResponseIDs = append(allStoredResponseIDs, bidderResp.ID)
			}
		}
	}
	return allStoredResponseIDs, impBiddersWithBidResponseIDs, impAuctionResponseIDs, nil
}

// ProcessStoredResponses takes the incoming request as JSON with any
// stored requests/imps already merged into it, scans it to find any stored auction response ids
// in the request/imps and produces a map of imp IDs to stored auction responses.
// Note that processStoredResponses must be called after processStoredRequests
// because stored imps and stored requests can contain stored auction responses
// so the stored requests/imps have to be merged into the incoming request prior to processing stored auction responses.
func ProcessStoredResponses(ctx context.Context, requestJson []byte, storedRespFetcher stored_requests.Fetcher, bidderMap map[string]openrtb_ext.BidderName) (map[string]json.RawMessage, map[string]map[string]json.RawMessage, []error) {
	impInfo, errs := parseImpInfo(requestJson)
	if len(errs) > 0 {
		return nil, nil, errs
	}
	storedResponsesIds, impBidderToStoredBidResponseId, impIdToRespId, err := extractStoredResponsesIds(impInfo, bidderMap)
	if err != nil {
		return nil, nil, append(errs, err)
	}

	if len(storedResponsesIds) > 0 {
		storedResponses, errs := storedRespFetcher.FetchResponses(ctx, storedResponsesIds)
		if len(errs) > 0 {
			return nil, nil, errs
		}

		impIdToStoredResp, impBidderToStoredBidResponse := buildStoredResponsesMaps(storedResponses, impBidderToStoredBidResponseId, impIdToRespId)
		return impIdToStoredResp, impBidderToStoredBidResponse, nil
	}
	return nil, nil, nil
}

func buildStoredResponsesMaps(storedResponses map[string]json.RawMessage, impBidderToStoredBidResponseId map[string]map[string]string, impIdToRespId map[string]string) (map[string]json.RawMessage, map[string]map[string]json.RawMessage) {
	//imp id to stored resp body
	impIdToStoredResp := make(map[string]json.RawMessage)
	//stored bid responses: imp id to bidder to stored response body
	impBidderToStoredBidResponse := make(map[string]map[string]json.RawMessage)

	for impId, respId := range impIdToRespId {
		impIdToStoredResp[impId] = storedResponses[respId]
	}

	for impId, bidderStoredResp := range impBidderToStoredBidResponseId {
		bidderStoredResponses := make(map[string]json.RawMessage)
		for bidderName, id := range bidderStoredResp {
			bidderStoredResponses[bidderName] = storedResponses[id]
		}
		impBidderToStoredBidResponse[impId] = bidderStoredResponses
	}
	return impIdToStoredResp, impBidderToStoredBidResponse
}

// parseImpInfo parses the request JSON and returns impression and unmarshalled imp.ext.prebid
// copied from exchange to isolate stored responses code from auction dependencies
func parseImpInfo(requestJson []byte) (impData []ImpExtPrebidData, errs []error) {

	if impArray, dataType, _, err := jsonparser.Get(requestJson, "imp"); err == nil && dataType == jsonparser.Array {
		_, err = jsonparser.ArrayEach(impArray, func(imp []byte, _ jsonparser.ValueType, _ int, err error) {
			impExtData, _, _, err := jsonparser.Get(imp, "ext", "prebid")
			var impExtPrebid openrtb_ext.ExtImpPrebid
			if impExtData != nil {
				if err := json.Unmarshal(impExtData, &impExtPrebid); err != nil {
					errs = append(errs, err)
				}
			}
			newImpData := ImpExtPrebidData{imp, impExtPrebid}
			impData = append(impData, newImpData)
		})
	}
	return
}

type ImpExtPrebidData struct {
	Imp          json.RawMessage
	ImpExtPrebid openrtb_ext.ExtImpPrebid
}
