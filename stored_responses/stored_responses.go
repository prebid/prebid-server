package stored_responses

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/stored_requests"
)

type ImpsWithAuctionResponseIDs map[string]string
type ImpBiddersWithBidResponseIDs map[string]map[string]string
type StoredResponseIDs []string
type StoredResponseIdToStoredResponse map[string]json.RawMessage
type BidderImpsWithBidResponses map[openrtb_ext.BidderName]map[string]json.RawMessage
type ImpsWithBidResponses map[string]json.RawMessage
type ImpBidderStoredResp map[string]map[string]json.RawMessage
type ImpBidderReplaceImpID map[string]map[string]bool
type BidderImpReplaceImpID map[string]map[string]bool

func InitStoredBidResponses(req *openrtb2.BidRequest, storedBidResponses ImpBidderStoredResp) BidderImpsWithBidResponses {
	return buildStoredResp(storedBidResponses)
}

func buildStoredResp(storedBidResponses ImpBidderStoredResp) BidderImpsWithBidResponses {
	// bidder -> imp id -> stored bid resp
	bidderToImpToResponses := BidderImpsWithBidResponses{}
	for impID, storedData := range storedBidResponses {
		for bidderName, storedResp := range storedData {
			if _, ok := bidderToImpToResponses[openrtb_ext.BidderName(bidderName)]; !ok {
				//new bidder with stored bid responses
				impToStoredResp := ImpsWithBidResponses{}
				impToStoredResp[impID] = storedResp
				bidderToImpToResponses[openrtb_ext.BidderName(bidderName)] = impToStoredResp
			} else {
				bidderToImpToResponses[openrtb_ext.BidderName(bidderName)][impID] = storedResp
			}
		}
	}
	return bidderToImpToResponses
}

func extractStoredResponsesIds(impInfo []*openrtb_ext.ImpWrapper) (
	StoredResponseIDs,
	ImpBiddersWithBidResponseIDs,
	ImpsWithAuctionResponseIDs,
	ImpBidderReplaceImpID,
	error,
) {
	// extractStoredResponsesIds returns:
	// 1) all stored responses ids from all imps
	allStoredResponseIDs := StoredResponseIDs{}
	// 2) stored bid responses: imp id to bidder to stored response id
	impBiddersWithBidResponseIDs := ImpBiddersWithBidResponseIDs{}
	// 3) imp id to stored resp id
	impAuctionResponseIDs := ImpsWithAuctionResponseIDs{}
	// 4) imp id to bidder to bool replace imp in response
	impBidderReplaceImp := ImpBidderReplaceImpID{}

	for index, impData := range impInfo {
		impId := impData.ID
		impExt, err := impData.GetImpExt()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		impExtPrebid := impExt.GetPrebid()
		if impExtPrebid == nil {
			continue
		}

		if impExtPrebid.StoredAuctionResponse != nil {
			if len(impExtPrebid.StoredAuctionResponse.ID) == 0 {
				return nil, nil, nil, nil, fmt.Errorf("request.imp[%d] has ext.prebid.storedauctionresponse specified, but \"id\" field is missing ", index)
			}
			allStoredResponseIDs = append(allStoredResponseIDs, impExtPrebid.StoredAuctionResponse.ID)

			impAuctionResponseIDs[impId] = impExtPrebid.StoredAuctionResponse.ID

		}
		if len(impExtPrebid.StoredBidResponse) > 0 {

			// bidders can be specified in imp.ext and in imp.ext.prebid.bidders
			allBidderNames := make([]string, 0)
			for bidderName := range impExtPrebid.Bidder {
				allBidderNames = append(allBidderNames, bidderName)
			}
			for extData := range impExt.GetExt() {
				// no bidders will not be processed
				allBidderNames = append(allBidderNames, extData)
			}

			bidderStoredRespId := make(map[string]string)
			bidderReplaceImpId := make(map[string]bool)
			for _, bidderResp := range impExtPrebid.StoredBidResponse {
				if len(bidderResp.ID) == 0 || len(bidderResp.Bidder) == 0 {
					return nil, nil, nil, nil, fmt.Errorf("request.imp[%d] has ext.prebid.storedbidresponse specified, but \"id\" or/and \"bidder\" fields are missing ", index)
				}

				for _, bidderName := range allBidderNames {
					if _, found := bidderStoredRespId[bidderName]; !found && strings.EqualFold(bidderName, bidderResp.Bidder) {
						bidderStoredRespId[bidderName] = bidderResp.ID
						impBiddersWithBidResponseIDs[impId] = bidderStoredRespId

						// stored response config can specify if imp id should be replaced with imp id from request
						replaceImpId := true
						if bidderResp.ReplaceImpId != nil {
							// replaceimpid is true if not specified
							replaceImpId = *bidderResp.ReplaceImpId
						}
						bidderReplaceImpId[bidderName] = replaceImpId
						impBidderReplaceImp[impId] = bidderReplaceImpId

						//storedAuctionResponseIds are not unique, but fetch will return single data for repeated ids
						allStoredResponseIDs = append(allStoredResponseIDs, bidderResp.ID)
					}
				}
			}
		}
	}
	return allStoredResponseIDs, impBiddersWithBidResponseIDs, impAuctionResponseIDs, impBidderReplaceImp, nil
}

