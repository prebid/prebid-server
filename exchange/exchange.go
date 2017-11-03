package exchange

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/adapters"
	"context"
	"time"
	"net/http"
)

type Exchange struct {
	// The list of adapters we will consider for this auction
	adapters []string
	adapterMap map[string]adapters.Bidder
}

func NewExchange(client *http.Client) *Exchange {
	e := new(Exchange)

	e.adapterMap = newAdapterMap(client)
	e.adapters = make([]string, 0, len(e.adapterMap))
	i :=0
	for a, _ := range e.adapterMap {
		e.adapters[i] = a
		i++
	}
}

func (e *Exchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) *openrtb.BidResponse {
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

    adapterBids := e.GetAllBids(ctx, liveAdapters, cleanRequests)

	// Build the response
	return e.BuildBidResponse(liveAdapters, adapterBids, bidRequest)
}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *Exchange) GetAllBids(ctx context.Context, liveAdapters []string, cleanRequests map[string]*openrtb.BidRequest) map[string]*adapters.PBSOrtbSeatBid {
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
			_ = err

			// Add in time reporting
			elapsed := time.Since(start)
			sb.responsetimemillis = elapsed/time.Millisecond
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
func (e *Exchange) BuildBidResponse(liveAdapters []string, adapterBids map[string]*adapters.PBSOrtbSeatBid, bidRequest *openrtb.BidRequest) *openrtb.BidResponse {
	bidResponse := new(openrtb.BidResponse)

	bidResponse.ID = bidRequest.ID
	if len(liveAdapters) == 0 {
		// signal "Invalid Request" if no valid bidders.
		bidResponse.NBR = openrtb.NoBidReasonCode.Ptr(openrtb.NoBidReasonCodeInvalidRequest)
	}

	return bidResponse
}
