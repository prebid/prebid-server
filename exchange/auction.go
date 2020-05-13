package exchange

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"strings"

	uuid "github.com/gofrs/uuid"
	"github.com/golang/glog"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/prebid_cache_client"
)

type DebugLog struct {
	Enabled     bool
	CacheType   prebid_cache_client.PayloadType
	Data        DebugData
	TTL         int64
	CacheKey    string
	CacheString string
	Regexp      *regexp.Regexp
}

type DebugData struct {
	Request  string
	Headers  string
	Response string
}

func (d *DebugLog) BuildCacheString() {
	if d.Regexp != nil {
		d.Data.Request = fmt.Sprintf(d.Regexp.ReplaceAllString(d.Data.Request, ""))
		d.Data.Headers = fmt.Sprintf(d.Regexp.ReplaceAllString(d.Data.Headers, ""))
		d.Data.Response = fmt.Sprintf(d.Regexp.ReplaceAllString(d.Data.Response, ""))
	}

	d.Data.Request = fmt.Sprintf("<Request>%s</Request>", d.Data.Request)
	d.Data.Headers = fmt.Sprintf("<Headers>%s</Headers>", d.Data.Headers)
	d.Data.Response = fmt.Sprintf("<Response>%s</Response>", d.Data.Response)

	d.CacheString = fmt.Sprintf("%s<Log>%s%s%s</Log>", xml.Header, d.Data.Request, d.Data.Headers, d.Data.Response)
}

func newAuction(seatBids map[openrtb_ext.BidderName]*pbsOrtbSeatBid, numImps int) *auction {
	winningBids := make(map[string]*pbsOrtbBid, numImps)
	winningBidsByBidder := make(map[string]map[openrtb_ext.BidderName]*pbsOrtbBid, numImps)

	for bidderName, seatBid := range seatBids {
		if seatBid != nil {
			for _, bid := range seatBid.bids {
				cpm := bid.bid.Price
				wbid, ok := winningBids[bid.bid.ImpID]
				if !ok || cpm > wbid.bid.Price {
					winningBids[bid.bid.ImpID] = bid
				}
				if bidMap, ok := winningBidsByBidder[bid.bid.ImpID]; ok {
					bestSoFar, ok := bidMap[bidderName]
					if !ok || cpm > bestSoFar.bid.Price {
						bidMap[bidderName] = bid
					}
				} else {
					winningBidsByBidder[bid.bid.ImpID] = make(map[openrtb_ext.BidderName]*pbsOrtbBid)
					winningBidsByBidder[bid.bid.ImpID][bidderName] = bid
				}
			}
		}
	}

	return &auction{
		winningBids:         winningBids,
		winningBidsByBidder: winningBidsByBidder,
	}
}

