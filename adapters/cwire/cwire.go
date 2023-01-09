package cwire

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

/*
Your bid adapter code will need to implement and export:

- The adapters.Builder method to create a new instance of the adapter based on
  the host configuration

- The adapters.Bidder interface consisting of the MakeRequests method to create
  outgoing requests to your bidding server and the MakeBids method to create bid
  responses.
*/

type CWireAdapter struct {
	endpoint string
}

// Builder builds a new instance of the CWire adapter for the given bidder with the given config.
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter, server config.Server) (adapters.Bidder, error) {
	bidder := &CWireAdapter{
		endpoint: config.Endpoint,
	}
	return bidder, nil
}

/*
The MakeRequests method is responsible for returning none, one, or many HTTP
requests to be sent to your bidding server. Bid adapters are forbidden from
directly initiating any form of network communication and must entirely rely
upon the core framework. This allows the core framework to optimize outgoing
connections using a managed pool and record networking metrics. The return type
adapters.RequestData allows your adapter to specify the HTTP method, url, body,
and headers.

This method is called once by the core framework for bid requests which have at
least one valid Impression for your adapter. Impressions not configured for
your adapter are not accessible.
*/
func (a *CWireAdapter) MakeRequests(request *openrtb2.BidRequest, reqInfo *adapters.ExtraRequestInfo) ([]*adapters.RequestData, []error) {
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
		},
	}

	return reqs, nil
}

/*
The MakeBids method is responsible for parsing the bidding server’s response
and mapping it to the OpenRTB 2.5 Bid Response object model.

This method is called for each response received from your bidding server
within the bidding time window (request.tmax). If there are no requests or if
all requests time out, the MakeBids method will not be called.
*/
func (a *CWireAdapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var resp openrtb2.BidResponse
	if err := json.Unmarshal(httpRes.Body, &resp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error while decoding response, err: %s", err),
		}}
	}

	bidderResponse := adapters.NewBidderResponse()
	bidderResponse.Currency = resp.Cur
	for _, sb := range resp.SeatBid {
		for _, bid := range sb.Bid {
			bidderResponse.Bids = append(bidderResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidderResponse, nil
}
