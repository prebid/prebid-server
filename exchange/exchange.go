package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"runtime/debug"
	"sort"
	"time"

	"github.com/prebid/prebid-server/stored_requests"

	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currencies"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbsmetrics"
	"github.com/prebid/prebid-server/prebid_cache_client"
)

// Exchange runs Auctions. Implementations must be threadsafe, and will be shared across many goroutines.
type Exchange interface {
	// HoldAuction executes an OpenRTB v2.5 Auction.
	HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, usersyncs IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher) (*openrtb.BidResponse, error)
}

// IdFetcher can find the user's ID for a specific Bidder.
type IdFetcher interface {
	// GetId returns the ID for the bidder. The boolean will be true if the ID exists, and false otherwise.
	GetId(bidder openrtb_ext.BidderName) (string, bool)
}

type exchange struct {
	adapterMap          map[openrtb_ext.BidderName]adaptedBidder
	me                  pbsmetrics.MetricsEngine
	cache               prebid_cache_client.Client
	cacheTime           time.Duration
	gDPR                gdpr.Permissions
	currencyConverter   *currencies.RateConverter
	UsersyncIfAmbiguous bool
	defaultTTLs         config.DefaultTTLs
}

// Container to pass out response ext data from the GetAllBids goroutines back into the main thread
type seatResponseExtra struct {
	ResponseTimeMillis int
	Errors             []openrtb_ext.ExtBidderError
}

type bidResponseWrapper struct {
	adapterBids  *pbsOrtbSeatBid
	adapterExtra *seatResponseExtra
	bidder       openrtb_ext.BidderName
}

func NewExchange(client *http.Client, cache prebid_cache_client.Client, cfg *config.Configuration, metricsEngine pbsmetrics.MetricsEngine, infos adapters.BidderInfos, gDPR gdpr.Permissions, currencyConverter *currencies.RateConverter) Exchange {
	e := new(exchange)

	e.adapterMap = newAdapterMap(client, cfg, infos)
	e.cache = cache
	e.cacheTime = time.Duration(cfg.CacheURL.ExpectedTimeMillis) * time.Millisecond
	e.me = metricsEngine
	e.gDPR = gDPR
	e.currencyConverter = currencyConverter
	e.UsersyncIfAmbiguous = cfg.GDPR.UsersyncIfAmbiguous
	e.defaultTTLs = cfg.CacheURL.DefaultTTLs
	return e
}