func (a *auction) setRoundedPrices(priceGranularity openrtb_ext.PriceGranularity) {
	roundedPrices := make(map[*pbsOrtbBid]string, 5*len(a.winningBids))
	for _, topBidsPerImp := range a.winningBidsByBidder {
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

func (a *auction) doCache(ctx context.Context, cache prebid_cache_client.Client, targData *targetData, bidRequest *openrtb.BidRequest, ttlBuffer int64, defaultTTLs *config.DefaultTTLs, bidCategory map[string]string, debugLog *DebugLog) []error {
	var bids, vast, includeBidderKeys, includeWinners bool = targData.includeCacheBids, targData.includeCacheVast, targData.includeBidderKeys, targData.includeWinners
	if !((bids || vast) && (includeBidderKeys || includeWinners)) {
		return nil
	}
	var errs []error
	expectNumBids := valOrZero(bids, len(a.roundedPrices))
	expectNumVast := valOrZero(vast, len(a.roundedPrices))
	bidIndices := make(map[int]*openrtb.Bid, expectNumBids)
	vastIndices := make(map[int]*openrtb.Bid, expectNumVast)
	toCache := make([]prebid_cache_client.Cacheable, 0, expectNumBids+expectNumVast)
	expByImp := make(map[string]int64)
	competitiveExclusion := false
	var hbCacheID string
	if len(bidCategory) > 0 {
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
	for _, topBidsPerImp := range a.winningBidsByBidder {
		for _, topBidPerBidder := range topBidsPerImp {
			impID := topBidPerBidder.bid.ImpID
			isOverallWinner := a.winningBids[impID] == topBidPerBidder
			if !includeBidderKeys && !isOverallWinner {
				continue
			}
			var customCacheKey string
			var catDur string
			useCustomCacheKey := false
			if competitiveExclusion && isOverallWinner {
				// set custom cache key for winning bid when competitive exclusion applies
				catDur = bidCategory[topBidPerBidder.bid.ID]
				if len(catDur) > 0 {
					customCacheKey = fmt.Sprintf("%s_%s", catDur, hbCacheID)
					useCustomCacheKey = true
				}
			}
			if bids {
				if jsonBytes, err := json.Marshal(topBidPerBidder.bid); err == nil {
					if useCustomCacheKey {
						// not allowed if bids is true; log error and cache normally
						errs = append(errs, errors.New("cannot use custom cache key for non-vast bids"))
					}
					toCache = append(toCache, prebid_cache_client.Cacheable{
						Type:       prebid_cache_client.TypeJSON,
						Data:       jsonBytes,
						TTLSeconds: cacheTTL(expByImp[impID], topBidPerBidder.bid.Exp, defTTL(topBidPerBidder.bidType, defaultTTLs), ttlBuffer),
					})
					bidIndices[len(toCache)-1] = topBidPerBidder.bid
				} else {
					errs = append(errs, err)
				}
			}
			if vast && topBidPerBidder.bidType == openrtb_ext.BidTypeVideo {
				vast := makeVAST(topBidPerBidder.bid)
				if jsonBytes, err := json.Marshal(vast); err == nil {
					if useCustomCacheKey {
						toCache = append(toCache, prebid_cache_client.Cacheable{
							Type:       prebid_cache_client.TypeXML,
							Data:       jsonBytes,
							TTLSeconds: cacheTTL(expByImp[impID], topBidPerBidder.bid.Exp, defTTL(topBidPerBidder.bidType, defaultTTLs), ttlBuffer),
							Key:        customCacheKey,
						})
					} else {
						toCache = append(toCache, prebid_cache_client.Cacheable{
							Type:       prebid_cache_client.TypeXML,
							Data:       jsonBytes,
							TTLSeconds: cacheTTL(expByImp[impID], topBidPerBidder.bid.Exp, defTTL(topBidPerBidder.bidType, defaultTTLs), ttlBuffer),
						})
					}
					vastIndices[len(toCache)-1] = topBidPerBidder.bid
				} else {
					errs = append(errs, err)
				}
			}
		}
	}

	if debugLog != nil && debugLog.Enabled {
		debugLog.BuildCacheString()
		debugLog.CacheKey = hbCacheID
		if jsonBytes, err := json.Marshal(debugLog.CacheString); err == nil {
			toCache = append(toCache, prebid_cache_client.Cacheable{
				Type:       debugLog.CacheType,
				Data:       jsonBytes,
				TTLSeconds: debugLog.TTL,
				Key:        "log_" + debugLog.CacheKey,
			})
		}
	}

	ids, err := cache.PutJson(ctx, toCache)
	if err != nil {
		errs = append(errs, err...)
	}

	if bids {
		a.cacheIds = make(map[*openrtb.Bid]string, len(bidIndices))
		for index, bid := range bidIndices {
			if ids[index] != "" {
				a.cacheIds[bid] = ids[index]
			}
		}
	}
	if vast {
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
	// winningBids is a map from imp.id to the highest overall CPM bid in that imp.
	winningBids map[string]*pbsOrtbBid
	// winningBidsByBidder stores the highest bid on each imp by each bidder.
	winningBidsByBidder map[string]map[openrtb_ext.BidderName]*pbsOrtbBid
	// roundedPrices stores the price strings rounded for each bid according to the price granularity.
	roundedPrices map[*pbsOrtbBid]string
	// cacheIds stores the UUIDs from Prebid Cache for fetching the full bid JSON.
	cacheIds map[*openrtb.Bid]string
	// vastCacheIds stores UUIDS from Prebid cache for fetching the VAST markup to video bids.
	vastCacheIds map[*openrtb.Bid]string
}
