package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/prebid/prebid-server/pbsmetrics"

	"github.com/cloudfoundry/gosigar"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/cors"
	"github.com/spf13/viper"
	"github.com/vrischmann/go-metrics-influxdb"
	"github.com/xeipuuv/gojsonschema"

	"os"
	"os/signal"
	"syscall"

	"crypto/tls"
	"strings"

	_ "github.com/lib/pq"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/audienceNetwork"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/indexExchange"
	"github.com/prebid/prebid-server/adapters/lifestreet"
	"github.com/prebid/prebid-server/adapters/pubmatic"
	"github.com/prebid/prebid-server/adapters/pulsepoint"
	"github.com/prebid/prebid-server/adapters/rubicon"
	"github.com/prebid/prebid-server/adapters/sovrn"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/cache/filecache"
	"github.com/prebid/prebid-server/cache/postgrescache"
	"github.com/prebid/prebid-server/config"
	infoEndpoints "github.com/prebid/prebid-server/endpoints/info"
	"github.com/prebid/prebid-server/endpoints/openrtb2"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbs/buckets"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/ssl"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/db_fetcher"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/stored_requests/backends/file_fetcher"
	"github.com/prebid/prebid-server/stored_requests/caches/in_memory"
	usersyncers "github.com/prebid/prebid-server/usersync"
)

type DomainMetrics struct {
	RequestMeter metrics.Meter
}

type AccountMetrics struct {
	RequestMeter      metrics.Meter
	BidsReceivedMeter metrics.Meter
	PriceHistogram    metrics.Histogram
	// store account by adapter metrics. Type is map[PBSBidder.BidderCode]
	AdapterMetrics map[string]*AdapterMetrics
}

type AdapterMetrics struct {
	NoCookieMeter     metrics.Meter
	ErrorMeter        metrics.Meter
	NoBidMeter        metrics.Meter
	TimeoutMeter      metrics.Meter
	RequestMeter      metrics.Meter
	RequestTimer      metrics.Timer
	PriceHistogram    metrics.Histogram
	BidsReceivedMeter metrics.Meter
}

var (
	metricsRegistry      metrics.Registry
	mRequestMeter        metrics.Meter
	mAppRequestMeter     metrics.Meter
	mNoCookieMeter       metrics.Meter
	mSafariRequestMeter  metrics.Meter
	mSafariNoCookieMeter metrics.Meter
	mErrorMeter          metrics.Meter
	mInvalidMeter        metrics.Meter
	mRequestTimer        metrics.Timer
	mCookieSyncMeter     metrics.Meter

	adapterMetrics map[string]*AdapterMetrics

	accountMetrics        map[string]*AccountMetrics // FIXME -- this seems like an unbounded queue
	accountMetricsRWMutex sync.RWMutex

	hostCookieSettings pbs.HostCookieSettings
)

var exchanges map[string]adapters.Adapter
var dataCache cache.Cache
var reqSchema *gojsonschema.Schema

type bidResult struct {
	bidder   *pbs.PBSBidder
	bid_list pbs.PBSBidSlice
}

const schemaDirectory = "./static/bidder-params"

const defaultPriceGranularity = "med"

// Constant keys for ad server targeting for responses to Prebid Mobile
const hbpbConstantKey = "hb_pb"
const hbCreativeLoadMethodConstantKey = "hb_creative_loadtype"
const hbBidderConstantKey = "hb_bidder"
const hbCacheIdConstantKey = "hb_cache_id"
const hbDealIdConstantKey = "hb_deal"
const hbSizeConstantKey = "hb_size"

// hb_creative_loadtype key can be one of `demand_sdk` or `html`
// default is `html` where the creative is loaded in the primary ad server's webview through AppNexus hosted JS
// `demand_sdk` is for bidders who insist on their creatives being loaded in their own SDK's webview
const hbCreativeLoadMethodHTML = "html"
const hbCreativeLoadMethodDemandSDK = "demand_sdk"

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func writeAuctionError(w http.ResponseWriter, s string, err error) {
	var resp pbs.PBSResponse
	if err != nil {
		resp.Status = fmt.Sprintf("%s: %v", s, err)
	} else {
		resp.Status = s
	}
	b, err := json.Marshal(&resp)
	if err != nil {
		glog.Errorf("Failed to marshal auction error JSON: %s", err)
	} else {
		w.Write(b)
	}
}

func getAccountMetrics(id string) *AccountMetrics {
	var am *AccountMetrics
	var ok bool

	accountMetricsRWMutex.RLock()
	am, ok = accountMetrics[id]
	accountMetricsRWMutex.RUnlock()

	if ok {
		return am
	}

	accountMetricsRWMutex.Lock()
	am, ok = accountMetrics[id]
	if !ok {
		am = &AccountMetrics{}
		am.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.requests", id), metricsRegistry)
		am.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.bids_received", id), metricsRegistry)
		am.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("account.%s.prices", id), metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		am.AdapterMetrics = makeExchangeMetrics(fmt.Sprintf("account.%s", id))
		accountMetrics[id] = am
	}
	accountMetricsRWMutex.Unlock()

	return am
}

