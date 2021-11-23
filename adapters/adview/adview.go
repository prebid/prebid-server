package adview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/mxmCherry/openrtb/v15/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
)

type adapter struct {
	endpoint *template.Template
}

// Builder builds a new instance of the adview adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	endpointTemplate, err := template.New("endpointTemplate").Parse(config.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("unable to parse endpoint url template: %v", err)
	}

	bidder := &adapter{
		endpoint: endpointTemplate,
	}
	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var bidderExt adapters.ExtImpBidder
	imp := &request.Imp[0]
	if err := json.Unmarshal(imp.Ext, &bidderExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("invalid imp.ext, %s", err.Error()),
		}}
	}
	//use adview
	var advImpExt openrtb_ext.ExtImpAdView
	if err := json.Unmarshal(bidderExt.Bidder, &advImpExt); err != nil {
		return nil, []error{&errortypes.BadInput{
			Message: fmt.Sprintf("invalid bidderExt.Bidder, %s", err.Error()),
		}}
	}

	imp.TagID = advImpExt.MasterTagID //tagid means posid
	//for adview bid request
	if imp.Banner != nil {
		if len(imp.Banner.Format) != 0 {
			bannerCopy := *imp.Banner
			bannerCopy.H = &imp.Banner.Format[0].H
			bannerCopy.W = &imp.Banner.Format[0].W
			imp.Banner = &bannerCopy
		}
	}

	// Check if imp comes with bid floor amount defined in a foreign currency
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
		// Convert to US dollars
		convertedValue, err := requestInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
		if err != nil {
			return nil, []error{err}
		}
		// Update after conversion. All imp elements inside request.Imp are shallow copies
		// therefore, their non-pointer values are not shared memory and are safe to modify.
		imp.BidFloorCur = "USD"
		imp.BidFloor = convertedValue
	}

	// Set the CUR of bid to USD after converting all floors
	request.Cur = []string{"USD"}

	url, err := a.buildEndpointURL(&advImpExt)
	if err != nil {
		return nil, []error{err}
	}

	reqJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	return []*adapters.RequestData{{
		Method: http.MethodPost,
		Body:   reqJSON,
		Uri:    url,
	}}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	bidResponse.Currency = "USD" //we just support USD for resp

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
func (a *adapter) buildEndpointURL(params *openrtb_ext.ExtImpAdView) (string, error) {
	endpointParams := macros.EndpointTemplateParams{AccountID: params.AccountID}
	return macros.ResolveMacros(a.endpoint, endpointParams)
}

func getMediaTypeForImp(impID string, imps []openrtb2.Imp) (openrtb_ext.BidType, error) {
	mediaType := openrtb_ext.BidTypeBanner
	for _, imp := range imps {
		if imp.ID == impID {
			if imp.Video != nil {
				mediaType = openrtb_ext.BidTypeVideo
			} else if imp.Native != nil {
				mediaType = openrtb_ext.BidTypeNative
			}
			return mediaType, nil
		}
	}
	return mediaType, nil
}
