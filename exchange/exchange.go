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

// Exchange is capable of running Auctions. It must be threadsafe, and will be shared
// across many goroutines.
type Exchange interface {
	// HoldAuction executes an OpenRTB v2.5 Auction.
	HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) *openrtb.BidResponse
}

type exchange struct {
	// The list of adapters we will consider for this auction
	adapters []openrtb_ext.BidderName
	adapterMap map[openrtb_ext.BidderName]adapters.Bidder
}

// Container to pass out response ext data from the GetAllBids goroutines back into the main thread
type seatResponseExtra struct {
	ResponseTimeMillis int
	Errors []string
}

type bidResponseWrapper struct {
	adapterBids *adapters.PBSOrtbSeatBid
	adapterExtra *seatResponseExtra
	bidder openrtb_ext.BidderName
}

func NewExchange(client *http.Client) Exchange {
	e := new(exchange)

	e.adapterMap = newAdapterMap(client)
	e.adapters = make([]openrtb_ext.BidderName, 0, len(e.adapterMap))
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
	liveAdapters := make([]openrtb_ext.BidderName, len(cleanRequests))
	i := 0
	for a, _ := range cleanRequests {
		liveAdapters[i] = a
		i++
	}
	// TODO: Possibly sort the list of adapters to support publisher's desired call order, or just randomize it.
	// Currently just implementing randomize
	openrtb_ext.RandomizeList(liveAdapters)

	adapterBids, adapterExtra := e.GetAllBids(ctx, liveAdapters, cleanRequests)

	// Build the response
	return e.BuildBidResponse(liveAdapters, adapterBids, bidRequest, adapterExtra)
}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *exchange) GetAllBids(ctx context.Context, liveAdapters []openrtb_ext.BidderName, cleanRequests map[openrtb_ext.BidderName]*openrtb.BidRequest) (map[openrtb_ext.BidderName]*adapters.PBSOrtbSeatBid, map[openrtb_ext.BidderName]*seatResponseExtra) {
	// Set up pointers to the bid results
	adapterBids := map[openrtb_ext.BidderName]*adapters.PBSOrtbSeatBid{}
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra)
	chBids := make(chan *bidResponseWrapper, len(liveAdapters))
	for _, a := range liveAdapters {
		// Here we actually call the adapters and collect the bids.
		go func(aName openrtb_ext.BidderName) {
			// Passing in aName so a doesn't change out from under the go routine
			brw := new(bidResponseWrapper)
			brw.bidder = aName
			start := time.Now()
			bids, err := e.adapterMap[aName].Bid(ctx, cleanRequests[aName])
			// TODO: Error handling

			// Add in time reporting
			elapsed := time.Since(start)
			brw.adapterBids = bids
			// Structure to record extra tracking data generated during bidding
			ae := new(seatResponseExtra)
			ae.ResponseTimeMillis = int(elapsed/time.Millisecond)
			serr := make([]string, len(err))
			for i :=0; i<len(err); i++ {
				serr[i] = err[i].Error()
			}
			ae.Errors = serr
			brw.adapterExtra = ae
			chBids <- brw
		}(a)
	}
	// Wait for the bidders to do their thing
	for i := 0; i < len(liveAdapters); i++ {
		brw := <- chBids
		adapterExtra[brw.bidder] = brw.adapterExtra
		adapterBids[brw.bidder] = brw.adapterBids
	}

	return adapterBids, adapterExtra
}

// This piece takes all the bids supplied by the adapters and crafts an openRTB response to send back to the requester
func (e *exchange) BuildBidResponse(liveAdapters []openrtb_ext.BidderName, adapterBids map[openrtb_ext.BidderName]*adapters.PBSOrtbSeatBid, bidRequest *openrtb.BidRequest, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra) *openrtb.BidResponse {
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
	bidResponse.SeatBid = seatBids

	return bidResponse
}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *exchange) MakeExtBidResponse(adapterBids map[openrtb_ext.BidderName]*adapters.PBSOrtbSeatBid, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, test int8) *openrtb_ext.ExtBidResponse {
	bidResponseExt := &openrtb_ext.ExtBidResponse{
		Errors: make(map[openrtb_ext.BidderName][]string, len(adapterBids)),
		ResponseTimeMillis: make(map[openrtb_ext.BidderName]int, len(adapterBids)),
	}
	if test == 1 {
		bidResponseExt.Debug = &openrtb_ext.ExtResponseDebug{
			HttpCalls: make(map[openrtb_ext.BidderName][]*openrtb_ext.ExtHttpCall),
		}
	}

	for a, b := range adapterBids {
		if b != nil {
			if test == 1 {
				// Fill debug info
				bidResponseExt.Debug.HttpCalls[a] = b.HttpCalls
			}
		}
		// Only make an entry for bidder errors if the bidder reported any.
		if len(adapterExtra[a].Errors) > 0 {
			bidResponseExt.Errors[a] = adapterExtra[a].Errors
		}
		bidResponseExt.ResponseTimeMillis[a] = adapterExtra[a].ResponseTimeMillis
		// Defering the filling of bidResponseExt.Usersync[a] until later

	}
	return bidResponseExt
}

// Return an openrtb seatBid for a bidder
// BuildBidResponse is responsible for ensuring nil bid seatbids are not included
func (e *exchange) MakeSeatBid(adapterBid *adapters.PBSOrtbSeatBid, adapter openrtb_ext.BidderName) *openrtb.SeatBid {
	seatBid := new(openrtb.SeatBid)
	seatBid.Seat = adapter.String()
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