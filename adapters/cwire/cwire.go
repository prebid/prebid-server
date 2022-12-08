package cwire

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mxmCherry/openrtb/v16/openrtb2"
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
func Builder(bidderName openrtb_ext.BidderName, config config.Adapter) (adapters.Bidder, error) {
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
	var errors []error

	var bidderParams map[int]openrtb_ext.ImpExtCWire

	fmt.Println("REQUEST HAS ARRIVED TO CWIRE ADAPTER")

	for i, imp := range request.Imp {
		var ext struct {
			Bidder openrtb_ext.ImpExtCWire
		}
		err := json.Unmarshal(imp.Ext, &ext)
		if err != nil {
			errors = append(errors, fmt.Errorf("Error while unmarshaling bidder parameters: %v", err))
			continue
		}
		bidderParams[i] = ext.Bidder
	}

	for _, err := range errors {
		fmt.Printf("ERROR WHILE PARSING BIDDER PARAMATER: %v\n", err)
	}

	headers := http.Header{}
	headers.Add("Content-Type", "application/json;charset=utf-8")
	headers.Add("Accept", "application/json")

	body := struct {
		OpenRTBRequest            openrtb2.BidRequest
		ImpressionBidderParamsMap map[int]openrtb_ext.ImpExtCWire
	}{
		OpenRTBRequest:            *request,
		ImpressionBidderParamsMap: bidderParams,
	}

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		errors = append(errors, fmt.Errorf("Error while encoding bidRequest: %v", err))
		return nil, errors
	}

	reqs := []*adapters.RequestData{
		&adapters.RequestData{
			Method:  "POST",
			Uri:     a.endpoint,
			Body:    bodyJSON,
			Headers: headers,
		},
	}

	return reqs, errors
}

/*
The MakeBids method is responsible for parsing the bidding server’s response
and mapping it to the OpenRTB 2.5 Bid Response object model.

This method is called for each response received from your bidding server
within the bidding time window (request.tmax). If there are no requests or if
all requests time out, the MakeBids method will not be called.
*/
func (a *CWireAdapter) MakeBids(bidReq *openrtb2.BidRequest, unused *adapters.RequestData, httpRes *adapters.ResponseData) (*adapters.BidderResponse, []error) {
	var errors []error

	if httpRes.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	var bidResp openrtb2.BidResponse
	if err := json.Unmarshal(httpRes.Body, &bidResp); err != nil {
		return nil, []error{&errortypes.BadServerResponse{
			Message: fmt.Sprintf("Error while decoding response, err: %s", err),
		}}
	}

	bidResponse := adapters.NewBidderResponse()

	for _, sb := range bidResp.SeatBid {
		for _, bid := range sb.Bid {
			bidResponse.Bids = append(bidResponse.Bids, &adapters.TypedBid{
				Bid:     &bid,
				BidType: openrtb_ext.BidTypeBanner,
			})
		}
	}

	return bidResponse, errors
}
