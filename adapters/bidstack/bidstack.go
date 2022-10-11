package bidstack

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/mxmCherry/openrtb/v16/openrtb2"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

const (
	currencyUSDISO4217 = "USD"

	headerKeyAuthorization         = "Authorization"
	headerValueAuthorizationBearer = "Bearer "
	headerKeyContentType           = "Content-Type"

	contentTypeApplicationJSON = "application/json"
)

var (
	ErrNoPublisherID = errors.New("publisher ID is missing")
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Bidstack adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}

	return bidder, nil
}

func (a adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers, err := prepareHeaders(request)
	if err != nil {
		return nil, []error{fmt.Errorf("headers prepare: %v", err)}
	}

	for _, imp := range request.Imp {
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != currencyUSDISO4217 {
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, currencyUSDISO4217)
			if err != nil {
				return nil, []error{fmt.Errorf("currency convert: %v", err)}
			}
			imp.BidFloorCur = currencyUSDISO4217
			imp.BidFloor = convertedValue
		}
	}

	requestJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{fmt.Errorf("bid request marshal: %v", err)}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Headers: headers,
		Body:    requestJSON,
	}

	return []*adapters.RequestData{requestData}, nil
}

func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	switch responseData.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusBadRequest:
		return nil, []error{errors.New("bad request from publisher")}
	case http.StatusOK:
		break
	default:
		return nil, []error{fmt.Errorf("unexpected response status code: %v", responseData.StatusCode)}
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{fmt.Errorf("bid response unmarshal: %v", err)}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur
	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			b := &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: openrtb_ext.BidTypeVideo,
			}
			bidResponse.Bids = append(bidResponse.Bids, b)
		}
	}

	return bidResponse, nil
}

func prepareHeaders(request *openrtb2.BidRequest) (headers http.Header, err error) {
	bidderParams, err := adapters.ExtractReqExtBidderParamsMap(request)
	if err != nil {
		return nil, fmt.Errorf("extract bidder params: %v", err)
	}
	publisherID := strings.ReplaceAll(string(bidderParams["publisherId"]), "\"", "")

	if publisherID == "" {
		return nil, ErrNoPublisherID
	}

	return http.Header{
		headerKeyContentType:   {contentTypeApplicationJSON},
		headerKeyAuthorization: {headerValueAuthorizationBearer + publisherID},
	}, nil
}
