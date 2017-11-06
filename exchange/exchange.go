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
	for a, _ := range e.adapterMap {
		e.adapters = append(e.adapters, a)
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
	return e.BuildBidResponse(liveAdapters, adapterBids, bidRequest, adapterExtra)
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

	// Create the SeatBids. We use a zero sized slice so that we can append non-zero seat bids, and not include seatBid
	// objects for seatBids without any bids. Preallocate the max possible size to avoid reallocating the array as we go.
	seatBids := make([]openrtb.SeatBid, 0, len(liveAdapters))
	for _, a := range liveAdapters {
		if adapterBids[a] != nil && len(adapterBids[a].Bids) > 0 {
			// Only add non-null seat bids
			// Possible performance improvement by grabbing a pointer to the current seatBid element, passing it to
			// MakeSeatBid, and then building the seatBid in place, rather than copying. Probably more confusing than
			// its worth
			sb := e.MakeSeatBid(adapterBids[a], a)
			seatBids = append(seatBids, *sb)
		}
	}

	return bidResponse
}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *exchange) MakeExtBidResponse(adapterBids map[string]*adapters.PBSOrtbSeatBid, adapterExtra map[string]*seatResponseExtra, test int8) *openrtb_ext.ExtBidResponse {
	bidResponseExt := &openrtb_ext.ExtBidResponse{
		Errors: make(map[string][]string, len(adapterBids)),
		ResponseTimeMillis: make(map[string]int, len(adapterBids)),
	}
	if test == 1 {
		bidResponseExt.Debug = &openrtb_ext.ExtResponseDebug{
			ServerCalls: make(map[string][]*openrtb_ext.ExtServerCall),
		}
	}

	for a, b := range adapterBids {
		if b != nil {
			if test == 1 {
				// Fill debug info
				bidResponseExt.Debug.ServerCalls[a] = b.ServerCalls
			}
			bidResponseExt.Errors[a] = adapterExtra[a].Errors
			bidResponseExt.ResponseTimeMillis[a] = adapterExtra[a].ResponseTimeMillis
			// Defering the filling of bidResponseExt.Usersync[a] until later
		}
	}
	return bidResponseExt
}

// Return an openrtb seatBid for a bidder
// BuildBidResponse is responsible for ensuring nil bid seatbids are not included
func (e *exchange) MakeSeatBid(adapterBid *adapters.PBSOrtbSeatBid, adapter string) *openrtb.SeatBid {
	seatBid := new(openrtb.SeatBid)
	seatBid.Seat = adapter
	// Prebid cannot support roadblocking
	seatBid.Group = 0
	sbExt := make(map[string]openrtb.RawJSON)
	sbExt["bidder"] = adapterBid.Ext

	ext, err := json.Marshal(sbExt)
	// TODO: handle errors
	_ = err
	seatBid.Ext = ext

	seatBid.Bid = e.MakeBid(adapterBid.Bids)

	return seatBid
}

// Create the Bid array inside of SeatBid
func (e *exchange) MakeBid(Bids []*adapters.PBSOrtbBid) []openrtb.Bid {
	bids := make([]openrtb.Bid, len(Bids))
	for i := 0; i < len(Bids); i++ {
		bids[i] = *Bids[i].Bid
		bidExt := new(openrtb_ext.ExtBid)
		bidExt.Bidder = bids[i].Ext
		bidPrebid := new(openrtb_ext.ExtBidPrebid)
		bidPrebid.Cache = Bids[i].Cache
		bidPrebid.Type = Bids[i].Type
		// TODO: Support targeting

		ext, err := json.Marshal(bidExt)
		// TODO: handle errors
		_ = err
		bids[i].Ext = ext
	}
	return bids
}