package exchange

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"context"
	"time"
	"net/http"
	"encoding/json"
	"fmt"
)

// Exchange runs Auctions. Implementations must be threadsafe, and will be shared across many goroutines.
type Exchange interface {
	// HoldAuction executes an OpenRTB v2.5 Auction.
	HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) (*openrtb.BidResponse, error)
}

type exchange struct {
	// The list of adapters we will consider for this auction
	adapters []openrtb_ext.BidderName
	adapterMap map[openrtb_ext.BidderName]adaptedBidder
}

// Container to pass out response ext data from the GetAllBids goroutines back into the main thread
type seatResponseExtra struct {
	ResponseTimeMillis int
	Errors []string
}

type bidResponseWrapper struct {
	adapterBids *pbsOrtbSeatBid
	adapterExtra *seatResponseExtra
	bidder openrtb_ext.BidderName
}

// Container for storing a pointer to the winning bid.
type targetData struct {
	targetFlag bool
	lengthMax int
	priceGranularity openrtb_ext.PriceGranularity
	// These two dictionaries index on imp.id to identify the winning bid for each imp.
	winningBids map[string]*openrtb.Bid
	winningBidders map[string]openrtb_ext.BidderName
}

type bidderTargeting struct {
	targetFlag bool
	lengthMax int
	priceGranularity openrtb_ext.PriceGranularity
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

func (e *exchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest) (*openrtb.BidResponse, error) {
	// Slice of BidRequests, each a copy of the original cleaned to only contain bidder data for the named bidder
	cleanRequests, errs := cleanOpenRTBRequests(bidRequest, e.adapters)
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
	targData := &targetData{
		targetFlag:	false,
		lengthMax:	0,
		priceGranularity:	openrtb_ext.PriceGranularityMedium,
		winningBids:		make(map[string]*openrtb.Bid, len(bidRequest.Imp)),
		winningBidders:		make(map[string]openrtb_ext.BidderName, len(bidRequest.Imp)),
	}
	requestExt := new(openrtb_ext.ExtRequest)
	err := json.Unmarshal(bidRequest.Ext, requestExt)
	if err != nil {
		return nil, fmt.Errorf("Error decoding Request.ext : %s", err.Error())
	}
	if requestExt.Prebid.Targeting != nil {
		if len(requestExt.Prebid.Targeting.PriceGranularity) > 0 {
			targData.targetFlag = true
			targData.lengthMax = requestExt.Prebid.Targeting.MaxLength
			targData.priceGranularity = requestExt.Prebid.Targeting.PriceGranularity
		}
	}

	adapterBids, adapterExtra := e.getAllBids(ctx, liveAdapters, cleanRequests, targData)

	// Build the response
	return e.buildBidResponse(liveAdapters, adapterBids, bidRequest, adapterExtra, targData, errs)
}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *exchange) getAllBids(ctx context.Context, liveAdapters []openrtb_ext.BidderName, cleanRequests map[openrtb_ext.BidderName]*openrtb.BidRequest, targData *targetData) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, map[openrtb_ext.BidderName]*seatResponseExtra) {
	// Set up pointers to the bid results
	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid, len(liveAdapters))
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, len(liveAdapters))
	chBids := make(chan *bidResponseWrapper, len(liveAdapters))
	for _, a := range liveAdapters {
		// Here we actually call the adapters and collect the bids.
		go func(aName openrtb_ext.BidderName) {
			// Passing in aName so a doesn't change out from under the go routine
			bidderTarg := &bidderTargeting{
				targetFlag: targData.targetFlag,
				lengthMax: targData.lengthMax,
				priceGranularity: targData.priceGranularity,
				bidder: aName,
			}
			brw := new(bidResponseWrapper)
			brw.bidder = aName
			start := time.Now()
			bids, err := e.adapterMap[aName].requestBid(ctx, cleanRequests[aName], bidderTarg)

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
	var err1 error = nil
	if targData.targetFlag {
		e.addWinningTargets(targData)
	}
	bidResponse.SeatBid = seatBids

	bidResponseExt := e.makeExtBidResponse(adapterBids, adapterExtra, bidRequest.Test, errList)
	ext, err := json.Marshal(bidResponseExt)
	bidResponse.Ext = ext
	if err1 != nil {
		err = err1
	}
	return bidResponse, err
}

func (e *exchange) addWinningTargets(targData *targetData) {
	for id, bid := range targData.winningBids {
		bidExt := new(openrtb_ext.ExtBid)
		err1 := json.Unmarshal(bid.Ext, bidExt)
		if err1 == nil && bidExt.Prebid.Targeting != nil {
			hbPbBidderKey := string(hbpbConstantKey+"_"+targData.winningBidders[id])
			hbBidderBidderKey := string(hbBidderConstantKey+"_"+targData.winningBidders[id])
			hbSizeBidderKey := string(hbSizeConstantKey+"_"+targData.winningBidders[id])
			hbDealIdBidderKey := string(hbDealIdConstantKey+"_"+targData.winningBidders[id])
			hbCacheIdBidderKey := string(hbCacheIdConstantKey+"_"+targData.winningBidders[id])
			if targData.lengthMax != 0 {
				hbPbBidderKey = hbPbBidderKey[:min(len(hbPbBidderKey), int(targData.lengthMax))]
				hbBidderBidderKey = hbBidderBidderKey[:min(len(hbBidderBidderKey), int(targData.lengthMax))]
				hbSizeBidderKey = hbSizeBidderKey[:min(len(hbSizeBidderKey), int(targData.lengthMax))]
				hbCacheIdBidderKey = hbCacheIdBidderKey[:min(len(hbSizeBidderKey), int(targData.lengthMax))]
				hbDealIdBidderKey = hbDealIdBidderKey[:min(len(hbSizeBidderKey), int(targData.lengthMax))]
			}

			bidExt.Prebid.Targeting[hbpbConstantKey] = bidExt.Prebid.Targeting[hbPbBidderKey]
			bidExt.Prebid.Targeting[hbBidderConstantKey] = bidExt.Prebid.Targeting[hbBidderBidderKey]
			if size, ok := bidExt.Prebid.Targeting[hbSizeBidderKey]; ok {
				bidExt.Prebid.Targeting[hbSizeConstantKey] = size
			}
			if cache, ok := bidExt.Prebid.Targeting[hbCacheIdBidderKey]; ok {
				bidExt.Prebid.Targeting[hbCacheIdConstantKey] = cache
			}
			if deal, ok := bidExt.Prebid.Targeting[hbDealIdBidderKey]; ok {
				bidExt.Prebid.Targeting[hbDealIdConstantKey] = deal
			}
			if targData.winningBidders[id] == "audienceNetwork" {
				bidExt.Prebid.Targeting[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodDemandSDK
			} else {
				bidExt.Prebid.Targeting[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodHTML
			}
			bid.Ext, err1 = json.Marshal(bidExt)
		}
	}

}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *exchange) makeExtBidResponse(adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, test int8, errList []error) *openrtb_ext.ExtBidResponse {
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
				bidResponseExt.Debug.HttpCalls[a] = b.httpCalls
			}
		}
		// Only make an entry for bidder errors if the bidder reported any.
		if len(adapterExtra[a].Errors) > 0 {
			bidResponseExt.Errors[a] = adapterExtra[a].Errors
		}
		if len(errList) > 0 {
			s := make([]string, len(errList))
			for i :=0; i<len(errList); i++ {
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
	bids := make([]openrtb.Bid, len(Bids))
	errList := make([]string, 0, 1)
	for i := 0; i < len(Bids); i++ {
		var err error
		bids[i] = *Bids[i].bid
		bidExt := new(openrtb_ext.ExtBid)
		bidExt.Bidder = bids[i].Ext
		bidPrebid := new(openrtb_ext.ExtBidPrebid)
		if targData.targetFlag {
			bidPrebid.Targeting = Bids[i].bidTargets
			cpm := bids[i].Price
			wbid, ok := targData.winningBids[bids[i].ImpID]
			if !ok || cpm > wbid.Price {
				targData.winningBidders[bids[i].ImpID] = adapter
				targData.winningBids[bids[i].ImpID] = &bids[i]
			}
		}
		bidExt.Prebid = bidPrebid
		bidPrebid.Type = Bids[i].bidType

		ext, err := json.Marshal(bidExt)
		if err != nil {
			errList = append(errList, fmt.Sprintf("Error writing SeatBid.Bid[%d].Ext: %s", i, err.Error()))
		}
		bids[i].Ext = ext
	}
	return bids, errList
}

// The following may move to /pbs/targeting with pbs/buckets going in there as well. But pbs/buckets in not yet in this branch
// This also duplicates code in pbs_light, which should be moved to /pbs/targeting. But that is beyond the current
// scope, and likely moot if the non-openrtb endpoint goes away.
const (
	hbpbConstantKey = "hb_pb"
	hbBidderConstantKey = "hb_bidder"
	hbSizeConstantKey = "hb_size"
	hbCreativeLoadMethodConstantKey = "hb_creative_loadtype"
	hbCreativeLoadMethodHTML = "html"
	hbCreativeLoadMethodDemandSDK = "demand_sdk"
	hbCacheIdConstantKey = "hb_cache_id"
	hbDealIdConstantKey = "hb_deal"
	)