type cookieSyncRequest struct {
	Bidders []string `json:"bidders"`
}

type cookieSyncResponse struct {
	Status       string           `json:"status"`
	BidderStatus []*pbs.PBSBidder `json:"bidder_status"`
}

type cookieSyncDeps struct {
	syncers      map[openrtb_ext.BidderName]usersyncers.Usersyncer
	optOutCookie *config.Cookie
	metric       metrics.Meter
}

func (deps *cookieSyncDeps) CookieSync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	deps.metric.Mark(1)
	userSyncCookie := pbs.ParsePBSCookieFromRequest(r, deps.optOutCookie)
	if !userSyncCookie.AllowSyncs() {
		http.Error(w, "User has opted out", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()

	csReq := &cookieSyncRequest{}
	csReqRaw := map[string]json.RawMessage{}
	err := json.NewDecoder(r.Body).Decode(&csReqRaw)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Failed to parse /cookie_sync request body: %v", err)
		}
		http.Error(w, "JSON parse failed", http.StatusBadRequest)
		return
	}
	biddersOmitted := true
	if biddersRaw, ok := csReqRaw["bidders"]; ok {
		biddersOmitted = false
		err := json.Unmarshal(biddersRaw, &csReq.Bidders)
		if err != nil {
			if glog.V(2) {
				glog.Infof("Failed to parse /cookie_sync request body (bidders list): %v", err)
			}
			http.Error(w, "JSON parse failed (bidders)", http.StatusBadRequest)
			return
		}
	}

	csResp := cookieSyncResponse{
		BidderStatus: make([]*pbs.PBSBidder, 0, len(csReq.Bidders)),
	}

	if userSyncCookie.LiveSyncCount() == 0 {
		csResp.Status = "no_cookie"
	} else {
		csResp.Status = "ok"
	}

	// If at the end (After possibly reading stored bidder lists) there still are no bidders,
	// and "bidders" is not found in the JSON, sync all bidders
	if len(csReq.Bidders) == 0 && biddersOmitted {
		for bidder := range deps.syncers {
			csReq.Bidders = append(csReq.Bidders, string(bidder))
		}
	}

	for _, bidder := range csReq.Bidders {
		if syncer, ok := deps.syncers[openrtb_ext.BidderName(bidder)]; ok {
			if !userSyncCookie.HasLiveSync(syncer.FamilyName()) {
				b := pbs.PBSBidder{
					BidderCode:   bidder,
					NoCookie:     true,
					UsersyncInfo: syncer.GetUsersyncInfo(),
				}
				csResp.BidderStatus = append(csResp.BidderStatus, &b)
			}
		}
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	//enc.SetIndent("", "  ")
	enc.Encode(csResp)
}

type auctionDeps struct {
	cfg     *config.Configuration
	syncers map[openrtb_ext.BidderName]usersyncers.Usersyncer
}