func (e *exchange) HoldAuction(ctx context.Context, bidRequest *openrtb.BidRequest, usersyncs IdFetcher, labels pbsmetrics.Labels, categoriesFetcher *stored_requests.CategoryFetcher) (*openrtb.BidResponse, error) {
	// Snapshot of resolved bid request for debug if test request
	var resolvedRequest json.RawMessage
	if bidRequest.Test == 1 {
		if r, err := json.Marshal(bidRequest); err != nil {
			glog.Errorf("Error marshalling bid request for debug: %v", err)
		} else {
			resolvedRequest = r
		}
	}

	for _, impInRequest := range bidRequest.Imp {
		var impLabels pbsmetrics.ImpLabels = pbsmetrics.ImpLabels{
			BannerImps: impInRequest.Banner != nil,
			VideoImps:  impInRequest.Video != nil,
			AudioImps:  impInRequest.Audio != nil,
			NativeImps: impInRequest.Native != nil,
		}
		e.me.RecordImps(impLabels)
	}

	// Slice of BidRequests, each a copy of the original cleaned to only contain bidder data for the named bidder
	blabels := make(map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels)
	cleanRequests, aliases, errs := cleanOpenRTBRequests(ctx, bidRequest, usersyncs, blabels, labels, e.gDPR, e.UsersyncIfAmbiguous)

	// List of bidders we have requests for.
	liveAdapters := make([]openrtb_ext.BidderName, len(cleanRequests))
	i := 0
	for a := range cleanRequests {
		liveAdapters[i] = a
		i++
	}
	// Randomize the list of adapters to make the auction more fair
	randomizeList(liveAdapters)
	// Process the request to check for targeting parameters.
	var targData *targetData
	shouldCacheBids := false
	shouldCacheVAST := false
	var bidAdjustmentFactors map[string]float64
	var requestExt openrtb_ext.ExtRequest
	if len(bidRequest.Ext) > 0 {
		err := json.Unmarshal(bidRequest.Ext, &requestExt)
		if err != nil {
			return nil, fmt.Errorf("Error decoding Request.ext : %s", err.Error())
		}
		bidAdjustmentFactors = requestExt.Prebid.BidAdjustmentFactors
		if requestExt.Prebid.Cache != nil {
			shouldCacheBids = requestExt.Prebid.Cache.Bids != nil
			shouldCacheVAST = requestExt.Prebid.Cache.VastXML != nil
		}

		if requestExt.Prebid.Targeting != nil {
			targData = &targetData{
				priceGranularity:  requestExt.Prebid.Targeting.PriceGranularity,
				includeWinners:    requestExt.Prebid.Targeting.IncludeWinners,
				includeBidderKeys: requestExt.Prebid.Targeting.IncludeBidderKeys,
			}
			if shouldCacheBids {
				targData.includeCacheBids = true
			}
			if shouldCacheVAST {
				targData.includeCacheVast = true
			}
			targData.cacheHost, targData.cachePath = e.cache.GetExtCacheData()
		}
	}

	// If we need to cache bids, then it will take some time to call prebid cache.
	// We should reduce the amount of time the bidders have, to compensate.
	auctionCtx, cancel := e.makeAuctionContext(ctx, shouldCacheBids)
	defer cancel()

	// Get currency rates conversions for the auction
	conversions := e.currencyConverter.Rates()

	adapterBids, adapterExtra, anyBidsReturned := e.getAllBids(auctionCtx, cleanRequests, aliases, bidAdjustmentFactors, blabels, conversions)

	if anyBidsReturned {

		var bidCategory map[string]string
		//If includebrandcategory is present in ext then CE feature is on.
		if requestExt.Prebid.Targeting != nil && requestExt.Prebid.Targeting.IncludeBrandCategory != nil {
			var err error
			bidCategory, adapterBids, err = applyCategoryMapping(ctx, requestExt, adapterBids, *categoriesFetcher, targData)
			if err != nil {
				return nil, fmt.Errorf("Error in category mapping : %s", err.Error())
			}
		}

		auc := newAuction(adapterBids, len(bidRequest.Imp))

		if targData != nil {
			auc.setRoundedPrices(targData.priceGranularity)
			cacheErrs := auc.doCache(ctx, e.cache, targData, bidRequest, 60, &e.defaultTTLs, bidCategory)
			if len(cacheErrs) > 0 {
				errs = append(errs, cacheErrs...)
			}
			targData.setTargeting(auc, bidRequest.App != nil, bidCategory)
		}
	}

	// Build the response
	return e.buildBidResponse(ctx, liveAdapters, adapterBids, bidRequest, resolvedRequest, adapterExtra, errs)
}

func (e *exchange) makeAuctionContext(ctx context.Context, needsCache bool) (auctionCtx context.Context, cancel context.CancelFunc) {
	auctionCtx = ctx
	cancel = func() {}
	if needsCache {
		if deadline, ok := ctx.Deadline(); ok {
			auctionCtx, cancel = context.WithDeadline(ctx, deadline.Add(-e.cacheTime))
		}
	}
	return
}

