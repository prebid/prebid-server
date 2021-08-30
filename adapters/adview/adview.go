package adview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adViewAdapter struct {
	EndpointTemplate template.Template
}

// Builder builds a new instance of the Adf adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {

	urlTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adViewAdapter{
		EndpointTemplate: *urlTemplate,
	}
	return bidder, nil
}

func (adapter *adViewAdapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var errors []error
	var validImps = make([]openrtb2.Imp, 0, len(request.Imp))
	requestData := make([]*adapters.RequestData, 0, len(request.Imp))

	for _, imp := range request.Imp {

		var bidderExt adapters.ExtImpBidder
		if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}
		//采用 adview
		var advImpExt openrtb_ext.ExtImpAdview
		if err := json.Unmarshal(bidderExt.Bidder, &advImpExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: err.Error(),
			})
			continue
		}

		imp.TagID = advImpExt.MasterTagID //tagid means posid

		//for adview bid request
		if imp.Banner != nil {
			if len(imp.Banner.Format) != 0 {
				imp.Banner.H = &imp.Banner.Format[0].H
				imp.Banner.W = &imp.Banner.Format[0].W
			}
		}

		validImps = append(validImps, imp)
		request.Imp = validImps

		//make json
		requestJSON, err := json.Marshal(request)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		//end point
		url, err := adapter.buildEndpointURL(&advImpExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		reqData := &adapters.RequestData{
			Method: http.MethodPost,
			Body:   requestJSON,
			Uri:    url,
		}

		requestData = append(requestData, reqData)
	}

	//return []*adapters.RequestData{requestData}, errors
	return requestData, errors
}

func (adapter *adViewAdapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	
	if responseData.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if responseData.StatusCode == http.StatusBadRequest {
		err := &errortypes.BadInput{
			Message: "Unexpected status code: 400. Bad request from publisher.",
		}
		return nil, []error{err}
	}

	if responseData.StatusCode != http.StatusOK {
		err := &errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d.", responseData.StatusCode),
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
			bidType, err := getMediaTypeForImp(bid.ImpID, request.Imp)
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

	return bidResponse, errors
}

// Builds endpoint url based on adapter-specific pub settings from imp.ext
func (adapter *adViewAdapter) buildEndpointURL(params *openrtb_ext.ExtImpAdview) (string, error) {

	endpointParams := macros.EndpointTemplateParams{AccountID: params.AccountID}
	return macros.ResolveMacros(adapter.EndpointTemplate, endpointParams)

}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Banner != nil {
				return openrtb_ext.BidTypeBanner, nil
			} else if imp.Video != nil {
				return openrtb_ext.BidTypeVideo, nil
			} else if imp.Native != nil {
				return openrtb_ext.BidTypeNative, nil
			}
		}
	}

	return "", &errortypes.BadInput{
		Message: fmt.Sprintf("Failed to find supported impression \"%s\" mediatype", impID),
	}
}