func (deps *auctionDeps) auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Add("Content-Type", "application/json")

	mRequestMeter.Mark(1)

	isSafari := false
	if ua := user_agent.New(r.Header.Get("User-Agent")); ua != nil {
		name, _ := ua.Browser()
		if name == "Safari" {
			isSafari = true
			mSafariRequestMeter.Mark(1)
		}
	}

	pbs_req, err := pbs.ParsePBSRequest(r, dataCache, &hostCookieSettings)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Failed to parse /auction request: %v", err)
		}
		writeAuctionError(w, "Error parsing request", err)
		mErrorMeter.Mark(1)
		return
	}

	status := "OK"
	if pbs_req.App != nil {
		mAppRequestMeter.Mark(1)
	} else if pbs_req.Cookie.LiveSyncCount() == 0 {
		mNoCookieMeter.Mark(1)
		if isSafari {
			mSafariNoCookieMeter.Mark(1)
		}
		status = "no_cookie"
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pbs_req.TimeoutMillis))
	defer cancel()

	account, err := dataCache.Accounts().Get(pbs_req.AccountID)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Invalid account id: %v", err)
		}
		writeAuctionError(w, "Unknown account id", fmt.Errorf("Unknown account"))
		mErrorMeter.Mark(1)
		return
	}

	am := getAccountMetrics(pbs_req.AccountID)
	am.RequestMeter.Mark(1)

	pbs_resp := pbs.PBSResponse{
		Status:       status,
		TID:          pbs_req.Tid,
		BidderStatus: pbs_req.Bidders,
	}

	ch := make(chan bidResult)
	sentBids := 0
	for _, bidder := range pbs_req.Bidders {
		if ex, ok := exchanges[bidder.BidderCode]; ok {
			ametrics := adapterMetrics[bidder.BidderCode]
			accountAdapterMetric := am.AdapterMetrics[bidder.BidderCode]
			ametrics.RequestMeter.Mark(1)
			accountAdapterMetric.RequestMeter.Mark(1)
			if pbs_req.App == nil {
				// If exchanges[bidderCode] exists, then deps.syncers[bidderCode] exists *except for districtm*.
				// OpenRTB handles aliases differently, so this hack will keep legacy code working. For all other
				// bidderCodes, deps.syncers[bidderCode] will exist if exchanges[bidderCode] also does.
				// This is guaranteed by the following unit tests, which compare these maps to the (source of truth) openrtb_ext.BidderMap:
				//   1. TestSyncers inside usersync/usersync_test.go
				//   2. TestExchangeMap inside pbs_light_test.go
				syncerCode := bidder.BidderCode
				if syncerCode == "districtm" {
					syncerCode = "appnexus"
				}
				syncer := deps.syncers[openrtb_ext.BidderName(syncerCode)]
				uid, _, _ := pbs_req.Cookie.GetUID(syncer.FamilyName())
				if uid == "" {
					bidder.NoCookie = true
					bidder.UsersyncInfo = syncer.GetUsersyncInfo()
					ametrics.NoCookieMeter.Mark(1)
					accountAdapterMetric.NoCookieMeter.Mark(1)
					if ex.SkipNoCookies() {
						continue
					}
				}
			}
			sentBids++
			go func(bidder *pbs.PBSBidder) {
				start := time.Now()
				bid_list, err := ex.Call(ctx, pbs_req, bidder)
				bidder.ResponseTime = int(time.Since(start) / time.Millisecond)
				ametrics.RequestTimer.UpdateSince(start)
				accountAdapterMetric.RequestTimer.UpdateSince(start)
				if err != nil {
					switch err {
					case context.DeadlineExceeded:
						ametrics.TimeoutMeter.Mark(1)
						accountAdapterMetric.TimeoutMeter.Mark(1)
						bidder.Error = "Timed out"
					case context.Canceled:
						fallthrough
					default:
						ametrics.ErrorMeter.Mark(1)
						accountAdapterMetric.ErrorMeter.Mark(1)
						bidder.Error = err.Error()
						glog.Warningf("Error from bidder %v. Ignoring all bids: %v", bidder.BidderCode, err)
					}
				} else if bid_list != nil {
					bid_list = checkForValidBidSize(bid_list, bidder)
					bidder.NumBids = len(bid_list)
					am.BidsReceivedMeter.Mark(int64(bidder.NumBids))
					accountAdapterMetric.BidsReceivedMeter.Mark(int64(bidder.NumBids))
					for _, bid := range bid_list {
						var cpm = int64(bid.Price * 1000)
						ametrics.PriceHistogram.Update(cpm)
						am.PriceHistogram.Update(cpm)
						accountAdapterMetric.PriceHistogram.Update(cpm)
						bid.ResponseTime = bidder.ResponseTime
					}
				} else {
					bidder.NoBid = true
					ametrics.NoBidMeter.Mark(1)
					accountAdapterMetric.NoBidMeter.Mark(1)
				}

				ch <- bidResult{
					bidder:   bidder,
					bid_list: bid_list,
				}
			}(bidder)

		} else {
			bidder.Error = "Unsupported bidder"
		}
	}

	for i := 0; i < sentBids; i++ {
		result := <-ch

		for _, bid := range result.bid_list {
			pbs_resp.Bids = append(pbs_resp.Bids, bid)
		}
	}
	if pbs_req.CacheMarkup == 1 {
		cobjs := make([]*pbc.CacheObject, len(pbs_resp.Bids))
		for i, bid := range pbs_resp.Bids {
			if bid.CreativeMediaType == "video" {
				cobjs[i] = &pbc.CacheObject{
					Value:   bid.Adm,
					IsVideo: true,
				}
			} else {
				cobjs[i] = &pbc.CacheObject{
					Value: &pbc.BidCache{
						Adm:    bid.Adm,
						NURL:   bid.NURL,
						Width:  bid.Width,
						Height: bid.Height,
					},
					IsVideo: false,
				}
			}
		}
		err = pbc.Put(ctx, cobjs)
		if err != nil {
			writeAuctionError(w, "Prebid cache failed", err)
			mErrorMeter.Mark(1)
			return
		}
		for i, bid := range pbs_resp.Bids {
			bid.CacheID = cobjs[i].UUID
			bid.CacheURL = deps.cfg.GetCachedAssetURL(bid.CacheID)
			bid.NURL = ""
			bid.Adm = ""
		}
	}

	if pbs_req.CacheMarkup == 2 {
		cacheVideoOnly(pbs_resp.Bids, ctx, w, deps)
	}

	if pbs_req.SortBids == 1 {
		sortBidsAddKeywordsMobile(pbs_resp.Bids, pbs_req, account.PriceGranularity)
	}

	if glog.V(2) {
		glog.Infof("Request for %d ad units on url %s by account %s got %d bids", len(pbs_req.AdUnits), pbs_req.Url, pbs_req.AccountID, len(pbs_resp.Bids))
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(pbs_resp)
	mRequestTimer.UpdateSince(pbs_req.Start)
}