// This piece sends all the requests to the bidder adapters and gathers the results.
func (e *exchange) getAllBids(ctx context.Context, cleanRequests map[openrtb_ext.BidderName]*openrtb.BidRequest, aliases map[string]string, bidAdjustments map[string]float64, blabels map[openrtb_ext.BidderName]*pbsmetrics.AdapterLabels, conversions currencies.Conversions) (map[openrtb_ext.BidderName]*pbsOrtbSeatBid, map[openrtb_ext.BidderName]*seatResponseExtra, bool) {
	// Set up pointers to the bid results
	adapterBids := make(map[openrtb_ext.BidderName]*pbsOrtbSeatBid, len(cleanRequests))
	adapterExtra := make(map[openrtb_ext.BidderName]*seatResponseExtra, len(cleanRequests))
	chBids := make(chan *bidResponseWrapper, len(cleanRequests))
	bidsFound := false

	for bidderName, req := range cleanRequests {
		// Here we actually call the adapters and collect the bids.
		coreBidder := resolveBidder(string(bidderName), aliases)
		bidderRunner := e.recoverSafely(func(aName openrtb_ext.BidderName, coreBidder openrtb_ext.BidderName, request *openrtb.BidRequest, bidlabels *pbsmetrics.AdapterLabels, conversions currencies.Conversions) {
			// Passing in aName so a doesn't change out from under the go routine
			if bidlabels.Adapter == "" {
				glog.Errorf("Exchange: bidlables for %s (%s) missing adapter string", aName, coreBidder)
				bidlabels.Adapter = coreBidder
			}
			brw := new(bidResponseWrapper)
			brw.bidder = aName
			// Defer basic metrics to insure we capture them after all the values have been set
			defer func() {
				e.me.RecordAdapterRequest(*bidlabels)
			}()
			start := time.Now()

			adjustmentFactor := 1.0
			if givenAdjustment, ok := bidAdjustments[string(aName)]; ok {
				adjustmentFactor = givenAdjustment
			}
			var reqInfo adapters.ExtraRequestInfo
			reqInfo.PbsEntryPoint = bidlabels.RType
			bids, err := e.adapterMap[coreBidder].requestBid(ctx, request, aName, adjustmentFactor, conversions, &reqInfo)

			// Add in time reporting
			elapsed := time.Since(start)
			brw.adapterBids = bids
			// Structure to record extra tracking data generated during bidding
			ae := new(seatResponseExtra)
			ae.ResponseTimeMillis = int(elapsed / time.Millisecond)
			// Timing statistics
			e.me.RecordAdapterTime(*bidlabels, time.Since(start))
			serr := errsToBidderErrors(err)
			bidlabels.AdapterBids = bidsToMetric(brw.adapterBids)
			bidlabels.AdapterErrors = errorsToMetric(err)
			// Append any bid validation errors to the error list
			ae.Errors = serr
			brw.adapterExtra = ae
			if bids != nil {
				for _, bid := range bids.bids {
					var cpm = float64(bid.bid.Price * 1000)
					e.me.RecordAdapterPrice(*bidlabels, cpm)
					e.me.RecordAdapterBidReceived(*bidlabels, bid.bidType, bid.bid.AdM != "")
				}
			}
			chBids <- brw
		}, chBids)
		go bidderRunner(bidderName, coreBidder, req, blabels[coreBidder], conversions)
	}
	// Wait for the bidders to do their thing
	for i := 0; i < len(cleanRequests); i++ {
		brw := <-chBids
		adapterBids[brw.bidder] = brw.adapterBids
		adapterExtra[brw.bidder] = brw.adapterExtra

		if !bidsFound && adapterBids[brw.bidder] != nil && len(adapterBids[brw.bidder].bids) > 0 {
			bidsFound = true
		}
	}

	return adapterBids, adapterExtra, bidsFound
}

func (e *exchange) recoverSafely(inner func(openrtb_ext.BidderName, openrtb_ext.BidderName, *openrtb.BidRequest, *pbsmetrics.AdapterLabels, currencies.Conversions), chBids chan *bidResponseWrapper) func(openrtb_ext.BidderName, openrtb_ext.BidderName, *openrtb.BidRequest, *pbsmetrics.AdapterLabels, currencies.Conversions) {
	return func(aName openrtb_ext.BidderName, coreBidder openrtb_ext.BidderName, request *openrtb.BidRequest, bidlabels *pbsmetrics.AdapterLabels, conversions currencies.Conversions) {
		defer func() {
			if r := recover(); r != nil {
				glog.Errorf("OpenRTB auction recovered panic from Bidder %s: %v. Stack trace is: %v", coreBidder, r, string(debug.Stack()))
				e.me.RecordAdapterPanic(*bidlabels)
				// Let the master request know that there is no data here
				brw := new(bidResponseWrapper)
				brw.adapterExtra = new(seatResponseExtra)
				chBids <- brw
			}
		}()
		inner(aName, coreBidder, request, bidlabels, conversions)
	}
}

