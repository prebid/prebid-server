package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"time"
)

// Exchange runs Auctions. Implementations must be threadsafe, and will be shared across many goroutines.
type Exchange interface {
	// HoldAuction executes an OpenRTB v2.5 Auction.
	HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, usersyncs IdFetcher, to *analytics.AuctionObject) (*openrtb.BidResponse, error)
	LogTransaction(*analytics.AuctionObject)
	IsLoggingEnabled() bool
}

// IdFetcher can find the user's ID for a specific Bidder.
type IdFetcher interface {
	// GetId returns the ID for the bidder. The boolean will be true if the ID exists, and false otherwise.
	GetId(bidder openrtb_ext.BidderName) (string, bool)
}

type exchange struct {
	// The list of adapters we will consider for this auction
	adapters   []openrtb_ext.BidderName
	adapterMap map[openrtb_ext.BidderName]adaptedBidder
	analytics  *analytics.Module
}

// Container to pass out response ext data from the GetAllBids goroutines back into the main thread
type seatResponseExtra struct {
	ResponseTimeMillis int
	Errors             []string
}

type bidResponseWrapper struct {
	adapterBids  *pbsOrtbSeatBid
	adapterExtra *seatResponseExtra
	bidder       openrtb_ext.BidderName
}

func NewExchange(client *http.Client, cfg *config.Configuration, atics *analytics.Module) Exchange {
	e := new(exchange)

	e.adapterMap = newAdapterMap(client, cfg)
	e.adapters = make([]openrtb_ext.BidderName, 0, len(e.adapterMap))
	for a, _ := range e.adapterMap {
		e.adapters = append(e.adapters, a)
	}
	e.analytics = atics
	return e
}

func (e *exchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, usersyncs IdFetcher, to *analytics.AuctionObject) (*openrtb.BidResponse, error) {
	// Slice of BidRequests, each a copy of the original cleaned to only contain bidder data for the named bidder
	cleanRequests, errs := cleanOpenRTBRequests(bidRequest, e.adapters, usersyncs)

	// List of bidders we have requests for.
	liveAdapters := make([]openrtb_ext.BidderName, len(cleanRequests))
	i := 0
	for a, _ := range cleanRequests {
		liveAdapters[i] = a
		i++
	}
	// Randomize the list of adapters to make the auction more fair
	randomizeList(liveAdapters)
	// Process the request to check for targeting parameters.
	var targData *targetData = nil
	if len(bidRequest.Ext) > 0 {
		var requestExt openrtb_ext.ExtRequest
		err := json.Unmarshal(bidRequest.Ext, &requestExt)
		if err != nil {
			return nil, fmt.Errorf("Error decoding Request.ext : %s", err.Error())
		}
		if requestExt.Prebid.Targeting != nil {
			targData = &targetData{
				lengthMax:        requestExt.Prebid.Targeting.MaxLength,
				priceGranularity: requestExt.Prebid.Targeting.PriceGranularity,
				winningBids:      make(map[string]*openrtb.Bid, len(bidRequest.Imp)),
				winningBidders:   make(map[string]openrtb_ext.BidderName, len(bidRequest.Imp)),
			}
		}
	}

	adapterBids, adapterExtra := e.getAllBids(ctx, liveAdapters, cleanRequests, targData, to)

	// Build the response
	return e.buildBidResponse(liveAdapters, adapterBids, bidRequest, adapterExtra, targData, errs)

}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *exchange) getAllBids(ctx context.Context, liveAdapters []openrtb_ext.BidderName, cleanRequests map[openrtb_ext.BidderName]*openrtb.BidRequest, targData *targetData, to *analytics.AuctionObject) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, map[openrtb_ext.BidderName]*seatResponseExtra) {
	// Set up pointers to the bid results
	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid, len(liveAdapters))
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, len(liveAdapters))
	chBids := make(chan *bidResponseWrapper, len(liveAdapters))
	for _, a := range liveAdapters {
		// Here we actually call the adapters and collect the bids.
		go func(aName openrtb_ext.BidderName) {
			// Passing in aName so a doesn't change out from under the go routine
			brw := new(bidResponseWrapper)
			brw.bidder = aName
			start := time.Now()
			bids, err := e.adapterMap[aName].requestBid(ctx, cleanRequests[aName], targData, aName, to)

			// Add in time reporting
			elapsed := time.Since(start)
			brw.adapterBids = bids
			// Structure to record extra tracking data generated during bidding
			ae := new(seatResponseExtra)
			ae.ResponseTimeMillis = int(elapsed / time.Millisecond)
			serr := make([]string, len(err))
			for i := 0; i < len(err); i++ {
				serr[i] = err[i].Error()
			}
			ae.Errors = serr
			brw.adapterExtra = ae
			chBids <- brw
		}(a)
	}
	// Wait for the bidders to do their thing
	for i := 0; i < len(liveAdapters); i++ {
		brw := <-chBids
		adapterExtra[brw.bidder] = brw.adapterExtra
		adapterBids[brw.bidder] = brw.adapterBids
	}

	return adapterBids, adapterExtra
}

