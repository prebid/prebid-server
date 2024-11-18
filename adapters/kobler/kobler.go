package kobler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint    string
	devEndpoint string
}

const (
	devBidderEndpoint = "https://bid-service.dev.essrtb.com/bid/prebid_server_rtb_call"
	supportedCurrency = "USD"
)

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint:    config.Endpoint,
		devEndpoint: devBidderEndpoint,
	}

	return bidder, nil
}

func (a adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	var requestData []*adapters.RequestData
	var errors []error

	if !contains(request.Cur, supportedCurrency) {
		request.Cur = append(request.Cur, supportedCurrency)
	}

	for i := range request.Imp {
		if err := convertImpCurrency(&request.Imp[i], reqInfo); err != nil {
			errors = append(errors, err)
			return nil, errors
		}
	}

	requestJSON, err := jsonutil.Marshal(request)
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")

	requestData = append(requestData, &adapters.RequestData{
		Method:  "POST",
		Uri:     a.getEndpoint(request),
		Body:    requestJSON,
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		Headers: headers,
	})

	return requestData, nil
}

func (a adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if responseData.StatusCode == http.StatusNoContent || responseData.Body == nil {
		return nil, nil
	}

	if responseData.StatusCode != http.StatusOK {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Unexpected status code: %d. Run with request.debug = 1 for more info.", responseData.StatusCode),
		}}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i, bid := range seatBid.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: getMediaTypeForBid(bid),
			})
		}
	}

	return bidResponse, nil
}

func (a adapter) getEndpoint(request *openrtb2.BidRequest) string {
	if request.Test == 1 {
		return a.devEndpoint
	}

	return a.endpoint
}

func getMediaTypeForBid(bid openrtb2.Bid) openrtb_ext.BidType {
	if bid.Ext != nil {
		var bidExt openrtb_ext.ExtBid
		err := jsonutil.Unmarshal(bid.Ext, &bidExt)
		if err == nil && bidExt.Prebid != nil {
			mediaType, err := openrtb_ext.ParseBidType(string(bidExt.Prebid.Type))
			if err == nil {
				return mediaType
			}
		}
	}

	return openrtb_ext.BidTypeBanner
}

func convertImpCurrency(imp *openrtb2.Imp, reqInfo *adapters.ExtraRequestInfo) error {
	if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != supportedCurrency {
		convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, supportedCurrency)
		if err != nil {
			return err
		}

		imp.BidFloor = convertedValue
		imp.BidFloorCur = supportedCurrency
	}

	return nil
}

func contains[T comparable](array []T, value T) bool {
	for _, item := range array {
		if item == value {
			return true
		}
	}

	return false
}
