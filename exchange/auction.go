package exchange

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/prebid_cache_client"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

const (
	DebugOverrideHeader string = "x-pbs-debug-override"
)

type DebugLog struct {
	Enabled       bool
	CacheType     prebid_cache_client.PayloadType
	Data          DebugData
	TTL           int64
	CacheKey      string
	CacheString   string
	Regexp        *regexp.Regexp
	DebugOverride bool
	//little optimization, it stores value of debugLog.Enabled || debugLog.DebugOverride
	DebugEnabledOrOverridden bool
}

type DebugData struct {
	Request  string
	Headers  string
	Response string
}

func (d *DebugLog) BuildCacheString() {
	if d.Regexp != nil {
		d.Data.Request = fmt.Sprint(d.Regexp.ReplaceAllString(d.Data.Request, ""))
		d.Data.Headers = fmt.Sprint(d.Regexp.ReplaceAllString(d.Data.Headers, ""))
		d.Data.Response = fmt.Sprint(d.Regexp.ReplaceAllString(d.Data.Response, ""))
	}

	d.Data.Request = fmt.Sprintf("<Request>%s</Request>", d.Data.Request)
	d.Data.Headers = fmt.Sprintf("<Headers>%s</Headers>", d.Data.Headers)
	d.Data.Response = fmt.Sprintf("<Response>%s</Response>", d.Data.Response)

	d.CacheString = fmt.Sprintf("%s<Log>%s%s%s</Log>", xml.Header, d.Data.Request, d.Data.Headers, d.Data.Response)
}

func IsDebugOverrideEnabled(debugHeader, configOverrideToken string) bool {
	return configOverrideToken != "" && debugHeader == configOverrideToken
}

func (d *DebugLog) PutDebugLogError(cache prebid_cache_client.Client, timeout int, errors []error) error {
	if len(d.Data.Response) == 0 && len(errors) == 0 {
		d.Data.Response = "No response or errors created"
	}

	if len(errors) > 0 {
		errStrings := []string{}
		for _, err := range errors {
			errStrings = append(errStrings, err.Error())
		}
		d.Data.Response = fmt.Sprintf("%s\nErrors:\n%s", d.Data.Response, strings.Join(errStrings, "\n"))
	}

	d.BuildCacheString()

	if len(d.CacheKey) == 0 {
		rawUUID, err := uuid.NewV4()
		if err != nil {
			return err
		}
		d.CacheKey = rawUUID.String()
	}

	data, err := jsonutil.Marshal(d.CacheString)
	if err != nil {
		return err
	}

	toCache := []prebid_cache_client.Cacheable{
		{
			Type:       d.CacheType,
			Data:       data,
			TTLSeconds: d.TTL,
			Key:        "log_" + d.CacheKey,
		},
	}

	if cache != nil {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Duration(timeout)*time.Millisecond))
		defer cancel()
		cache.PutJson(ctx, toCache)
	}

	return nil
}

func newAuction(seatBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, numImps int, preferDeals bool) *auction {
	winningBids := make(map[string]*entities.PbsOrtbBid, numImps)
	allBidsByBidder := make(map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid, numImps)

	for bidderName, seatBid := range seatBids {
		if seatBid != nil {
			for _, bid := range seatBid.Bids {
				wbid, ok := winningBids[bid.Bid.ImpID]
				if !ok || isNewWinningBid(bid.Bid, wbid.Bid, preferDeals) {
					winningBids[bid.Bid.ImpID] = bid
				}

				if bidMap, ok := allBidsByBidder[bid.Bid.ImpID]; ok {
					bidMap[bidderName] = append(bidMap[bidderName], bid)
				} else {
					allBidsByBidder[bid.Bid.ImpID] = map[openrtb_ext.BidderName][]*entities.PbsOrtbBid{
						bidderName: {bid},
					}
				}
			}
		}
	}

	return &auction{
		winningBids:     winningBids,
		allBidsByBidder: allBidsByBidder,
	}
}

// isNewWinningBid calculates if the new bid (nbid) will win against the current winning bid (wbid) given preferDeals.
func isNewWinningBid(bid, wbid *openrtb2.Bid, preferDeals bool) bool {
	if preferDeals {
		if len(wbid.DealID) > 0 && len(bid.DealID) == 0 {
			return false
		}
		if len(wbid.DealID) == 0 && len(bid.DealID) > 0 {
			return true
		}
	}
	return bid.Price > wbid.Price
}

func (a *auction) validateAndUpdateMultiBid(adapterBids map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid, preferDeals bool, accountDefaultBidLimit int) {
	bidsSnipped := false
	// sort bids for multibid targeting
	for _, topBidsPerBidder := range a.allBidsByBidder {
		for bidder, topBids := range topBidsPerBidder {
			sort.Slice(topBids, func(i, j int) bool {
				return isNewWinningBid(topBids[i].Bid, topBids[j].Bid, preferDeals)
			})

			// assert hard limit on bids count per imp, per adapter.
			if accountDefaultBidLimit != 0 && len(topBids) > accountDefaultBidLimit {
				for i := accountDefaultBidLimit; i < len(topBids); i++ {
					topBids[i].Bid = nil
					topBids[i] = nil
					bidsSnipped = true
				}

				topBidsPerBidder[bidder] = topBids[:accountDefaultBidLimit]
			}
		}
	}

	if bidsSnipped { // remove the marked bids from original references
		for _, seatBid := range adapterBids {
			if seatBid != nil {
				bids := make([]*entities.PbsOrtbBid, 0, accountDefaultBidLimit)
				for i := 0; i < len(seatBid.Bids); i++ {
					if seatBid.Bids[i].Bid != nil {
						bids = append(bids, seatBid.Bids[i])
					}
				}
				seatBid.Bids = bids
			}
		}
	}
}

