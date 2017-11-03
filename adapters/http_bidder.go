package adapters

import (
	"bytes"
	"context"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"golang.org/x/net/context/ctxhttp"
	"io/ioutil"
	"net/http"
	"time"
)

// HttpBidder is the interface which almost all bidders implement.
//
// Its only responsibility is to make HTTP request(s) from a BidRequest, and return Bids from the
// HTTP response(s).
type SingleHttpBidder interface {
	// MakeHttpRequests makes the HTTP requests which should be made to fetch bids.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
	MakeHttpRequests(request *openrtb.BidRequest) (*RequestData, []error)

	// MakeBids unpacks the server's response into Bids.
	// This method **should not** close the response body. The caller will fully read and close it so that the
	// connections get reused properly.
	//
	// The bids can be nil (for no bids), but should not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the server response didn't have the expected format.
	MakeBids(request *openrtb.BidRequest, response *ResponseData) ([]*TypedBid, []error)
}

// MultiHttpBidder should be implemented by adapters which need to make multiple requests to fetch bids.
// For the best results, Bidders are strongly encouraged to use the SingleHttpBidder instead.
type MultiHttpBidder interface {
	// MakeHttpRequests makes the HTTP requests which should be made to fetch bids.
	//
	// The requests can be nil (if no external calls are needed), but must not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the request contained ad types which this bidder doesn't support.
	MakeHttpRequests(request *openrtb.BidRequest) ([]*RequestData, []error)

	// MakeBids unpacks the server's response into Bids.
	// This method **should not** close the response body. The caller will fully read and close it so that the
	// connections get reused properly.
	//
	// The bids can be nil (for no bids), but should not contain nil elements.
	//
	// The errors should contain a list of errors which explain why this bidder's bids will be
	// "subpar" in some way. For example: the server response didn't have the expected format.
	MakeBids(request *openrtb.BidRequest, response *ResponseData) ([]*TypedBid, []error)
}

// AdaptSingleHttpBidder bridges the APIs between a Bidder and a SingleHttpBidder.
func AdaptSingleHttpBidder(bidderCode string, bidder SingleHttpBidder, client *http.Client) Bidder {
	return &singleBidderAdapter{
		Bidder:     bidder,
		BidderCode: bidderCode,
		Client:     client,
	}
}

type singleBidderAdapter struct {
	Bidder     SingleHttpBidder
	BidderCode string
	Client     *http.Client
}

func (bidder *singleBidderAdapter) Bid(ctx context.Context, request *openrtb.BidRequest) (*PBSOrtbSeatBid, []error) {
	start := time.Now()
	reqData, errs := bidder.Bidder.MakeHttpRequests(request)

	if reqData == nil {
		return nil, errs
	}

	httpReq, err := http.NewRequest("POST", reqData.Uri, bytes.NewBuffer(reqData.Body))
	if err != nil {
		return nil, append(errs, err)
	}
	httpReq.Header = reqData.Headers

	httpResp, err := ctxhttp.Do(ctx, bidder.Client, httpReq)
	seatBid := &PBSOrtbSeatBid{
		ServerCalls: []*openrtb_ext.ExtServerCall{
			{
				Uri:         reqData.Uri,
				RequestBody: string(reqData.Body),
				Status:      -1,
			},
		},
	}
	if err != nil {
		return seatBid, append(errs, err)
	}

	respBody, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return seatBid, append(errs, err)
	}
	defer httpResp.Body.Close()
	seatBid.ServerCalls[0].Status = httpResp.StatusCode
	seatBid.ServerCalls[0].ResponseBody = string(respBody)

	bids, moreErrs := bidder.Bidder.MakeBids(request, &ResponseData{
		StatusCode: httpResp.StatusCode,
		Body:       respBody,
		Headers:    httpResp.Header,
	})
	errs = append(errs, moreErrs...)
	if len(bids) == 0 {
		return seatBid, errs
	}

	responseTime := int(time.Since(start) / time.Millisecond)
	pbsBids := make([]*PBSOrtbBid, 0, len(bids))
	for i := 0; i < len(bids); i++ {
		pbsBids = append(pbsBids, &PBSOrtbBid{
			Bid:                bids[i].Bid,
			Cache:              nil, // TODO: Cache properly
			Type:               bids[i].BidType,
			ResponseTimeMillis: responseTime,
		})
	}
	seatBid.Bids = pbsBids
	return seatBid, errs
}