// cache video bids only for Web
func cacheVideoOnly(bids pbs.PBSBidSlice, ctx context.Context, w http.ResponseWriter, deps *auctionDeps) {
	var cobjs []*pbc.CacheObject
	for _, bid := range bids {
		if bid.CreativeMediaType == "video" {
			cobjs = append(cobjs, &pbc.CacheObject{
				Value:   bid.Adm,
				IsVideo: true,
			})
		}
	}
	err := pbc.Put(ctx, cobjs)
	if err != nil {
		writeAuctionError(w, "Prebid cache failed", err)
		mErrorMeter.Mark(1)
		return
	}
	videoIndex := 0
	for _, bid := range bids {
		if bid.CreativeMediaType == "video" {
			bid.CacheID = cobjs[videoIndex].UUID
			bid.CacheURL = deps.cfg.GetCachedAssetURL(bid.CacheID)
			bid.NURL = ""
			bid.Adm = ""
			videoIndex++
		}
	}
}

// checkForValidBidSize goes through list of bids & find those which are banner mediaType and with height or width not defined
// determine the num of ad unit sizes that were used in corresponding bid request
// if num_adunit_sizes == 1, assign the height and/or width to bid's height/width
// if num_adunit_sizes > 1, reject the bid (remove from list) and return an error
// return updated bid list object for next steps in auction
func checkForValidBidSize(bids pbs.PBSBidSlice, bidder *pbs.PBSBidder) pbs.PBSBidSlice {
	finalValidBids := make([]*pbs.PBSBid, len(bids))
	finalBidCounter := 0
bidLoop:
	for _, bid := range bids {
		if bid.CreativeMediaType == "banner" && (bid.Height == 0 || bid.Width == 0) {
			for _, adunit := range bidder.AdUnits {
				if adunit.BidID == bid.BidID && adunit.Code == bid.AdUnitCode {
					if len(adunit.Sizes) == 1 {
						bid.Width, bid.Height = adunit.Sizes[0].W, adunit.Sizes[0].H
						finalValidBids[finalBidCounter] = bid
						finalBidCounter = finalBidCounter + 1
					} else if len(adunit.Sizes) > 1 {
						glog.Warningf("Bid was rejected for bidder %s because no size was defined", bid.BidderCode)
					}
					continue bidLoop
				}
			}
		} else {
			finalValidBids[finalBidCounter] = bid
			finalBidCounter = finalBidCounter + 1
		}
	}
	return finalValidBids[:finalBidCounter]
}

