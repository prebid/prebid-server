package adapters

import (
	"bytes"
	"context"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
)

// HttpBidder is the interface which almost all bidders implement.
//
// Its only responsibility is to make HTTP request(s) from a BidRequest, and return Bids from the
// HTTP response(s).
type HttpBidder interface {
	// MakeHttpRequests makes the HTTP requests which should be made to fetch bids.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
	MakeHttpRequests(request *openrtb.BidRequest) ([]*RequestData, []error)

	// MakeBids unpacks the server's response into Bids.
	//
	// The bids can be nil (for no bids), but should not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the server response didn't have the expected format.
	MakeBids(request *openrtb.BidRequest, response *ResponseData) ([]*TypedBid, []error)
}

// AdaptHttpBidder bridges the APIs between a Bidder and an HttpBidder.
func AdaptHttpBidder(bidder HttpBidder, client *http.Client) Bidder {
	return &bidderAdapter{
		Bidder: bidder,
		Client: client,
	}
}

type bidderAdapter struct {
	Bidder HttpBidder
	Client *http.Client
}

func (bidder *bidderAdapter) Bid(ctx context.Context, request *openrtb.BidRequest) (*PBSOrtbSeatBid, []error) {
	reqData, errs := bidder.Bidder.MakeHttpRequests(request)

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
			go func(data *RequestData) {
				responseChannel <- bidder.doRequest(ctx, data)
			}(oneReqData) // Method arg avoids a race condition on oneReqData
		}
	}

	seatBid := &PBSOrtbSeatBid{
		Bids:      make([]*PBSOrtbBid, 0, len(reqData)),
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
				seatBid.Bids = append(seatBid.Bids, &PBSOrtbBid{
					Bid:  bid.Bid,
					Type: bid.BidType,
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
// HttpBidder interface.
func (bidder *bidderAdapter) doRequest(ctx context.Context, req *RequestData) *httpCallInfo {
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
		response: &ResponseData{
			StatusCode: httpResp.StatusCode,
			Body:       respBody,
			Headers:    httpResp.Header,
		},
		err: err,
	}
}

type httpCallInfo struct {
	request  *RequestData
	response *ResponseData
	err      error
}

// TypedBid packages the openrtb.Bid with any bidder-specific information that PBS needs to populate an
// openrtb_ext.ExtBidPrebid.
//
// PBS will use TypedBid.Bid.Ext to populate "response.seatbid[i].bid.ext.bidder" in the final PBS response,
// and the TypedBid.BidType to populate "response.seatbid[i].bid.ext.prebid.type".
//
// All other fields from the openrtb_ext.ExtBidPrebid can be built uniformly across all HttpBidders...
// so there's no reason that each individual bidder needs to send them.
type TypedBid struct {
	Bid     *openrtb.Bid
	BidType openrtb_ext.BidType
}

// ResponseData packages together information from the server's http.Response.
//
// This exists so that prebid-server core code can implement its "debug" API uniformly across all adapters.
// It will also let us test valyala/vasthttp vs. net/http without changing all the adapters
type ResponseData struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// RequestData packages together the fields needed to make an http.Request.
//
// This exists so that prebid-server core code can implement its "debug" API uniformly across all adapters.
// It will also let us test valyala/vasthttp vs. net/http without changing all the adapters
type RequestData struct {
	Method  string
	Uri     string
	Body    []byte
	Headers http.Header
}
