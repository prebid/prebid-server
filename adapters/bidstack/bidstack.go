package bidstack

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	currencyUSDISO4217 = "USD"

	headerKeyAuthorization         = "Authorization"
	headerValueAuthorizationBearer = "Bearer "
	headerKeyContentType           = "Content-Type"

	contentTypeApplicationJSON = "application/json"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Bidstack adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}

	return bidder, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers, err := prepareHeaders(request)
	if err != nil {
		return nil, []error{fmt.Errorf("headers prepare: %v", err)}
	}

	for i := range request.Imp {
		imp := &request.Imp[i]
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != currencyUSDISO4217 {
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, currencyUSDISO4217)
			if err != nil {
				return nil, []error{err}
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
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
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
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
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
	bidderImpExt, err := getBidderExt(request.Imp[0])
	if err != nil {
		return nil, fmt.Errorf("get bidder ext: %v", err)
	}

	return http.Header{
		headerKeyContentType:   {contentTypeApplicationJSON},
		headerKeyAuthorization: {headerValueAuthorizationBearer + bidderImpExt.PublisherID},
	}, nil
}

func getBidderExt(imp openrtb2.Imp) (bidderImpExt openrtb_ext.ImpExtBidstack, err error) {
	var impExt adapters.ExtImpBidder
	if err = jsonutil.Unmarshal(imp.Ext, &impExt); err != nil {
		return bidderImpExt, fmt.Errorf("imp ext: %v", err)
	}
	if err = jsonutil.Unmarshal(impExt.Bidder, &bidderImpExt); err != nil {
		return bidderImpExt, fmt.Errorf("bidder ext: %v", err)
	}
	return bidderImpExt, nil
}