// sortBidsAddKeywordsMobile sorts the bids and adds ad server targeting keywords to each bid.
// The bids are sorted by cpm to find the highest bid.
// The ad server targeting keywords are added to all bids, with specific keywords for the highest bid.
func sortBidsAddKeywordsMobile(bids pbs.PBSBidSlice, pbs_req *pbs.PBSRequest, priceGranularitySetting openrtb_ext.PriceGranularity) {
	if priceGranularitySetting == "" {
		priceGranularitySetting = defaultPriceGranularity
	}

	// record bids by ad unit code for sorting
	code_bids := make(map[string]pbs.PBSBidSlice, len(bids))
	for _, bid := range bids {
		code_bids[bid.AdUnitCode] = append(code_bids[bid.AdUnitCode], bid)
	}

	// loop through ad units to find top bid
	for _, unit := range pbs_req.AdUnits {
		bar := code_bids[unit.Code]

		if len(bar) == 0 {
			if glog.V(3) {
				glog.Infof("No bids for ad unit '%s'", unit.Code)
			}
			continue
		}
		sort.Sort(bar)

		// after sorting we need to add the ad targeting keywords
		for i, bid := range bar {
			// We should eventually check for the error and do something.
			roundedCpm, err := buckets.GetPriceBucketString(bid.Price, priceGranularitySetting)
			if err != nil {
				glog.Error(err.Error())
			}

			hbSize := ""
			if bid.Width != 0 && bid.Height != 0 {
				width := strconv.FormatUint(bid.Width, 10)
				height := strconv.FormatUint(bid.Height, 10)
				hbSize = width + "x" + height
			}

			hbPbBidderKey := hbpbConstantKey + "_" + bid.BidderCode
			hbBidderBidderKey := hbBidderConstantKey + "_" + bid.BidderCode
			hbCacheIdBidderKey := hbCacheIdConstantKey + "_" + bid.BidderCode
			hbDealIdBidderKey := hbDealIdConstantKey + "_" + bid.BidderCode
			hbSizeBidderKey := hbSizeConstantKey + "_" + bid.BidderCode
			if pbs_req.MaxKeyLength != 0 {
				hbPbBidderKey = hbPbBidderKey[:min(len(hbPbBidderKey), int(pbs_req.MaxKeyLength))]
				hbBidderBidderKey = hbBidderBidderKey[:min(len(hbBidderBidderKey), int(pbs_req.MaxKeyLength))]
				hbCacheIdBidderKey = hbCacheIdBidderKey[:min(len(hbCacheIdBidderKey), int(pbs_req.MaxKeyLength))]
				hbDealIdBidderKey = hbDealIdBidderKey[:min(len(hbDealIdBidderKey), int(pbs_req.MaxKeyLength))]
				hbSizeBidderKey = hbSizeBidderKey[:min(len(hbSizeBidderKey), int(pbs_req.MaxKeyLength))]
			}

			// fixes #288 where map was being overwritten instead of updated
			if bid.AdServerTargeting == nil {
				bid.AdServerTargeting = make(map[string]string)
			}
			pbs_kvs := bid.AdServerTargeting

			pbs_kvs[hbPbBidderKey] = roundedCpm
			pbs_kvs[hbBidderBidderKey] = bid.BidderCode
			pbs_kvs[hbCacheIdBidderKey] = bid.CacheID

			if hbSize != "" {
				pbs_kvs[hbSizeBidderKey] = hbSize
			}
			if bid.DealId != "" {
				pbs_kvs[hbDealIdBidderKey] = bid.DealId
			}
			// For the top bid, we want to add the following additional keys
			if i == 0 {
				pbs_kvs[hbpbConstantKey] = roundedCpm
				pbs_kvs[hbBidderConstantKey] = bid.BidderCode
				pbs_kvs[hbCacheIdConstantKey] = bid.CacheID
				if bid.DealId != "" {
					pbs_kvs[hbDealIdConstantKey] = bid.DealId
				}
				if hbSize != "" {
					pbs_kvs[hbSizeConstantKey] = hbSize
				}
				if bid.BidderCode == "audienceNetwork" {
					pbs_kvs[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodDemandSDK
				} else {
					pbs_kvs[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodHTML
				}
			}
		}
	}
}

func status(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// could add more logic here, but doing nothing means 200 OK
}

// NewJsonDirectoryServer is used to serve .json files from a directory as a single blob. For example,
// given a directory containing the files "a.json" and "b.json", this returns a Handle which serves JSON like:
//
// {
//   "a": { ... content from the file a.json ... },
//   "b": { ... content from the file b.json ... }
// }
//
// This function stores the file contents in memory, and should not be used on large directories.
// If the root directory, or any of the files in it, cannot be read, then the program will exit.
func NewJsonDirectoryServer(validator openrtb_ext.BidderParamValidator) httprouter.Handle {
	// Slurp the files into memory first, since they're small and it minimizes request latency.
	files, err := ioutil.ReadDir(schemaDirectory)
	if err != nil {
		glog.Fatalf("Failed to read directory %s: %v", schemaDirectory, err)
	}

	data := make(map[string]json.RawMessage, len(files))
	for _, file := range files {
		bidder := strings.TrimSuffix(file.Name(), ".json")
		bidderName, isValid := openrtb_ext.BidderMap[bidder]
		if !isValid {
			glog.Fatalf("Schema exists for an unknown bidder: %s", bidder)
		}
		data[bidder] = json.RawMessage(validator.Schema(bidderName))
	}
	response, err := json.Marshal(data)
	if err != nil {
		glog.Fatalf("Failed to marshal bidder param JSON-schema: %v", err)
	}

	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		w.Header().Add("Content-Type", "application/json")
		w.Write(response)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "static/index.html")
}

type NoCache struct {
	handler http.Handler
}

func (m NoCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Add("Pragma", "no-cache")
	w.Header().Add("Expires", "0")
	m.handler.ServeHTTP(w, r)
}

func validate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Add("Content-Type", "text/plain")
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Unable to read body\n")
		return
	}

	if reqSchema == nil {
		fmt.Fprintf(w, "Validation schema not loaded\n")
		return
	}

	js := gojsonschema.NewStringLoader(string(b))
	result, err := reqSchema.Validate(js)
	if err != nil {
		fmt.Fprintf(w, "Error parsing json: %v\n", err)
		return
	}

	if result.Valid() {
		fmt.Fprintf(w, "Validation successful\n")
		return
	}

	for _, err := range result.Errors() {
		fmt.Fprintf(w, "Error: %s %v\n", err.Context().String(), err)
	}

	return
}