// This piece takes all the bids supplied by the adapters and crafts an openRTB response to send back to the requester
func (e *exchange) buildBidResponse(liveAdapters []openrtb_ext.BidderName, adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, bidRequest *openrtb.BidRequest, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, targData *targetData, errList []error) (*openrtb.BidResponse, error) {
	bidResponse := new(openrtb.BidResponse)

	bidResponse.ID = bidRequest.ID
	if len(liveAdapters) == 0 {
		// signal "Invalid Request" if no valid bidders.
		bidResponse.NBR = openrtb.NoBidReasonCode.Ptr(openrtb.NoBidReasonCodeInvalidRequest)
	}

	// Create the SeatBids. We use a zero sized slice so that we can append non-zero seat bids, and not include seatBid
	// objects for seatBids without any bids. Preallocate the max possible size to avoid reallocating the array as we go.
	seatBids := make([]openrtb.SeatBid, 0, len(liveAdapters))
	for _, a := range liveAdapters {
		if adapterBids[a] != nil && len(adapterBids[a].bids) > 0 {
			// Only add non-null seat bids
			// Possible performance improvement by grabbing a pointer to the current seatBid element, passing it to
			// MakeSeatBid, and then building the seatBid in place, rather than copying. Probably more confusing than
			// its worth
			sb := e.makeSeatBid(adapterBids[a], a, adapterExtra, targData)
			seatBids = append(seatBids, *sb)
		}
	}
	targData.addWinningTargets()
	bidResponse.SeatBid = seatBids

	bidResponseExt := e.makeExtBidResponse(adapterBids, adapterExtra, bidRequest.Test, errList)
	ext, err := json.Marshal(bidResponseExt)
	bidResponse.Ext = ext
	return bidResponse, err
}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *exchange) makeExtBidResponse(adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, test int8, errList []error) *openrtb_ext.ExtBidResponse {
	bidResponseExt := &openrtb_ext.ExtBidResponse{
		Errors:             make(map[openrtb_ext.BidderName][]string, len(adapterBids)),
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
				bidResponseExt.Debug.HttpCalls[a] = b.httpCalls
			}
		}
		// Only make an entry for bidder errors if the bidder reported any.
		if len(adapterExtra[a].Errors) > 0 {
			bidResponseExt.Errors[a] = adapterExtra[a].Errors
		}
		if len(errList) > 0 {
			s := make([]string, len(errList))
			for i := 0; i < len(errList); i++ {
				s[i] = errList[i].Error()
			}
			bidResponseExt.Errors["prebid"] = s
		}
		bidResponseExt.ResponseTimeMillis[a] = adapterExtra[a].ResponseTimeMillis
		// Defering the filling of bidResponseExt.Usersync[a] until later

	}
	return bidResponseExt
}

// Return an openrtb seatBid for a bidder
// BuildBidResponse is responsible for ensuring nil bid seatbids are not included
func (e *exchange) makeSeatBid(adapterBid *pbsOrtbSeatBid, adapter openrtb_ext.BidderName, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, targData *targetData) *openrtb.SeatBid {
	seatBid := new(openrtb.SeatBid)
	seatBid.Seat = adapter.String()
	// Prebid cannot support roadblocking
	seatBid.Group = 0

	if len(adapterBid.ext) > 0 {
		sbExt := ExtSeatBid{
			Bidder: adapterBid.ext,
		}

		ext, err := json.Marshal(sbExt)
		if err != nil {
			adapterExtra[adapter].Errors = append(adapterExtra[adapter].Errors, fmt.Sprintf("Error writing SeatBid.Ext: %s", err.Error()))
		}
		seatBid.Ext = ext
	}

	var errList []string
	seatBid.Bid, errList = e.makeBid(adapterBid.bids, targData, adapter)
	if len(errList) > 0 {
		adapterExtra[adapter].Errors = append(adapterExtra[adapter].Errors, errList...)
	}

	return seatBid
}

// Create the Bid array inside of SeatBid
func (e *exchange) makeBid(Bids []*pbsOrtbBid, targData *targetData, adapter openrtb_ext.BidderName) ([]openrtb.Bid, []string) {
	bids := make([]openrtb.Bid, 0, len(Bids))
	errList := make([]string, 0, 1)
	for i, thisBid := range Bids {
		bidExt := &openrtb_ext.ExtBid{
			Bidder: thisBid.bid.Ext,
			Prebid: &openrtb_ext.ExtBidPrebid{
				Targeting: thisBid.bidTargets,
				Type:      thisBid.bidType,
			},
		}

		ext, err := json.Marshal(bidExt)
		if err != nil {
			errList = append(errList, fmt.Sprintf("Error writing SeatBid.Bid[%d].Ext: %s", i, err.Error()))
		} else {
			bids = append(bids, *thisBid.bid)
			targData.addBid(adapter, &(bids[len(bids)-1]))
			bids[len(bids)-1].Ext = ext
		}
	}
	return bids, errList
}

func (e *exchange) LogTransaction(to *analytics.AuctionObject) {
	(*e.analytics).LogToModule(to)
}

func (e *exchange) IsLoggingEnabled() bool {
	return *e.analytics != nil
}