func bidsToMetric(bids *pbsOrtbSeatBid) pbsmetrics.AdapterBid {
	if bids == nil || len(bids.bids) == 0 {
		return pbsmetrics.AdapterBidNone
	}
	return pbsmetrics.AdapterBidPresent
}

func errorsToMetric(errs []error) map[pbsmetrics.AdapterError]struct{} {
	if len(errs) == 0 {
		return nil
	}
	ret := make(map[pbsmetrics.AdapterError]struct{}, len(errs))
	var s struct{}
	for _, err := range errs {
		switch errortypes.DecodeError(err) {
		case errortypes.TimeoutCode:
			ret[pbsmetrics.AdapterErrorTimeout] = s
		case errortypes.BadInputCode:
			ret[pbsmetrics.AdapterErrorBadInput] = s
		case errortypes.BadServerResponseCode:
			ret[pbsmetrics.AdapterErrorBadServerResponse] = s
		case errortypes.FailedToRequestBidsCode:
			ret[pbsmetrics.AdapterErrorFailedToRequestBids] = s
		default:
			ret[pbsmetrics.AdapterErrorUnknown] = s
		}
	}
	return ret
}

func errsToBidderErrors(errs []error) []openrtb_ext.ExtBidderError {
	serr := make([]openrtb_ext.ExtBidderError, len(errs))
	for i := 0; i < len(errs); i++ {
		serr[i].Code = errortypes.DecodeError(errs[i])
		serr[i].Message = errs[i].Error()
	}
	return serr
}

// This piece takes all the bids supplied by the adapters and crafts an openRTB response to send back to the requester
func (e *exchange) buildBidResponse(ctx context.Context, liveAdapters []openrtb_ext.BidderName, adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, bidRequest *openrtb.BidRequest, resolvedRequest json.RawMessage, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, errList []error) (*openrtb.BidResponse, error) {
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
		//while processing every single bib, do we need to handle categories here?
		if adapterBids[a] != nil && len(adapterBids[a].bids) > 0 {
			sb := e.makeSeatBid(adapterBids[a], a, adapterExtra)
			seatBids = append(seatBids, *sb)
		}
	}

	bidResponse.SeatBid = seatBids

	bidResponseExt := e.makeExtBidResponse(adapterBids, adapterExtra, bidRequest, resolvedRequest, errList)
	buffer := &bytes.Buffer{}
	enc := json.NewEncoder(buffer)
	enc.SetEscapeHTML(false)
	err := enc.Encode(bidResponseExt)
	bidResponse.Ext = buffer.Bytes()

	return bidResponse, err
}

