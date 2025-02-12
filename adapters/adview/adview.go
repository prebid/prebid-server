package adview

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"text/template"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/macros"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint *template.Template
}

type adviewBidExt struct {
	BidType int `json:"formattype,omitempty"`
}

// Builder builds a new instance of the adview adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
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

	var requests []*adapters.RequestData
	var errors []error

	//must copy the original request.
	requestCopy := *request
	for _, imp := range request.Imp {
		var bidderExt adapters.ExtImpBidder
		if err := jsonutil.Unmarshal(imp.Ext, &bidderExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid imp.ext, %s", err.Error()),
			})
			continue
		}
		//use adview
		var advImpExt openrtb_ext.ExtImpAdView
		if err := jsonutil.Unmarshal(bidderExt.Bidder, &advImpExt); err != nil {
			errors = append(errors, &errortypes.BadInput{
				Message: fmt.Sprintf("invalid bidderExt.Bidder, %s", err.Error()),
			})
			continue
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
				errors = append(errors, err)
				continue
			}
			// Update after conversion. All imp elements inside request.Imp are shallow copies
			// therefore, their non-pointer values are not shared memory and are safe to modify.
			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		// Set the CUR of bid to USD after converting all floors
		requestCopy.Cur = []string{"USD"}
		requestCopy.Imp = []openrtb2.Imp{imp}

		url, err := a.buildEndpointURL(&advImpExt)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		reqJSON, err := json.Marshal(requestCopy) //request
		if err != nil {
			errors = append(errors, err)
			continue
		}

		requestData := &adapters.RequestData{
			Method: http.MethodPost,
			Uri:    url,
			Body:   reqJSON,
			ImpIDs: openrtb_ext.GetImpIDs(requestCopy.Imp),
		}
		requests = append(requests, requestData)
	}
	return requests, errors
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
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	//we just support USD for resp
	bidResponse.Currency = "USD"

	var errors []error
	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidType, err := getMediaTypeForBid(bid)
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

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case openrtb2.MarkupBanner:
		return openrtb_ext.BidTypeBanner, nil
	case openrtb2.MarkupVideo:
		return openrtb_ext.BidTypeVideo, nil
	case openrtb2.MarkupNative:
		return openrtb_ext.BidTypeNative, nil
	default:
		return "", fmt.Errorf("Unable to fetch mediaType in impID: %s, mType: %d", bid.ImpID, bid.MType)
	}
}