// ProcessStoredResponses takes the incoming request as JSON with any
// stored requests/imps already merged into it, scans it to find any stored auction response ids and stored bid response ids
// in the request/imps and produces a map of imp IDs to stored auction responses and map of imp to bidder to stored response.
// Note that processStoredResponses must be called after processStoredRequests
// because stored imps and stored requests can contain stored auction responses and stored bid responses
// so the stored requests/imps have to be merged into the incoming request prior to processing stored auction responses.
func ProcessStoredResponses(ctx context.Context, requestWrapper *openrtb_ext.RequestWrapper, storedRespFetcher stored_requests.Fetcher) (ImpsWithBidResponses, ImpBidderStoredResp, BidderImpReplaceImpID, []error) {

	storedResponsesIds, impBidderToStoredBidResponseId, impIdToRespId, impBidderReplaceImp, err := extractStoredResponsesIds(requestWrapper.GetImp())
	if err != nil {
		return nil, nil, nil, []error{err}
	}

	if len(storedResponsesIds) > 0 {
		storedResponses, errs := storedRespFetcher.FetchResponses(ctx, storedResponsesIds)
		if len(errs) > 0 {
			return nil, nil, nil, errs
		}
		bidderImpIdReplaceImp := flipMap(impBidderReplaceImp)

		impIdToStoredResp, impBidderToStoredBidResponse, errs := buildStoredResponsesMaps(storedResponses, impBidderToStoredBidResponseId, impIdToRespId)

		return impIdToStoredResp, impBidderToStoredBidResponse, bidderImpIdReplaceImp, errs
	}
	return nil, nil, nil, nil
}

// flipMap takes map[impID][bidderName]replaceImpId and modifies it to map[bidderName][impId]replaceImpId
func flipMap(impBidderReplaceImpId ImpBidderReplaceImpID) BidderImpReplaceImpID {
	flippedMap := BidderImpReplaceImpID{}
	for impId, impData := range impBidderReplaceImpId {
		for bidder, replaceImpId := range impData {
			if _, ok := flippedMap[bidder]; !ok {
				flippedMap[bidder] = make(map[string]bool)
			}
			flippedMap[bidder][impId] = replaceImpId
		}
	}
	return flippedMap
}

func buildStoredResponsesMaps(storedResponses StoredResponseIdToStoredResponse, impBidderToStoredBidResponseId ImpBiddersWithBidResponseIDs, impIdToRespId ImpsWithAuctionResponseIDs) (ImpsWithBidResponses, ImpBidderStoredResp, []error) {
	var errs []error
	//imp id to stored resp body
	impIdToStoredResp := ImpsWithBidResponses{}
	//stored bid responses: imp id to bidder to stored response body
	impBidderToStoredBidResponse := ImpBidderStoredResp{}

	for impId, respId := range impIdToRespId {
		if len(storedResponses[respId]) == 0 {
			errs = append(errs, fmt.Errorf("failed to fetch stored auction response for impId = %s and storedAuctionResponse id = %s", impId, respId))
		} else {
			impIdToStoredResp[impId] = storedResponses[respId]
		}
	}

	for impId, bidderStoredResp := range impBidderToStoredBidResponseId {
		bidderStoredResponses := StoredResponseIdToStoredResponse{}
		for bidderName, id := range bidderStoredResp {
			if len(storedResponses[id]) == 0 {
				errs = append(errs, fmt.Errorf("failed to fetch stored bid response for impId = %s, bidder = %s and storedBidResponse id = %s", impId, bidderName, id))
			} else {
				bidderStoredResponses[bidderName] = storedResponses[id]
			}
		}
		impBidderToStoredBidResponse[impId] = bidderStoredResponses
	}
	return impIdToStoredResp, impBidderToStoredBidResponse, errs
}
