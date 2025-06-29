package seedtag

import (
	"errors"
	"net/http"
	"strings"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the Seedtag adapter for the given bidder with the given config.
func Builder(_ openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

// MakeRequests makes outgoing HTTP requests to Seedtag endpoint.
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {

	var errors []error
	imps := make([]openrtb2.Imp, 0, len(request.Imp))

	for _, imp := range request.Imp {
		// Convert Floor into USD
		if imp.BidFloor > 0 && imp.BidFloorCur != "" && strings.ToUpper(imp.BidFloorCur) != "USD" {
			convertedValue, err := reqInfo.ConvertCurrency(imp.BidFloor, imp.BidFloorCur, "USD")
			if err != nil {
				errors = append(errors, err)
				continue
			}

			imp.BidFloorCur = "USD"
			imp.BidFloor = convertedValue
		}

		imps = append(imps, imp)
	}

	if len(imps) < 1 {
		return nil, errors
	}

	requestCopy := *request
	requestCopy.Imp = imps
	requestJSON, err := jsonutil.Marshal(requestCopy)
	if err != nil {
		return nil, []error{err}
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	requestData := &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpoint,
		Body:    requestJSON,
		Headers: headers,
		ImpIDs:  openrtb_ext.GetImpIDs(requestCopy.Imp),
	}

	return []*adapters.RequestData{requestData}, errors
}

// MakeBids unpacks the server's response into Bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, requestData *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	err := adapters.CheckResponseStatusCodeForErrors(responseData)
	if err != nil {
		return nil, []error{err}
	}

	var response openrtb2.BidResponse
	if err := jsonutil.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(requestData.ImpIDs))

	var errs []error

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bid := &seatBid.Bid[i]

			bidType, err := getMediaTypeForBid(*bid)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     bid,
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	switch bid.MType {
	case 1:
		return openrtb_ext.BidTypeBanner, nil
	case 2:
		return openrtb_ext.BidTypeVideo, nil
	default:
		return "", errors.New("bid.MType invalid")
	}

}