func loadDataCache(cfg *config.Configuration, db *sql.DB) (err error) {
	switch cfg.DataCache.Type {
	case "dummy":
		dataCache, err = dummycache.New()
		if err != nil {
			glog.Fatalf("Dummy cache not configured: %s", err.Error())
		}

	case "postgres":
		if db == nil {
			return fmt.Errorf("Nil db cannot connect to postgres. Did you forget to set the config.stored_requests.postgres values?")
		}
		mem := sigar.Mem{}
		mem.Get()
		dataCache = postgrescache.New(db, postgrescache.CacheConfig{
			Size: cfg.DataCache.CacheSize,
			TTL:  cfg.DataCache.TTLSeconds,
		})
		return nil
	case "filecache":
		dataCache, err = filecache.New(cfg.DataCache.Filename)
		if err != nil {
			return fmt.Errorf("FileCache Error: %s", err.Error())
		}

	default:
		return fmt.Errorf("Unknown datacache.type: %s", cfg.DataCache.Type)
	}
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
	viper.SetConfigName("pbs")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/config")

	viper.SetDefault("external_url", "http://localhost:8000")
	viper.SetDefault("port", 8000)
	viper.SetDefault("admin_port", 6060)
	viper.SetDefault("default_timeout_ms", 250)
	viper.SetDefault("cache.expected_millis", 10)
	viper.SetDefault("datacache.type", "dummy")
	// no metrics configured by default (metrics{host|database|username|password})

	viper.SetDefault("stored_requests.filesystem", "true")
	viper.SetDefault("adapters.pubmatic.endpoint", "http://hbopenbid.pubmatic.com/translator?source=prebid-server")
	viper.SetDefault("adapters.rubicon.endpoint", "http://exapi-us-east.rubiconproject.com/a/api/exchange.json")
	viper.SetDefault("adapters.rubicon.usersync_url", "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid")
	viper.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	viper.SetDefault("adapters.index.usersync_url", "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=https%3A%2F%2Fprebid.adnxs.com%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3DindexExchange%26uid%3D")
	viper.SetDefault("adapters.sovrn.endpoint", "http://ap.lijit.com/rtb/bid?src=prebid_server")
	viper.SetDefault("adapters.sovrn.usersync_url", "//ap.lijit.com/pixel?")
	viper.SetDefault("adapters.adform.endpoint", "http://adx.adform.net/adx")
	viper.SetDefault("adapters.adform.usersync_url", "//cm.adform.net/cookie?redirect_url=")
	viper.SetDefault("max_request_size", 1024*256)
	viper.SetDefault("adapters.conversant.endpoint", "http://media.msg.dotomi.com/s2s/header/24")
	viper.SetDefault("adapters.conversant.usersync_url", "http://prebid-match.dotomi.com/prebid/match?rurl=")
	viper.SetDefault("host_cookie.ttl_days", 90)

	// Set environment variable support:
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix("PBS")
	viper.AutomaticEnv()
	viper.ReadInConfig()

	flag.Parse() // read glog settings from cmd line
}

func main() {
	cfg, err := config.New(viper.GetViper())
	if err != nil {
		glog.Fatalf("Viper was unable to read configurations: %v", err)
	}

	if err := serve(cfg); err != nil {
		glog.Errorf("prebid-server failed: %v", err)
	}
}

func setupExchanges(cfg *config.Configuration) {
	exchanges = newExchangeMap(cfg)

	metricsRegistry = metrics.NewPrefixedRegistry("prebidserver.")
	mRequestMeter = metrics.GetOrRegisterMeter("requests", metricsRegistry)
	mAppRequestMeter = metrics.GetOrRegisterMeter("app_requests", metricsRegistry)
	mNoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", metricsRegistry)
	mSafariRequestMeter = metrics.GetOrRegisterMeter("safari_requests", metricsRegistry)
	mSafariNoCookieMeter = metrics.GetOrRegisterMeter("safari_no_cookie_requests", metricsRegistry)
	mErrorMeter = metrics.GetOrRegisterMeter("error_requests", metricsRegistry)
	mInvalidMeter = metrics.GetOrRegisterMeter("invalid_requests", metricsRegistry)
	mRequestTimer = metrics.GetOrRegisterTimer("request_time", metricsRegistry)
	mCookieSyncMeter = metrics.GetOrRegisterMeter("cookie_sync_requests", metricsRegistry)

	accountMetrics = make(map[string]*AccountMetrics)
	adapterMetrics = makeExchangeMetrics("adapter")
}

func newExchangeMap(cfg *config.Configuration) map[string]adapters.Adapter {
	// These keys _must_ coincide with the bidder code in Prebid.js, if the adapter exists in both projects
	return map[string]adapters.Adapter{
		"appnexus":      appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig),
		"districtm":     appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig),
		"indexExchange": indexExchange.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["indexexchange"].Endpoint),
		"pubmatic":      pubmatic.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pubmatic"].Endpoint),
		"pulsepoint":    pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pulsepoint"].Endpoint),
		"rubicon": rubicon.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["rubicon"].Endpoint,
			cfg.Adapters["rubicon"].XAPI.Username, cfg.Adapters["rubicon"].XAPI.Password, cfg.Adapters["rubicon"].XAPI.Tracker),
		"audienceNetwork": audienceNetwork.NewAdapterFromFacebook(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["facebook"].PlatformID),
		"lifestreet":      lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig),
		"conversant":      conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["conversant"].Endpoint),
		"adform":          adform.NewAdformAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["adform"].Endpoint),
		"sovrn":           sovrn.NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["sovrn"].Endpoint),
	}
}

