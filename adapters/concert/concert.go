package concert

import (
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/adapters"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/errortypes"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"net/http"
)

type adapter struct {
	endpoint string
}

const adapterVersion = "1.0.0"

// Builder builds a new instance of the Concert adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	// Unmarshal the request into a map
	var requestMap map[string]interface{}
	err = json.Unmarshal(requestJSON, &requestMap)
	if err != nil {
		return nil, []error{err}
	}

	if requestMap["ext"] == nil {
		requestMap["ext"] = make(map[string]interface{})
	}
	requestMap["ext"].(map[string]interface{})["adapterVersion"] = adapterVersion

	// Marshal the map back into JSON
	requestJSON, err = json.Marshal(requestMap)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method: "POST",
		Uri:    a.endpoint,
		Body:   requestJSON,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher. Run with request.debug = 1 for more info.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			imp, _ := getImpByID(bid.ImpID, request.Imp)
			bidType, err := getMediaTypeForBid(bid, imp)
			if err != nil {
				errors = append(errors, err)
				continue
			}
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	if len(errors) > 0 {
		return nil, errors
	}

	if len(bidResponse.Bids) == 0 {
		return nil, []error{fmt.Errorf("no bids returned")}
	}

	return bidResponse, nil
}

func getImpByID(impID string, imps []openrtb2.Imp) (*openrtb2.Imp, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			return &imp, nil
		}
	}
	return nil, fmt.Errorf("no matching imp found for id %s", impID)
}

func getMediaTypeForBid(bid openrtb2.Bid, imp *openrtb2.Imp) (openrtb_ext.BidType, error) {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := json.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			return openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
		}
	}

	if imp != nil {
		if imp.Video != nil {
			return openrtb_ext.BidTypeVideo, nil
		} else if imp.Banner != nil {
			return openrtb_ext.BidTypeBanner, nil
		} else if imp.Audio != nil {
			return openrtb_ext.BidTypeAudio, nil
		}
	}

	return "", &errortypes.BadServerResponse{
		Message: fmt.Sprintf("Failed to parse impression \"%s\" mediatype", bid.ImpID),
	}
}
