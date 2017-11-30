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
	"github.com/prebid/prebid-server/pbs/buckets"
	"strconv"
)

// adaptedBidder defines the contract needed to participate in an Auction within an Exchange.
//
// This interface exists to help segregate core auction logic.
//
// Any logic which can be done _within a single Seat_ goes inside one of these.
// Any logic which _requires responses from all Seats_ goes inside the Exchange.
//
// This interface differs from adapters.Bidder to help minimize code duplication across the
// adapters.Bidder implementations.
type adaptedBidder interface {
	// requestBid fetches bids for the given request.
	//
	// An adaptedBidder *may* return non-nil values here. Errors should describe situations which
	// make the bid (or no-bid) "less than ideal." Common examples include:
	//
	// 1. Connection issues.
	// 2. Imps with Media Types which this Bidder doesn't support.
	// 3. The Context timeout expired before all expected bids were returned.
	// 4. The Server sent back an unexpected Response, so some bids were ignored.
	//
	// Any errors will be user-facing in the API.
	// Error messages should help publishers understand what might account for "bad" bids.
	requestBid(ctx context.Context, request *openrtb.BidRequest, bidderTarg *bidderTargeting) (*pbsOrtbSeatBid, []error)
}

// pbsOrtbBid is a Bid returned by an adaptedBidder.
//
// pbsOrtbBid.Bid.Ext will become "response.seatbid[i].bid.ext.bidder" in the final OpenRTB response.
// pbsOrtbBid.BidType will become "response.seatbid[i].bid.ext.prebid.type" in the final OpenRTB response.
type pbsOrtbBid struct {
	bid *openrtb.Bid
	bidType openrtb_ext.BidType
	bidTargets map[string]string
}

// pbsOrtbSeatBid is a SeatBid returned by an adaptedBidder.
//
// This is distinct from the openrtb.SeatBid so that the prebid-server ext can be passed back with typesafety.
type pbsOrtbSeatBid struct {
	// bids is the list of bids which this adaptedBidder wishes to make.
	bids []*pbsOrtbBid
	// httpcalls is the list of debugging info. It should only be populated if the request.test == 1.
	// This will become response.ext.debug.httpcalls.{bidder} on the final Response.
	httpCalls []*openrtb_ext.ExtHttpCall
	// ext contains the extension for this seatbid.
	// if len(bids) > 0, this will become response.seatbid[i].ext.{bidder} on the final OpenRTB response.
	// if len(bids) == 0, this will be ignored because the OpenRTB spec doesn't allow a SeatBid with 0 Bids.
	ext openrtb.RawJSON
}

// adaptBidder converts an adapters.Bidder into an exchange.adaptedBidder.
//
// The name refers to the "Adapter" architecture pattern, and should not be confused with a Prebid "Adapter"
// (which is being phased out and replaced by Bidder for OpenRTB auctions)
func adaptBidder(bidder adapters.Bidder, client *http.Client) adaptedBidder {
	return &bidderAdapter{
		Bidder: bidder,
		Client: client,
	}
}

type bidderAdapter struct {
	Bidder adapters.Bidder
	Client *http.Client
}

func (bidder *bidderAdapter) requestBid(ctx context.Context, request *openrtb.BidRequest, bidderTarg *bidderTargeting) (*pbsOrtbSeatBid, []error) {
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
		httpCalls: make([]*openrtb_ext.ExtHttpCall, 0, len(reqData)),
	}

	// If the bidder made multiple requests, we still want them to enter as many bids as possible...
	// even if the timeout occurs sometime halfway through.
	for i := 0; i < len(reqData); i++ {
		httpInfo := <-responseChannel
		// If this is a test bid, capture debugging info from the requests.
		if request.Test == 1 {
			seatBid.httpCalls = append(seatBid.httpCalls, makeExt(httpInfo))
		}

		if httpInfo.err == nil {
			bids, moreErrs := bidder.Bidder.MakeBids(request, httpInfo.response)
			errs = append(errs, moreErrs...)
			for _, bid := range bids {
				var targets map[string]string
				targets = nil
				if bidderTarg.targetFlag {
					cpm := bid.Bid.Price
					width := bid.Bid.W
					height := bid.Bid.H
					deal := bid.Bid.DealID
					cacheKey := ""
					var err error
					targets, err = bidder.makePrebidTargets(cpm, width, height, cacheKey, deal, bidderTarg)
					if err != nil {
						errs = append(errs, err)
						// set CPM to 0 if we could not bucket it
						cpm = 0.0
					}
				}

				seatBid.bids = append(seatBid.bids, &pbsOrtbBid{
					bid:  bid.Bid,
					bidType: bid.BidType,
					bidTargets: targets,
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

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (bidder *bidderAdapter) makePrebidTargets(cpm float64, width uint64, height uint64, cache string, deal string, bidderTarg *bidderTargeting) (map[string]string, error) {
	roundedCpm, err := buckets.GetPriceBucketString(cpm, bidderTarg.priceGranularity)

	hbSize := ""
	if width != 0 && height != 0 {
		w := strconv.FormatUint(width, 10)
		h := strconv.FormatUint(height, 10)
		hbSize = w + "x" + h
	}

	hbPbBidderKey := string(hbpbConstantKey + "_" + bidderTarg.bidder)
	hbBidderBidderKey := string(hbBidderConstantKey + "_" + bidderTarg.bidder)
	hbSizeBidderKey := string(hbSizeConstantKey + "_" + bidderTarg.bidder)
	hbDealIdBidderKey := string(hbDealIdConstantKey + "_" + bidderTarg.bidder)
	hbCacheIdBidderKey := string(hbCacheIdConstantKey + "_" + bidderTarg.bidder)
	if bidderTarg.lengthMax != 0 {
		hbPbBidderKey = hbPbBidderKey[:min(len(hbPbBidderKey), int(bidderTarg.lengthMax))]
		hbBidderBidderKey = hbBidderBidderKey[:min(len(hbBidderBidderKey), int(bidderTarg.lengthMax))]
		hbSizeBidderKey = hbSizeBidderKey[:min(len(hbSizeBidderKey), int(bidderTarg.lengthMax))]
		hbCacheIdBidderKey = hbCacheIdBidderKey[:min(len(hbSizeBidderKey), int(bidderTarg.lengthMax))]
		hbDealIdBidderKey = hbDealIdBidderKey[:min(len(hbSizeBidderKey), int(bidderTarg.lengthMax))]
	}
	pbs_kvs := map[string]string{
		hbPbBidderKey:      roundedCpm,
		hbBidderBidderKey:  string(bidderTarg.bidder),
	}

	if hbSize != "" {
		pbs_kvs[hbSizeBidderKey] = hbSize
	}
	if len(cache) > 0 {
		pbs_kvs[hbCacheIdBidderKey] = cache
	}
	if len(deal) > 0 {
		pbs_kvs[hbDealIdBidderKey] = deal
	}
	return pbs_kvs, err
}