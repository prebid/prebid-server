package exchange

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	uuid "github.com/gofrs/uuid"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"
)

func newAuction(seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, numImps int) *auction {
	impsToTopBids := make(map[string]*pbsOrtbBid, numImps)
	impsToBiddersTopBids := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid, numImps)

	for bidderName, seatBid := range seatBids {
		if seatBid != nil {
			for _, thisBid := range seatBid.bids {
				// If we still don't have the highest bid for this imp in impsToTopBids, or the one we have is worse than our current bid, update
				_, ok := impsToTopBids[thisBid.bid.ImpID]
				if !ok || thisBid.bid.Price > impsToTopBids[thisBid.bid.ImpID].bid.Price {
					impsToTopBids[thisBid.bid.ImpID] = thisBid
				}
				// Do we have bids from this imp registered in impsToBiddersTopBids yet?
				if _, ok := impsToBiddersTopBids[thisBid.bid.ImpID]; ok {
					// There are bids from this imp but, are there bids comming from this bidder name?
					_, ok := impsToBiddersTopBids[thisBid.bid.ImpID][bidderName]
					if !ok || thisBid.bid.Price > impsToBiddersTopBids[thisBid.bid.ImpID][bidderName].bid.Price {
						// We didn't find a bid from this bidder or the one we found is lower than our current bid. Update
						impsToBiddersTopBids[thisBid.bid.ImpID][bidderName] = thisBid
					}
				} else {
					// No we don't have bids from this imp nor bidder in impsToBiddersTopBids, create one with current bid's data
					impsToBiddersTopBids[thisBid.bid.ImpID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
					impsToBiddersTopBids[thisBid.bid.ImpID][bidderName] = thisBid
				}
			}
		}
	}

	return &auction{
		impsToTopBids:        impsToTopBids,
		impsToBiddersTopBids: impsToBiddersTopBids,
	}
}

func (a *auction) setRoundedPrices(priceGranularity openrtb_ext.PriceGranularity) {
	roundedPrices := make(map[*pbsOrtbBid]string, 5*len(a.impsToTopBids))
	for _, topBidsPerImp := range a.impsToBiddersTopBids {
		for _, topBidPerBidder := range topBidsPerImp {
			roundedPrice, err := GetCpmStringValue(topBidPerBidder.bid.Price, priceGranularity)
			if err != nil {
				glog.Errorf(`Error rounding price according to granularity. This shouldn't happen unless /openrtb2 input validation is buggy. Granularity was "%v".`, priceGranularity)
			}
			roundedPrices[topBidPerBidder] = roundedPrice
		}
	}
	a.roundedPrices = roundedPrices
}

