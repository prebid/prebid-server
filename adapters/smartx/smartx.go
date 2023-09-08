package smartx

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
)

// adapter is a implementation of the adapters.Bidder interface.
type adapter struct {
	endpointURL string
}

func Builder(_ openrtb_ext.BidderName, config config.Adapter, _ config.Server) (adapters.Bidder, error) {
	//println("Builder smartx", config.Endpoint)
	// initialize the adapter and return it
	return &adapter{
		endpointURL: config.Endpoint,
	}, nil
}

// MakeRequests prepares the HTTP requests which should be made to fetch bids.
func (a *adapter) MakeRequests(openRTBRequest *openrtb2.BidRequest, requestInfo *adapters.ExtraRequestInfo) (requestsToBidder []*adapters.RequestData, errs []error) {
	// parse the requests answer
	openRTBRequestJSON, err := json.MarshalIndent(openRTBRequest, "", "   ")
	if err != nil {
		// can't parse the request, this is an critical error
		errs = append(errs, fmt.Errorf("marshal bidRequest: %w", err))
		return nil, errs
	}

	// create the HEADER of the request
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("x-openrtb-version", "2.5")

	if openRTBRequest.Device != nil {
		if openRTBRequest.Device.UA != "" {
			headers.Set("User-Agent", openRTBRequest.Device.UA)
		}

		if openRTBRequest.Device.IP != "" {
			headers.Set("Forwarded", "for="+openRTBRequest.Device.IP)
			headers.Set("X-Forwarded-For", openRTBRequest.Device.IP)
		}
	}

	// add the new request to the list
	return append(requestsToBidder, &adapters.RequestData{
		Method:  http.MethodPost,
		Uri:     a.endpointURL,
		Body:    openRTBRequestJSON,
		Headers: headers,
	}), nil
}

// MakeBids unpacks the server's response into Bids.
func (a *adapter) MakeBids(request *openrtb2.BidRequest, _ *adapters.RequestData, responseData *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	// check HTTP status code 400 - Bad Request
	// check HTTP status code NOT 200
	if err := adapters.CheckResponseStatusCodeForErrors(responseData); err != nil {
		return nil, []error{err}
	}

	// check HTTP status code 204 - No Content
	if adapters.IsResponseStatusCodeNoContent(responseData) {
		return nil, nil
	}

	var response openrtb2.BidResponse
	if err := json.Unmarshal(responseData.Body, &response); err != nil {
		return nil, []error{err}
	}

	if len(response.SeatBid) == 0 {
		return nil, []error{errors.New("no bidders found in JSON response")}
	}

	bidResponse := adapters.NewBidderResponseWithBidsCapacity(len(request.Imp))
	bidResponse.Currency = response.Cur

	var errs []error

	// loop the SeatBids
	for _, seatBid := range response.SeatBid {
		// loop the Bids
		for i, bid := range seatBid.Bid {
			// get the MType of the bid
			bidType, err := getMediaTypeForBid(bid)

			if err != nil {
				// if an error occures add it to the errors list and continue
				errs = append(errs, err)
				continue
			}

			// add the response to the list
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &seatBid.Bid[i],
				BidType: bidType,
			})
		}
	}

	return bidResponse, errs
}

func getMediaTypeForBid(bid openrtb2.Bid) (openrtb_ext.BidType, error) {
	return openrtb_ext.BidTypeVideo, nil
}