func applyCategoryMapping(ctx context.Context, requestExt openrtb_ext.ExtRequest, seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, categoriesFetcher stored_requests.CategoryFetcher, targData *targetData) (map[string]string, map[openrtb_ext.BidderName]*pbsOrtbSeatBid, error) {
	res := make(map[string]string)

	type bidDedupe struct {
		bidderName openrtb_ext.BidderName
		bidIndex   int
		bidID      string
	}

	dedupe := make(map[string]bidDedupe)

	brandCatExt := requestExt.Prebid.Targeting.IncludeBrandCategory

	//If ext.prebid.targeting.includebrandcategory is present in ext then competitive exclusion feature is on.
	var includeBrandCategory = brandCatExt != nil //if not present - category will no be appended

	var primaryAdServer string
	var publisher string
	var err error

	if includeBrandCategory && brandCatExt.WithCategory {
		//if ext.prebid.targeting.includebrandcategory present but primaryadserver/publisher not present then error out the request right away.
		primaryAdServer, err = getPrimaryAdServer(brandCatExt.PrimaryAdServer) //1-Freewheel 2-DFP
		if err != nil {
			return res, seatBids, err
		}
		publisher = brandCatExt.Publisher
	}

	seatBidsToRemove := make([]openrtb_ext.BidderName, 0)

	for bidderName, seatBid := range seatBids {
		bidsToRemove := make([]int, 0)
		for bidInd := range seatBid.bids {
			bid := seatBid.bids[bidInd]
			var duration int
			var category string
			var pb string

			if bid.bidVideo != nil {
				duration = bid.bidVideo.Duration
				category = bid.bidVideo.PrimaryCategory
			}
			if brandCatExt.WithCategory && category == "" {
				bidIabCat := bid.bid.Cat
				if len(bidIabCat) != 1 {
					//TODO: add metrics
					//on receiving bids from adapters if no unique IAB category is returned  or if no ad server category is returned discard the bid
					bidsToRemove = append(bidsToRemove, bidInd)
					continue
				}
				//if unique IAB category is present then translate it to the adserver category based on mapping file
				category, err = categoriesFetcher.FetchCategories(ctx, primaryAdServer, publisher, bidIabCat[0])
				if err != nil || category == "" {
					//TODO: add metrics
					//if mapping required but no mapping file is found then discard the bid
					bidsToRemove = append(bidsToRemove, bidInd)
					continue
				}

			}

			// TODO: consider should we remove bids with zero duration here?

			pb, _ = GetCpmStringValue(bid.bid.Price, targData.priceGranularity)

			newDur := duration
			if len(requestExt.Prebid.Targeting.DurationRangeSec) > 0 {
				durationRange := requestExt.Prebid.Targeting.DurationRangeSec
				sort.Ints(durationRange)
				//if the bid is above the range of the listed durations (and outside the buffer), reject the bid
				if duration > durationRange[len(durationRange)-1] {
					bidsToRemove = append(bidsToRemove, bidInd)
					continue
				}
				for _, dur := range durationRange {
					if duration <= dur {
						newDur = dur
						break
					}
				}
			}

			var categoryDuration string
			if brandCatExt.WithCategory {
				categoryDuration = fmt.Sprintf("%s_%s_%ds", pb, category, newDur)
			} else {
				categoryDuration = fmt.Sprintf("%s_%ds", pb, newDur)
			}

			if dupe, ok := dedupe[categoryDuration]; ok {
				// 50% chance for either bid with duplicate categoryDuration values to be kept
				if rand.Intn(100) < 50 {
					if dupe.bidderName == bidderName {
						// An older bid from the current bidder
						bidsToRemove = append(bidsToRemove, dupe.bidIndex)
					} else {
						// An older bid from a different seatBid we've already finished with
						oldSeatBid := (seatBids)[dupe.bidderName]
						if len(oldSeatBid.bids) == 1 {
							seatBidsToRemove = append(seatBidsToRemove, bidderName)
						} else {
							oldSeatBid.bids = append(oldSeatBid.bids[:dupe.bidIndex], oldSeatBid.bids[dupe.bidIndex+1:]...)
						}
					}
					delete(res, dupe.bidID)
				} else {
					// Remove this bid
					bidsToRemove = append(bidsToRemove, bidInd)
					continue
				}
			}
			res[bid.bid.ID] = categoryDuration
			dedupe[categoryDuration] = bidDedupe{bidderName: bidderName, bidIndex: bidInd, bidID: bid.bid.ID}
		}

		if len(bidsToRemove) > 0 {
			sort.Ints(bidsToRemove)
			if len(bidsToRemove) == len(seatBid.bids) {
				//if all bids are invalid - remove entire seat bid
				seatBidsToRemove = append(seatBidsToRemove, bidderName)
			} else {
				bids := seatBid.bids
				for i := len(bidsToRemove) - 1; i >= 0; i-- {
					remInd := bidsToRemove[i]
					bids = append(bids[:remInd], bids[remInd+1:]...)
				}
				seatBid.bids = bids
			}
		}

	}
	if len(seatBidsToRemove) > 0 {
		if len(seatBidsToRemove) == len(seatBids) {
			//delete all seat bids
			seatBids = nil
		} else {
			for _, seatBidInd := range seatBidsToRemove {
				delete(seatBids, seatBidInd)
			}

		}
	}

	return res, seatBids, nil
}