func makeExchangeMetrics(adapterOrAccount string) map[string]*AdapterMetrics {
	var adapterMetrics = make(map[string]*AdapterMetrics)
	for exchange := range exchanges {
		a := AdapterMetrics{}
		a.NoCookieMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_cookie_requests", adapterOrAccount, exchange), metricsRegistry)
		a.ErrorMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.error_requests", adapterOrAccount, exchange), metricsRegistry)
		a.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.requests", adapterOrAccount, exchange), metricsRegistry)
		a.NoBidMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.no_bid_requests", adapterOrAccount, exchange), metricsRegistry)
		a.TimeoutMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.timeout_requests", adapterOrAccount, exchange), metricsRegistry)
		a.RequestTimer = metrics.GetOrRegisterTimer(fmt.Sprintf("%[1]s.%[2]s.request_time", adapterOrAccount, exchange), metricsRegistry)
		a.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("%[1]s.%[2]s.prices", adapterOrAccount, exchange), metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		if adapterOrAccount != "adapter" {
			a.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("%[1]s.%[2]s.bids_received", adapterOrAccount, exchange), metricsRegistry)
		}

		adapterMetrics[exchange] = &a
	}
	return adapterMetrics
}

func serve(cfg *config.Configuration) error {
	var db *sql.DB
	if cfg.StoredRequests.Postgres != nil {
		if conn, err := db_fetcher.NewPostgresDb(cfg.StoredRequests.Postgres); err != nil {
			glog.Fatalf("Failed to connect to postgres: %v", err)
		} else {
			db = conn
		}
	}
	if err := loadDataCache(cfg, db); err != nil {
		return fmt.Errorf("Prebid Server could not load data cache: %v", err)
	}

	setupExchanges(cfg)

	if cfg.Metrics.Host != "" {
		go influxdb.InfluxDB(
			metricsRegistry,      // metrics registry
			time.Second*10,       // interval
			cfg.Metrics.Host,     // the InfluxDB url
			cfg.Metrics.Database, // your InfluxDB database
			cfg.Metrics.Username, // your InfluxDB user
			cfg.Metrics.Password, // your InfluxDB password
		)
	}

	b, err := ioutil.ReadFile("static/pbs_request.json")
	if err != nil {
		glog.Errorf("Unable to open pbs_request.json: %v", err)
	} else {
		sl := gojsonschema.NewStringLoader(string(b))
		reqSchema, err = gojsonschema.NewSchema(sl)
		if err != nil {
			glog.Errorf("Unable to load request schema: %v", err)
		}
	}

	stopSignals := make(chan os.Signal)
	signal.Notify(stopSignals, syscall.SIGTERM, syscall.SIGINT)

	/* Run admin on different port thats not exposed */
	adminURI := fmt.Sprintf("%s:%d", cfg.Host, cfg.AdminPort)
	adminServer := &http.Server{Addr: adminURI}
	go (func() {
		fmt.Println("Admin running on: ", adminURI)
		err := adminServer.ListenAndServe()
		glog.Errorf("Admin server: %v", err)
		stopSignals <- syscall.SIGTERM
	})()

	paramsValidator, err := openrtb_ext.NewBidderParamsValidator(schemaDirectory)
	if err != nil {
		glog.Fatalf("Failed to create the bidder params validator. %v", err)
	}

	// TODO: Currently setupExchanges() creates metricsRegistry. We will need to do this
	// here if/when the legacy endpoint goes away.
	theClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        400,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     60 * time.Second,
			TLSClientConfig:     &tls.Config{RootCAs: ssl.GetRootCAPool()},
		},
	}
	theMetrics := pbsmetrics.NewMetrics(metricsRegistry, openrtb_ext.BidderList())
	theExchange := exchange.NewExchange(theClient, pbc.NewClient(&cfg.CacheURL), cfg, theMetrics)

	byId, byAmpId, err := NewFetchers(&(cfg.StoredRequests), db)
	if err != nil {
		glog.Fatalf("Failed to initialize config backends. %v", err)
	}

	openrtbEndpoint, err := openrtb2.NewEndpoint(theExchange, paramsValidator, byId, cfg, theMetrics)
	if err != nil {
		glog.Fatalf("Failed to create the openrtb endpoint handler. %v", err)
	}

	ampEndpoint, err := openrtb2.NewAmpEndpoint(theExchange, paramsValidator, byAmpId, cfg, theMetrics)
	if err != nil {
		glog.Fatalf("Failed to create the amp endpoint handler. %v", err)
	}

	syncers := usersyncers.NewSyncerMap(cfg)

	router := httprouter.New()
	router.POST("/auction", (&auctionDeps{cfg, syncers}).auction)
	router.POST("/openrtb2/auction", openrtbEndpoint)
	router.GET("/openrtb2/amp", ampEndpoint)
	router.GET("/info/bidders", infoEndpoints.NewBiddersEndpoint())
	router.GET("/info/bidders/:bidderName", infoEndpoints.NewBidderDetailsEndpoint("./static/bidder-info", openrtb_ext.BidderList()))
	router.GET("/bidders/params", NewJsonDirectoryServer(paramsValidator))
	router.POST("/cookie_sync", (&cookieSyncDeps{syncers, &(hostCookieSettings.OptOutCookie), mCookieSyncMeter}).CookieSync)
	router.POST("/validate", validate)
	router.GET("/status", status)
	router.GET("/", serveIndex)
	router.ServeFiles("/static/*filepath", http.Dir("static"))

	hostCookieSettings = pbs.HostCookieSettings{
		Domain:       cfg.HostCookie.Domain,
		Family:       cfg.HostCookie.Family,
		CookieName:   cfg.HostCookie.CookieName,
		OptOutURL:    cfg.HostCookie.OptOutURL,
		OptInURL:     cfg.HostCookie.OptInURL,
		OptOutCookie: cfg.HostCookie.OptOutCookie,
		TTL:          time.Duration(cfg.HostCookie.TTL) * 24 * time.Hour,
	}

	userSyncDeps := &pbs.UserSyncDeps{
		HostCookieSettings: &hostCookieSettings,
		ExternalUrl:        cfg.ExternalURL,
		RecaptchaSecret:    cfg.RecaptchaSecret,
		Metrics:            metricsRegistry,
	}

	router.GET("/getuids", userSyncDeps.GetUIDs)
	router.GET("/setuid", userSyncDeps.SetUID)
	router.POST("/optout", userSyncDeps.OptOut)
	router.GET("/optout", userSyncDeps.OptOut)

	pbc.InitPrebidCache(cfg.CacheURL.GetBaseURL())

	// Add CORS middleware
	c := cors.New(cors.Options{
		AllowCredentials: true,
		AllowedHeaders:   []string{"Origin", "X-Requested-With", "Content-Type", "Accept"}})
	corsRouter := c.Handler(router)

	// Add no cache headers
	noCacheHandler := NoCache{corsRouter}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      noCacheHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	go (func() {
		fmt.Printf("Main server running on: %s\n", server.Addr)
		serverErr := server.ListenAndServe()
		glog.Errorf("Main server: %v", serverErr)
		stopSignals <- syscall.SIGTERM
	})()

	<-stopSignals

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		glog.Errorf("Main server shutdown: %v", err)
	}
	if err := adminServer.Shutdown(ctx); err != nil {
		glog.Errorf("Admin server shutdown: %v", err)
	}

	return nil
}