func (a *auction) doCache(
	ctx context.Context,
	cache prebid_cache_client.Client,
	targData *targetData,
	bidRequest *openrtb.BidRequest,
	ttlBuffer int64,
	defaultTTLs *config.DefaultTTLs,
	bidCategory map[string]string,
) []error {

	if (!targData.includeCacheBids && !targData.includeCacheVast) || (!targData.includeBidderKeys && !targData.includeWinners) {
		return nil
	}

	var errs []error
	var cacheErr error
	var expectNumBids, expectNumVast, newlyCached int
	var bidIndices, vastIndices map[int]*openrtb.Bid
	var toCache []prebid_cache_client.Cacheable
	var expByImp map[string]int64
	var hbCacheID string
	var competitiveExclusion, isNonVast, isVast bool

	expectNumBids = valOrZero(targData.includeCacheBids, len(a.roundedPrices))
	expectNumVast = valOrZero(targData.includeCacheVast, len(a.roundedPrices))
	bidIndices = make(map[int]*openrtb.Bid, expectNumBids)
	vastIndices = make(map[int]*openrtb.Bid, expectNumVast)
	toCache = make([]prebid_cache_client.Cacheable, 0, expectNumBids+expectNumVast)
	expByImp = make(map[string]int64)
	competitiveExclusion = false

	if len(bidCategory) > 0 && targData.includeCacheVast {
		// assert:  category of winning bids never duplicated
		if rawUuid, err := uuid.NewV4(); err == nil {
			hbCacheID = rawUuid.String()
			competitiveExclusion = true
		} else {
			errs = append(errs, errors.New("failed to create custom cache key"))
		}
	}

	// Grab the imp TTLs
	for _, imp := range bidRequest.Imp {
		expByImp[imp.ID] = imp.Exp
	}

	// if targData.includeBidderKeys is true, we should cache all bids, both winning and losing. In other words, we'll cache
	// banners and/or videos found in impsToBiddersTopBids map[string]map[openrtb_ext.BidderName]*pbsOrtbBid
	if targData.includeBidderKeys {
		for impID := range a.impsToBiddersTopBids {
			for _, bidToCache := range a.impsToBiddersTopBids[impID] {
				isNonVast, isVast, newlyCached, cacheErr = a.cacheBid(bidToCache, targData.includeCacheBids, targData.includeCacheVast, competitiveExclusion, &toCache, expByImp, defaultTTLs, ttlBuffer, bidCategory, hbCacheID)
				if isNonVast {
					bidIndices[len(toCache)-newlyCached] = bidToCache.bid
					newlyCached--
				}
				if isVast {
					vastIndices[len(toCache)-newlyCached] = bidToCache.bid
				}
				if cacheErr != nil {
					errs = append(errs, cacheErr)
				}
			}
		}
	} else {
		// targData.includeBidderKeys is false, therefore, targData.includeWinners is true and we should cache only winning bids
		// which are found in a.impsToTopBids
		for _, bidToCache := range a.impsToTopBids {
			isNonVast, isVast, newlyCached, cacheErr = a.cacheBid(bidToCache, targData.includeCacheBids, targData.includeCacheVast, competitiveExclusion, &toCache, expByImp, defaultTTLs, ttlBuffer, bidCategory, hbCacheID)
			if isNonVast {
				bidIndices[len(toCache)-newlyCached] = bidToCache.bid
				newlyCached--
			}
			if isVast {
				vastIndices[len(toCache)-newlyCached] = bidToCache.bid
			}
			if cacheErr != nil {
				errs = append(errs, cacheErr)
			}
		}
	}

	ids, err := cache.PutJson(ctx, toCache)
	if err != nil {
		errs = append(errs, err...)
	}

	if targData.includeCacheBids {
		a.cacheIds = make(map[*openrtb.Bid]string, len(bidIndices))
		for index, bid := range bidIndices {
			if ids[index] != "" {
				a.cacheIds[bid] = ids[index]
			}
		}
	}
	if targData.includeCacheVast {
		a.vastCacheIds = make(map[*openrtb.Bid]string, len(vastIndices))
		for index, bid := range vastIndices {
			if ids[index] != "" {
				if competitiveExclusion && strings.HasSuffix(ids[index], hbCacheID) {
					// omit the pb_cat_dur_ portion of cache ID
					a.vastCacheIds[bid] = hbCacheID
				} else {
					a.vastCacheIds[bid] = ids[index]
				}
			}
		}
	}
	return errs
}

func (a *auction) cacheBid(bidToCache *pbsOrtbBid, incBannerBids bool, incVastBids bool, hasCustomCacheKey bool, toCache *[]prebid_cache_client.Cacheable, expByImp map[string]int64, defaultTTLs *config.DefaultTTLs, ttlBuffer int64, bidCategory map[string]string, hbCacheID string) (bool, bool, int, error) {
	var chachedBid, chachedVast bool = false, false
	var cachedSoFar int = len(*toCache)
	var anError error = nil

	if incBannerBids { //banner
		if jsonBytes, err := json.Marshal(bidToCache.bid); err == nil {
			if hasCustomCacheKey {
				anError = errors.New("cannot use custom cache key for non-vast bids")
			}
			*toCache = append(*toCache, prebid_cache_client.Cacheable{
				Type:       prebid_cache_client.TypeJSON,
				Data:       jsonBytes,
				TTLSeconds: cacheTTL(expByImp[bidToCache.bid.ImpID], bidToCache.bid.Exp, defTTL(bidToCache.bidType, defaultTTLs), ttlBuffer),
			})
			chachedBid = true
		} else {
			anError = err
		}

	}
	if incVastBids && bidToCache.bidType == openrtb_ext.BidTypeVideo { //video
		if jsonBytes, err := json.Marshal(makeVAST(bidToCache.bid)); err == nil {
			_, isTopBid := a.impsToTopBids[bidToCache.bid.ImpID]
			if catDur, ok := bidCategory[a.impsToTopBids[bidToCache.bid.ImpID].bid.ID]; ok && isTopBid {
				*toCache = append(*toCache, prebid_cache_client.Cacheable{
					Type:       prebid_cache_client.TypeXML,
					Data:       jsonBytes,
					TTLSeconds: cacheTTL(expByImp[bidToCache.bid.ImpID], bidToCache.bid.Exp, defTTL(bidToCache.bidType, defaultTTLs), ttlBuffer),
					Key:        fmt.Sprintf("%s_%s", catDur, hbCacheID),
				})
			} else {
				*toCache = append(*toCache, prebid_cache_client.Cacheable{
					Type:       prebid_cache_client.TypeXML,
					Data:       jsonBytes,
					TTLSeconds: cacheTTL(expByImp[bidToCache.bid.ImpID], bidToCache.bid.Exp, defTTL(bidToCache.bidType, defaultTTLs), ttlBuffer),
				})
			}
			chachedVast = true
		} else {
			anError = err
		}
	}
	return chachedBid, chachedVast, len(*toCache) - cachedSoFar, anError
}

