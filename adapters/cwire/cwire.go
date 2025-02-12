package cwire

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

/*
Bid adapter implements and exports the requirements:

- The adapters.Builder method to create a new instance of the adapter based on
  the host configuration

- The adapters.Bidder interface consisting of the MakeRequests method to create
  outgoing requests to your bidding server and the MakeBids method to create bid
  responses.
*/

type adapter struct {
	endpoint string
}

// Builder builds a new instance of the CWire adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &adapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

/*
This method creates an HTTP requests that should be sent to the CWire OpenRTB endpoint.
It's only provided with valid impressions for the adapter, it's not called if there is none.
For optimization purposes bid adapters are forbidden from directly initiating any form of
network communication and must entirely rely upon the core framework (adapters.RequestData).
*/
func (a *adapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	resJSON, err := json.Marshal(request)
	if err != nil {
		return nil, []error{fmt.Errorf("Error while encoding OpenRTB BidRequest: %v", err)}
	}

	reqs := []*adapters.RequestData{
		{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    resJSON,
			Headers: headers,
			ImpIDs:  openrtb_ext.GetImpIDs(request.Imp),
		},
	}

	return reqs, nil
}

/*
This method is called for every Bid Response from CWire's OpenRTB endpoint.
It maps the responses to core framework's OpenRTB 2.5 Bid Response object model.
*/
func (a *adapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if httpRes.StatusCode != http.StatusOK {
		return nil, []error{
			fmt.Errorf("Unexpected status code: %d. Run with request.debug = 1 for more info", httpRes.StatusCode),
		}
	}

	var resp openrtb2.BidResponse
	if err := jsonutil.Unmarshal(httpRes.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error while decoding response, err: %s", err),
		}}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = resp.Cur
	for _, sb := range resp.SeatBid {
		for i := range sb.Bid {
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &sb.Bid[i],
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidderResponse, nil
}
