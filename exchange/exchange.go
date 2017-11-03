package exchange

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/adapters"
	"context"
	"time"
	"net/http"
	"encoding/json"
)

// Exchange runs an OpenRTB Auction
type Exchange interface {
	HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) *openrtb.BidResponse
}

type exchange struct {
	// The list of adapters we will consider for this auction
	adapters []string
	adapterMap map[string]adapters.Bidder
}

// Container to pass out response ext data from the GetAllBids goroutines back into the main thread
type seatResponseExtra struct {
	ResponseTimeMillis int
	Errors []string
}

func NewExchange(client *http.Client) Exchange {
	e := new(exchange)

	e.adapterMap = newAdapterMap(client)
	e.adapters = make([]string, 0, len(e.adapterMap))
	i :=0
	for a, _ := range e.adapterMap {
		e.adapters[i] = a
		i++
	}
	return e
}

func (e *exchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) *openrtb.BidResponse {
	// Slice of BidRequests, each a copy of the original cleaned to only contain bidder data for the named bidder
	// TODO: modify adapters locally to impliment bseats and wseats
	cleanRequests := openrtb_ext.CleanOpenRTBRequests(bidRequest, e.adapters)
	// List of bidders we have requests for.
	liveAdapters := make([]string, len(cleanRequests))
	i := 0
	for a, _ := range cleanRequests {
		liveAdapters[i] = a
		i++
	}
	// TODO: Possibly sort the list of adapters to support publisher's desired call order, or just randomize it.
	// Currently just implementing randomize
	openrtb_ext.RandomizeList(liveAdapters)

	adapterExtra := make(map[string]*seatResponseExtra)

    adapterBids := e.GetAllBids(ctx, liveAdapters, cleanRequests, adapterExtra)

	// Build the response
	return e.BuildBidResponse(liveAdapters, adapterBids, bidRequest)
}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *exchange) GetAllBids(ctx context.Context, liveAdapters []string, cleanRequests map[string]*openrtb.BidRequest, adapterExtra map[string]*seatResponseExtra) map[string]*adapters.PBSOrtbSeatBid {
	// Set up pointers to the bid results
	adapterBids := map[string]*adapters.PBSOrtbSeatBid{}
	chBids := make(chan int, len(liveAdapters))
	for _, a := range liveAdapters {
		// Here we actually call the adapters and collect the bids.
		go func(aName string) {
			// Passing in aName so a doesn't change out from under the go routine
			start := time.Now()
			sb, err := e.adapterMap[aName].Bid(ctx, cleanRequests[aName])
			// TODO: Error handling

			// Add in time reporting
			elapsed := time.Since(start)
			ae := new(seatResponseExtra)
			ae.ResponseTimeMillis = int(elapsed/time.Millisecond)
			serr := make([]string, len(err))
			for i :=0; i<len(err); i++ {
				serr[i] = err[i].Error()
			}
			ae.Errors = serr
			adapterBids[aName] = sb
			chBids <- 1
		}(a)
	}
	// Wait for the bidders to do their thing
	for i := 0; i < len(liveAdapters); i++ {
		<-chBids
	}

	return adapterBids
}

// This piece takes all the bids supplied by the adapters and crafts an openRTB response to send back to the requester
func (e *exchange) BuildBidResponse(liveAdapters []string, adapterBids map[string]*adapters.PBSOrtbSeatBid, bidRequest *openrtb.BidRequest, adapterExtra map[string]*seatResponseExtra) *openrtb.BidResponse {
	bidResponse := new(openrtb.BidResponse)

	bidResponse.ID = bidRequest.ID
	if len(liveAdapters) == 0 {
		// signal "Invalid Request" if no valid bidders.
		bidResponse.NBR = openrtb.NoBidReasonCode.Ptr(openrtb.NoBidReasonCodeInvalidRequest)
	}

	bidResponseExt := e.MakeExtBidResponse(adapterBids, adapterExtra, bidRequest.Test)
	ext, err := json.Marshal(bidResponseExt)
	// TODO: handle errors
	_ = err
	bidResponse.Ext = ext

	return bidResponse
}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *Exchange) MakeExtBidResponse(adapterBids map[string]*adapters.PBSOrtbSeatBid, adapterExtra map[string]*seatResponseExtra, test int8) openrtb_ext.ExtBidResponse {
	bidResponseExt := new(openrtb_ext.ExtBidResponse)
	if test == 1 {
		bidResponseExt.Debug = new(openrtb_ext.ExtResponseDebug)
	}

	for a, b := range adapterBids {
		if test == 1 {
			// Fill debug info
			bidResponseExt.Debug.ServerCalls[a] = b.ServerCalls
		}
		bidResponseExt.Errors[a] = adapterExtra[a].Errors
		bidResponseExt.ResponseTimeMillis[a] = adapterExtra[a].ResponseTimeMillis
		// Defering the filling of bidResponseExt.Usersync[a] until later

	}
}