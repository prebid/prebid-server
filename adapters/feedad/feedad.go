package feedad

import (
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const feedAdAdapterVersion = "1.0.0"

func getHeaders(request *openrtb2.BidRequest) http.Header {
	headers := http.Header{}
	headers.Add("Accept", "application/json")
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("X-FA-PBS-Adapter-Version", feedAdAdapterVersion)
	headers.Add("X-Openrtb-Version", "2.5")

	if request.Device != nil {
		if len(request.Device.IPv6) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IPv6)
		}

		if len(request.Device.IP) > 0 {
			headers.Add("X-Forwarded-For", request.Device.IP)
		}
	}

	return headers
}

type adapter struct {
	endpoint string
}

func (a *adapter) MakeBids(internalRequest *openrtb2.BidRequest, externalRequest *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
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
	err := jsonutil.Unmarshal(responseData.Body, &response)
	if err != nil {
		return nil, []error{err}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(internalRequest.Imp))
	bidResponse.Currency = response.Cur

	for _, seatBid := range response.SeatBid {
		for i := range seatBid.Bid {
			bidResponse.Bids = append(
				bidResponse.Bids,
				&adapters.TypedBid{
					Bid:     &seatBid.Bid[i],
					BidType: openrtb_ext.BidTypeBanner,
				},
			)
		}
	}

	return bidResponse, nil
}

func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	requestBody, err := jsonutil.Marshal(request)
	if err != nil {
		return nil, []error{err}
	}

	requestData := &adapters.RequestData{
		Method:  "POST",
		Uri:     a.endpoint,
		Body:    requestBody,
		Headers: getHeaders(request),
		ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
	}
	return []*adapters.RequestData{requestData}, nil
}

func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}