func getPrimaryAdServer(adServerId int) (string, error) {
	switch adServerId {
	case 1:
		return "freewheel", nil
	case 2:
		return "dfp", nil
	default:
		return "", fmt.Errorf("Primary ad server %d not recognized", adServerId)
	}
}

// Extract all the data from the SeatBids and build the ExtBidResponse
func (e *exchange) makeExtBidResponse(adapterBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra, req *openrtb.BidRequest, resolvedRequest json.RawMessage, errList []error) *openrtb_ext.ExtBidResponse {
	bidResponseExt := &openrtb_ext.ExtBidResponse{
		Errors:               make(map[openrtb_ext.BidderName][]openrtb_ext.ExtBidderError, len(adapterBids)),
		ResponseTimeMillis:   make(map[openrtb_ext.BidderName]int, len(adapterBids)),
		RequestTimeoutMillis: req.TMax,
	}
	if req.Test == 1 {
		bidResponseExt.Debug = &openrtb_ext.ExtResponseDebug{
			HttpCalls: make(map[openrtb_ext.BidderName][]*openrtb_ext.ExtHttpCall),
		}
		if err := json.Unmarshal(resolvedRequest, &bidResponseExt.Debug.ResolvedRequest); err != nil {
			glog.Errorf("Error unmarshalling bid request snapshot: %v", err)
		}
	}

	for a, b := range adapterBids {
		if b != nil && req.Test == 1 {
			// Fill debug info
			bidResponseExt.Debug.HttpCalls[a] = b.httpCalls
		}
		// Only make an entry for bidder errors if the bidder reported any.
		if len(adapterExtra[a].Errors) > 0 {
			bidResponseExt.Errors[a] = adapterExtra[a].Errors
		}
		if len(errList) > 0 {
			bidResponseExt.Errors[openrtb_ext.PrebidExtKey] = errsToBidderErrors(errList)
		}
		bidResponseExt.ResponseTimeMillis[a] = adapterExtra[a].ResponseTimeMillis
		// Defering the filling of bidResponseExt.Usersync[a] until later

	}
	return bidResponseExt
}

// Return an openrtb seatBid for a bidder
// BuildBidResponse is responsible for ensuring nil bid seatbids are not included
func (e *exchange) makeSeatBid(adapterBid *pbsOrtbSeatBid, adapter openrtb_ext.BidderName, adapterExtra map[openrtb_ext.BidderName]*seatResponseExtra) *openrtb.SeatBid {
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
			extError := openrtb_ext.ExtBidderError{
				Code:    errortypes.DecodeError(err),
				Message: fmt.Sprintf("Error writing SeatBid.Ext: %s", err.Error()),
			}
			adapterExtra[adapter].Errors = append(adapterExtra[adapter].Errors, extError)
		}
		seatBid.Ext = ext
	}

	var errList []error
	seatBid.Bid, errList = e.makeBid(adapterBid.bids, adapter)
	if len(errList) > 0 {
		adapterExtra[adapter].Errors = append(adapterExtra[adapter].Errors, errsToBidderErrors(errList)...)
	}

	return seatBid
}

// Create the Bid array inside of SeatBid
func (e *exchange) makeBid(Bids []*pbsOrtbBid, adapter openrtb_ext.BidderName) ([]openrtb.Bid, []error) {
	bids := make([]openrtb.Bid, 0, len(Bids))
	errList := make([]error, 0, 1)
	for _, thisBid := range Bids {
		bidExt := &openrtb_ext.ExtBid{
			Bidder: thisBid.bid.Ext,
			Prebid: &openrtb_ext.ExtBidPrebid{
				Targeting: thisBid.bidTargets,
				Type:      thisBid.bidType,
				Video:     thisBid.bidVideo,
			},
		}

		ext, err := json.Marshal(bidExt)
		if err != nil {
			errList = append(errList, err)
		} else {
			bids = append(bids, *thisBid.bid)
			bids[len(bids)-1].Ext = ext
		}
	}
	return bids, errList
}
