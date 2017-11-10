package exchange

import (
	"context"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"bytes"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"fmt"
	"net/http"
	"github.com/prebid/prebid-server/adapters"
)

// bidder defines the contract needed to participate in an Auction within an Exchange.
//
// This interface exists to help segregate core auction code.
//
// Any work which can be done _within a single Seat_ goes inside one of these.
// Any work which _requires responses from all Seats_ goes inside the Exchange.
//
// This interface differs from adapters.Bidder to help minimize code duplication across the
// adapters.Bidder implementations.
type bidder interface {
	// Bid gets the bids from this bidder for the given request.
	//
	// Per the OpenRTB spec, a SeatBid may not be empty. If so, then any errors which contribute
	// to the "no bid" bid should be returned here instead.
	//
	// A Bidder *may* return two non-nil values here. Errors should describe situations which
	// make the bid (or no-bid) "less than ideal." Common examples include:
	//
	// 1. Connection issues.
	// 2. Imps with Media Types which this Bidder doesn't support.
	// 3. The Context timeout expired before all expected bids were returned.
	// 4. The Server sent back an unexpected Response, so some bids were ignored.
	//
	// Any errors will be user-facing in the API.
	// Error messages should help publishers understand what might account for "bad" bids.
	requestBid(ctx context.Context, request *openrtb.BidRequest) (*pbsOrtbSeatBid, []error)
}

// pbsOrtbBid is a Bid returned by a Bidder.
//
// pbsOrtbBid.Bid.Ext will become "response.seatbid[bidder].bid[i].ext.bidder" in the final PBS response.
type pbsOrtbBid struct {
	bid *openrtb.Bid
	bidType openrtb_ext.BidType
}

// pbsOrtbBid is a SeatBid returned by a Bidder.
//
// PBS does not support the "Group" option from the OpenRTB SeatBid. All bids must be winnable independently.
type pbsOrtbSeatBid struct {
	// Bids is the list of bids in this SeatBid. If len(Bids) == 0, no SeatBid will be entered for this bidder.
	// This is because the OpenRTB 2.5 spec requires at least one bid for each SeatBid.
	bids []*pbsOrtbBid
	// HttpCalls will become response.ext.debug.httpcalls.{bidder} on the final Response.
	HttpCalls []*openrtb_ext.ExtHttpCall
	// Ext will become response.seatbid[i].ext.{bidder} on the final Response, *only if* len(Bids) > 0.
	// If len(Bids) == 0, no SeatBid will be entered, and this field will be ignored.
	Ext openrtb.RawJSON
}

// adaptBidder converts an adapters.Bidder into an exchange.Bidder.
//
// The name refers to the "Adapter" architecture pattern, and should not be confused with a Prebid "Adapter"
// (which is being phased out and replaced by Bidder for OpenRTB auctions)
func adaptBidder(bidder adapters.Bidder, client *http.Client) bidder {
	return &bidderAdapter{
		Bidder: bidder,
		Client: client,
	}
}

type bidderAdapter struct {
	Bidder adapters.Bidder
	Client *http.Client
}

func (bidder *bidderAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest) (*pbsOrtbSeatBid, []error) {
	reqData, errs := bidder.Bidder.MakeRequests(request)

	if len(reqData) == 0 {
		return nil, errs
	}

	// Make any HTTP requests in parallel.
	// If the bidder only needs to make one, save some cycles by just using the current one.
	responseChannel := make(chan *httpCallInfo, len(reqData))
	if len(reqData) == 1 {
		responseChannel <- bidder.doRequest(ctx, reqData[0])
	} else {
		for _, oneReqData := range reqData {
			go func(data *adapters.RequestData) {
				responseChannel <- bidder.doRequest(ctx, data)
			}(oneReqData) // Method arg avoids a race condition on oneReqData
		}
	}

	seatBid := &pbsOrtbSeatBid{
		bids:      make([]*pbsOrtbBid, 0, len(reqData)),
		HttpCalls: make([]*openrtb_ext.ExtHttpCall, 0, len(reqData)),
	}

	// If the bidder made multiple requests, we still want them to enter as many bids as possible...
	// even if the timeout occurs sometime halfway through.
	for i := 0; i < len(reqData); i++ {
		httpInfo := <-responseChannel
		// If this is a test bid, capture debugging info from the requests.
		if request.Test == 1 {
			seatBid.HttpCalls = append(seatBid.HttpCalls, makeExt(httpInfo))
		}

		if httpInfo.err == nil {
			bids, moreErrs := bidder.Bidder.MakeBids(request, httpInfo.response)
			errs = append(errs, moreErrs...)
			for _, bid := range bids {
				seatBid.bids = append(seatBid.bids, &pbsOrtbBid{
					bid:  bid.Bid,
					bidType: bid.BidType,
				})
			}
		} else {
			errs = append(errs, httpInfo.err)
		}
	}

	return seatBid, errs
}

// makeExt transforms information about the HTTP call into the contract class for the PBS response.
func makeExt(httpInfo *httpCallInfo) *openrtb_ext.ExtHttpCall {
	if httpInfo.err == nil {
		return &openrtb_ext.ExtHttpCall{
			Uri:          httpInfo.request.Uri,
			RequestBody:  string(httpInfo.request.Body),
			ResponseBody: string(httpInfo.response.Body),
			Status:       httpInfo.response.StatusCode,
		}
	} else if httpInfo.request == nil {
		return &openrtb_ext.ExtHttpCall{}
	} else {
		return &openrtb_ext.ExtHttpCall{
			Uri:         httpInfo.request.Uri,
			RequestBody: string(httpInfo.request.Body),
		}
	}
}

// doRequest makes a request, handles the response, and returns the data needed by the
// Bidder interface.
func (bidder *bidderAdapter) doRequest(ctx context.Context, req *adapters.RequestData) *httpCallInfo {
	httpReq, err := http.NewRequest(req.Method, req.Uri, bytes.NewBuffer(req.Body))
	if err != nil {
		return &httpCallInfo{
			request: req,
			err:     err,
		}
	}
	httpReq.Header = req.Headers

	httpResp, err := ctxhttp.Do(ctx, bidder.Client, httpReq)
	if err != nil {
		return &httpCallInfo{
			request: req,
			err:     err,
		}
	}

	respBody, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return &httpCallInfo{
			request: req,
			err:     err,
		}
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 400 {
		err = fmt.Errorf("Server responded with failure status: %d. Set request.test = 1 for debugging info.", httpResp.StatusCode)
	}

	return &httpCallInfo{
		request: req,
		response: &adapters.ResponseData{
			StatusCode: httpResp.StatusCode,
			Body:       respBody,
			Headers:    httpResp.Header,
		},
		err: err,
	}
}

type httpCallInfo struct {
	request  *adapters.RequestData
	response *adapters.ResponseData
	err      error
}