const requestConfigPath = "./stored_requests/data/by_id"

// NewFetchers returns an Account-based config fetcher and a Request-based config fetcher, in that order.
// If it can't generate both of those from the given config, then an error will be returned.
//
// This function assumes that the argument config has been validated.
func NewFetchers(cfg *config.StoredRequests, db *sql.DB) (byId stored_requests.Fetcher, byAmpId stored_requests.Fetcher, err error) {
	if cfg.Files {
		glog.Infof("Loading Stored Requests from filesystem at path %s", requestConfigPath)
		byId, err = file_fetcher.NewFileFetcher(requestConfigPath)
		// Currently assuming the file store is "flat", that is IDs are unique across all config types
		// and that the files for all the types sit next to each other.
		byAmpId = byId
	} else if cfg.Postgres != nil {
		// Be careful not to log the password here, for security reasons
		glog.Infof("Loading Stored Requests from Postgres. DB=%s, host=%s, port=%d, user=%s, query=%s", cfg.Postgres.Database, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.Username, cfg.Postgres.QueryTemplate)
		byId = db_fetcher.NewFetcher(db, cfg.Postgres.MakeQuery)
		byAmpId = db_fetcher.NewFetcher(db, cfg.Postgres.MakeAmpQuery)
	} else {
		glog.Warning("No Stored Request support configured. request.imp[i].ext.prebid.storedrequest will be ignored. If you need this, check your app config")
		byId = empty_fetcher.EmptyFetcher()
		byAmpId = byId
	}

	if cfg.InMemoryCache != nil {
		glog.Infof("Using a Stored Request in-memory cache. Max size: %d bytes. TTL: %d seconds.", cfg.InMemoryCache.Size, cfg.InMemoryCache.TTL)
		byId = stored_requests.WithCache(byId, in_memory.NewLRUCache(cfg.InMemoryCache))
		byAmpId = stored_requests.WithCache(byAmpId, in_memory.NewLRUCache(cfg.InMemoryCache))
	}
	return
}