// makeVAST returns some VAST XML for the given bid. If AdM is defined,
// it takes precedence. Otherwise the Nurl will be wrapped in a redirect tag.
func makeVAST(bid *openrtb.Bid) string {
	if bid.AdM == "" {
		return `<VAST version="3.0"><Ad><Wrapper>` +
			`<AdSystem>prebid.org wrapper</AdSystem>` +
			`<VASTAdTagURI><![CDATA[` + bid.NURL + `]]></VASTAdTagURI>` +
			`<Impression></Impression><Creatives></Creatives>` +
			`</Wrapper></Ad></VAST>`
	}
	return bid.AdM
}

func valOrZero(useVal bool, val int) int {
	if useVal {
		return val
	}
	return 0
}

func maybeMake(shouldMake bool, capacity int) []prebid_cache_client.Cacheable {
	if shouldMake {
		return make([]prebid_cache_client.Cacheable, 0, capacity)
	}
	return nil
}

func cacheTTL(impTTL int64, bidTTL int64, defTTL int64, buffer int64) (ttl int64) {
	if impTTL <= 0 && bidTTL <= 0 {
		// Only use default if there is no imp nor bid TTL provided. We don't want the default
		// to cut short a requested longer TTL.
		return addBuffer(defTTL, buffer)
	}
	if impTTL <= 0 {
		// Use <= to handle the case of someone sending a negative ttl. We treat it as zero
		return addBuffer(bidTTL, buffer)
	}
	if bidTTL <= 0 {
		return addBuffer(impTTL, buffer)
	}
	if impTTL < bidTTL {
		return addBuffer(impTTL, buffer)
	}
	return addBuffer(bidTTL, buffer)
}

func addBuffer(base int64, buffer int64) int64 {
	if base <= 0 {
		return 0
	}
	return base + buffer
}

func defTTL(bidType openrtb_ext.BidType, defaultTTLs *config.DefaultTTLs) (ttl int64) {
	switch bidType {
	case openrtb_ext.BidTypeBanner:
		return int64(defaultTTLs.Banner)
	case openrtb_ext.BidTypeVideo:
		return int64(defaultTTLs.Video)
	case openrtb_ext.BidTypeNative:
		return int64(defaultTTLs.Native)
	case openrtb_ext.BidTypeAudio:
		return int64(defaultTTLs.Audio)
	}
	return 0
}

type auction struct {
	// We'll store the hightest bid comming from each Imp in the request
	impsToTopBids map[string]*pbsOrtbBid
	// We'll store the hightest bids comming from each Bidder of each Imp in the request
	impsToBiddersTopBids map[string]map[openrtb_ext.BidderName]*pbsOrtbBid
	// roundedPrices stores the price strings rounded for each bid according to the price granularity.
	roundedPrices map[*pbsOrtbBid]string
	// cacheIds stores the UUIDs from Prebid Cache for fetching the full bid JSON.
	cacheIds map[*openrtb.Bid]string
	// vastCacheIds stores UUIDS from Prebid cache for fetching the VAST markup to video bids.
	vastCacheIds map[*openrtb.Bid]string
}