func (a *auction) setRoundedPrices(targetingData targetData) {
	roundedPrices := make(map[*entities.PbsOrtbBid]string, 5*len(a.winningBids))
	for _, topBidsPerImp := range a.allBidsByBidder {
		for _, topBidsPerBidder := range topBidsPerImp {
			for _, topBid := range topBidsPerBidder {
				roundedPrices[topBid] = GetPriceBucket(*topBid.Bid, targetingData)
			}
		}
	}
	a.roundedPrices = roundedPrices
}

func (a *auction) doCache(ctx context.Context, cache prebid_cache_client.Client, targData *targetData, evTracking *eventTracking, bidRequest *openrtb2.BidRequest, ttlBuffer int64, defaultTTLs *config.DefaultTTLs, bidCategory map[string]string, debugLog *DebugLog) []error {
	var bids, vast, includeBidderKeys, includeWinners bool = targData.includeCacheBids, targData.includeCacheVast, targData.includeBidderKeys, targData.includeWinners
	if !((bids || vast) && (includeBidderKeys || includeWinners)) {
		return nil
	}
	var errs []error
	expectNumBids := valOrZero(bids, len(a.roundedPrices))
	expectNumVast := valOrZero(vast, len(a.roundedPrices))
	bidIndices := make(map[int]*openrtb2.Bid, expectNumBids)
	vastIndices := make(map[int]*openrtb2.Bid, expectNumVast)
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
	for impID, topBidsPerImp := range a.allBidsByBidder {
		for bidderName, topBidsPerBidder := range topBidsPerImp {
			for _, topBid := range topBidsPerBidder {
				isOverallWinner := a.winningBids[impID] == topBid
				if !includeBidderKeys && !isOverallWinner {
					continue
				}
				var customCacheKey string
				var catDur string
				useCustomCacheKey := false
				if competitiveExclusion && isOverallWinner || includeBidderKeys {
					// set custom cache key for winning bid when competitive exclusion applies
					catDur = bidCategory[topBid.Bid.ID]
					if len(catDur) > 0 {
						customCacheKey = fmt.Sprintf("%s_%s", catDur, hbCacheID)
						useCustomCacheKey = true
					}
				}
				if bids {
					if jsonBytes, err := jsonutil.Marshal(topBid.Bid); err == nil {
						jsonBytes, err = evTracking.modifyBidJSON(topBid, bidderName, jsonBytes)
						if err != nil {
							errs = append(errs, err)
						}
						if useCustomCacheKey {
							// not allowed if bids is true; log error and cache normally
							errs = append(errs, errors.New("cannot use custom cache key for non-vast bids"))
						}
						toCache = append(toCache, prebid_cache_client.Cacheable{
							Type:       prebid_cache_client.TypeJSON,
							Data:       jsonBytes,
							TTLSeconds: cacheTTL(expByImp[impID], topBid.Bid.Exp, defTTL(topBid.BidType, defaultTTLs), ttlBuffer),
						})
						bidIndices[len(toCache)-1] = topBid.Bid
					} else {
						errs = append(errs, err)
					}
				}
				if vast && topBid.BidType == openrtb_ext.BidTypeVideo {
					vastXML := makeVAST(topBid.Bid)
					if jsonBytes, err := jsonutil.Marshal(vastXML); err == nil {
						if useCustomCacheKey {
							toCache = append(toCache, prebid_cache_client.Cacheable{
								Type:       prebid_cache_client.TypeXML,
								Data:       jsonBytes,
								TTLSeconds: cacheTTL(expByImp[impID], topBid.Bid.Exp, defTTL(topBid.BidType, defaultTTLs), ttlBuffer),
								Key:        customCacheKey,
							})
						} else {
							toCache = append(toCache, prebid_cache_client.Cacheable{
								Type:       prebid_cache_client.TypeXML,
								Data:       jsonBytes,
								TTLSeconds: cacheTTL(expByImp[impID], topBid.Bid.Exp, defTTL(topBid.BidType, defaultTTLs), ttlBuffer),
							})
						}
						vastIndices[len(toCache)-1] = topBid.Bid
					} else {
						errs = append(errs, err)
					}
				}
			}
		}
	}

	if len(toCache) > 0 && debugLog != nil && debugLog.DebugEnabledOrOverridden {
		debugLog.CacheKey = hbCacheID
		debugLog.BuildCacheString()
		if jsonBytes, err := jsonutil.Marshal(debugLog.CacheString); err == nil {
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
		a.cacheIds = make(map[*openrtb2.Bid]string, len(bidIndices))
		for index, bid := range bidIndices {
			if ids[index] != "" {
				a.cacheIds[bid] = ids[index]
			}
		}
	}
	if vast {
		a.vastCacheIds = make(map[*openrtb2.Bid]string, len(vastIndices))
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
func makeVAST(bid *openrtb2.Bid) string {
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
	winningBids map[string]*entities.PbsOrtbBid
	// allBidsByBidder is map from ImpID to another map that maps bidderName to all bids from that bidder.
	allBidsByBidder map[string]map[openrtb_ext.BidderName][]*entities.PbsOrtbBid
	// roundedPrices stores the price strings rounded for each bid according to the price granularity.
	roundedPrices map[*entities.PbsOrtbBid]string
	// cacheIds stores the UUIDs from Prebid Cache for fetching the full bid JSON.
	cacheIds map[*openrtb2.Bid]string
	// vastCacheIds stores UUIDS from Prebid cache for fetching the VAST markup to video bids.
	vastCacheIds map[*openrtb2.Bid]string
}
