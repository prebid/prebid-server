package exchange

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"context"
	"time"
	"net/http"
	"encoding/json"
	"fmt"
	"github.com/prebid/prebid-server/pbs"
	"strconv"
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
	cpm float64
	bid *openrtb.Bid
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
	cleanRequests, errs := openrtb_ext.CleanOpenRTBRequests(bidRequest, e.adapters)
	// List of bidders we have requests for.
	liveAdapters := make([]openrtb_ext.BidderName, len(cleanRequests))
	i := 0
	for a, _ := range cleanRequests {
		liveAdapters[i] = a
		i++
	}
	// Randomize the list of adapters to make the auction more fair
	openrtb_ext.RandomizeList(liveAdapters)

	adapterBids, adapterExtra := e.GetAllBids(ctx, liveAdapters, cleanRequests)

	// Build the response
	return e.BuildBidResponse(liveAdapters, adapterBids, bidRequest, adapterExtra, errs)
}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *exchange) GetAllBids(ctx context.Context, liveAdapters []openrtb_ext.BidderName, cleanRequests map[openrtb_ext.BidderName]*openrtb.BidRequest) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, map[openrtb_ext.BidderName]*seatResponseExtra) {
	// Set up pointers to the bid results
	adapterBids := map[openrtb_ext.BidderName]*pbsOrtbSeatBid{}
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra)
	chBids := make(chan *bidResponseWrapper, len(liveAdapters))
	for _, a := range liveAdapters {
		// Here we actually call the adapters and collect the bids.
		go func(aName openrtb_ext.BidderName) {
			// Passing in aName so a doesn't change out from under the go routine
			brw := new(bidResponseWrapper)
			brw.bidder = aName
			start := time.Now()
			bids, err := e.adapterMap[aName].requestBid(ctx, cleanRequests[aName])

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
func (e *exchange) BuildBidResponse(liveAdapters []openrtb_ext.BidderName, adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, bidRequest *openrtb.BidRequest, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, errList []error) (*openrtb.BidResponse, error) {
	bidResponse := new(openrtb.BidResponse)

	bidResponse.ID = bidRequest.ID
	if len(liveAdapters) == 0 {
		// signal "Invalid Request" if no valid bidders.
		bidResponse.NBR = openrtb.NoBidReasonCode.Ptr(openrtb.NoBidReasonCodeInvalidRequest)
	}

	// Process the request to check for targeting parameters.
	targData := &targetData{
		targetFlag:false,
		lengthMax:0,
		priceGranularity:openrtb_ext.PriceGranularityMedium,
		cpm:0.0,
		bid:nil,
		bidder:openrtb_ext.BidderName(""),
	}
	requestExt := new(openrtb_ext.ExtRequest)
	err := json.Unmarshal(bidRequest.Ext, requestExt)
	if err != nil {
		errList = append(errList, fmt.Errorf("Error decoding Request.ext : %s", err.Error()))
	}
	if requestExt.Prebid.Targeting != nil {
		if len(requestExt.Prebid.Targeting.PriceGranularity) > 0 {
			targData.targetFlag = true
			targData.lengthMax = requestExt.Prebid.Targeting.MaxLength
			targData.priceGranularity = requestExt.Prebid.Targeting.PriceGranularity
		}
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
			sb := e.MakeSeatBid(adapterBids[a], a, adapterExtra, targData)
			seatBids = append(seatBids, *sb)
		}
	}
	var err1 error = nil
	if targData.targetFlag && targData.bid != nil {
		bidExt := new(openrtb_ext.ExtBid)
		err1 = json.Unmarshal(targData.bid.Ext, bidExt)
		if err1 == nil {
			bidExt.Prebid.Targeting[hbpbConstantKey] = bidExt.Prebid.Targeting[string(hbpbConstantKey+"-"+targData.bidder)]
			bidExt.Prebid.Targeting[hbBidderConstantKey] = bidExt.Prebid.Targeting[string(hbBidderConstantKey+"-"+targData.bidder)]
			bidExt.Prebid.Targeting[hbSizeConstantKey] = bidExt.Prebid.Targeting[string(hbSizeConstantKey+"-"+targData.bidder)]
			if targData.bidder == "audienceNetwork" {
				bidExt.Prebid.Targeting[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodDemandSDK
			} else {
				bidExt.Prebid.Targeting[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodHTML
			}

		}
	}
	bidResponse.SeatBid = seatBids

	bidResponseExt := e.MakeExtBidResponse(adapterBids, adapterExtra, bidRequest.Test, errList)
	ext, err := json.Marshal(bidResponseExt)
	bidResponse.Ext = ext
	if err1 != nil {
		err = err1
	}
	return bidResponse, err
}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *exchange) MakeExtBidResponse(adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, test int8, errList []error) *openrtb_ext.ExtBidResponse {
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
func (e *exchange) MakeSeatBid(adapterBid *pbsOrtbSeatBid, adapter openrtb_ext.BidderName, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, targData *targetData) *openrtb.SeatBid {
	seatBid := new(openrtb.SeatBid)
	seatBid.Seat = adapter.String()
	// Prebid cannot support roadblocking
	seatBid.Group = 0

	sbExt := ExtSeatBid{
		Bidder: adapterBid.ext,
	}

	ext, err := json.Marshal(sbExt)
	if err != nil {
		adapterExtra[adapter].Errors = append(adapterExtra[adapter].Errors, fmt.Sprintf("Error writing SeatBid.Ext: %s", err.Error()))
	}
	seatBid.Ext = ext
	var errList []string
	seatBid.Bid, errList = e.MakeBid(adapterBid.bids, targData, adapter)
	if len(errList) > 0 {
		adapterExtra[adapter].Errors = append(adapterExtra[adapter].Errors, errList...)
	}

	return seatBid
}

// Create the Bid array inside of SeatBid
func (e *exchange) MakeBid(Bids []*pbsOrtbBid, targData *targetData, adapter openrtb_ext.BidderName) ([]openrtb.Bid, []string) {
	bids := make([]openrtb.Bid, len(Bids))
	errList := make([]string, 0, 1)
	for i := 0; i < len(Bids); i++ {
		bids[i] = *Bids[i].bid
		bidExt := new(openrtb_ext.ExtBid)
		bidExt.Bidder = bids[i].Ext
		bidPrebid := new(openrtb_ext.ExtBidPrebid)
		//bidPrebid.Cache = Bids[i].Cache
		bidPrebid.Type = Bids[i].bidType
		if targData.targetFlag {
			cpm := bids[i].Price
			width := bids[i].W
			height := bids[i].H
			bidPrebid.Targeting = e.MakePrebidTargets(cpm, width, height, bidPrebid.Cache.Key, targData, adapter)
			if cpm > targData.cpm {
				targData.cpm = cpm
				targData.bidder = adapter
				targData.bid = &bids[i]
			}
		}
		bidExt.Prebid = bidPrebid

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
	)

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (e *exchange) MakePrebidTargets(cpm float64, width uint64, height uint64, cache string, targData *targetData, adapter openrtb_ext.BidderName) map[string]string {
	priceBucketStringMap := pbs.GetPriceBucketString(cpm)
	roundedCpm := priceBucketStringMap[string(targData.priceGranularity)]

	hbSize := ""
	if width != 0 && height != 0 {
		w := strconv.FormatUint(width, 10)
		h := strconv.FormatUint(height, 10)
		hbSize = w + "x" + h
	}

	hbPbBidderKey := string(hbpbConstantKey + "_" + adapter)
	hbBidderBidderKey := string(hbBidderConstantKey + "_" + adapter)
	hbSizeBidderKey := string(hbSizeConstantKey + "_" + adapter)
	if targData.lengthMax != 0 {
		hbPbBidderKey = hbPbBidderKey[:min(len(hbPbBidderKey), int(targData.lengthMax))]
		hbBidderBidderKey = hbBidderBidderKey[:min(len(hbBidderBidderKey), int(targData.lengthMax))]
		hbSizeBidderKey = hbSizeBidderKey[:min(len(hbSizeBidderKey), int(targData.lengthMax))]
	}
	pbs_kvs := map[string]string{
		hbPbBidderKey:      roundedCpm,
		hbBidderBidderKey:  string(adapter),
	}
	if hbSize != "" {
		pbs_kvs[hbSizeBidderKey] = hbSize
	}

	return pbs_kvs
}