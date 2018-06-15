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
	"time"

	"github.com/cloudfoundry/gosigar"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"github.com/rs/cors"
	"github.com/spf13/viper"
	"github.com/xeipuuv/gojsonschema"

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
	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/cache/filecache"
	"github.com/prebid/prebid-server/cache/postgrescache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/endpoints"
	infoEndpoints "github.com/prebid/prebid-server/endpoints/info"
	"github.com/prebid/prebid-server/endpoints/openrtb2"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbsmetrics"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/server"
	"github.com/prebid/prebid-server/ssl"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/usersync/usersyncers"

	storedRequestsConf "github.com/prebid/prebid-server/stored_requests/config"
)

var hostCookieSettings pbs.HostCookieSettings

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

type auctionDeps struct {
	cfg           *config.Configuration
	syncers       map[openrtb_ext.BidderName]usersync.Usersyncer
	gdprPerms     gdpr.Permissions
	metricsEngine pbsmetrics.MetricsEngine
}

func (deps *auctionDeps) auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Add("Content-Type", "application/json")

	labels := pbsmetrics.Labels{
		Source:        pbsmetrics.DemandUnknown,
		RType:         pbsmetrics.ReqTypeLegacy,
		PubID:         "",
		Browser:       pbsmetrics.BrowserOther,
		CookieFlag:    pbsmetrics.CookieFlagUnknown,
		RequestStatus: pbsmetrics.RequestStatusOK,
	}

	if ua := user_agent.New(r.Header.Get("User-Agent")); ua != nil {
		name, _ := ua.Browser()
		if name == "Safari" {
			labels.Browser = pbsmetrics.BrowserSafari
		}
	}

	pbs_req, err := pbs.ParsePBSRequest(r, &deps.cfg.AuctionTimeouts, dataCache, &hostCookieSettings)
	// Defer here because we need pbs_req defined.
	defer func() {
		if pbs_req == nil {
			deps.metricsEngine.RecordRequest(labels)
			deps.metricsEngine.RecordImps(labels, 0)
		} else {
			// handles the case that ParsePBSRequest returns an error, so pbs_req.Start is not defined
			deps.metricsEngine.RecordRequest(labels)
			deps.metricsEngine.RecordImps(labels, len(pbs_req.AdUnits))
			deps.metricsEngine.RecordRequestTime(labels, time.Since(pbs_req.Start))
		}
	}()

	if err != nil {
		if glog.V(2) {
			glog.Infof("Failed to parse /auction request: %v", err)
		}
		writeAuctionError(w, "Error parsing request", err)
		labels.RequestStatus = pbsmetrics.RequestStatusBadInput
		return
	}

	status := "OK"
	if pbs_req.App != nil {
		labels.Source = pbsmetrics.DemandApp
	} else {
		labels.Source = pbsmetrics.DemandWeb
		if pbs_req.Cookie.LiveSyncCount() == 0 {
			labels.CookieFlag = pbsmetrics.CookieFlagNo
			status = "no_cookie"
		} else {
			labels.CookieFlag = pbsmetrics.CookieFlagYes
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pbs_req.TimeoutMillis))
	defer cancel()

	account, err := dataCache.Accounts().Get(pbs_req.AccountID)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Invalid account id: %v", err)
		}
		writeAuctionError(w, "Unknown account id", fmt.Errorf("Unknown account"))
		labels.RequestStatus = pbsmetrics.RequestStatusBadInput
		return
	}
	labels.PubID = pbs_req.AccountID

	pbs_resp := pbs.PBSResponse{
		Status:       status,
		TID:          pbs_req.Tid,
		BidderStatus: pbs_req.Bidders,
	}

	ch := make(chan bidResult)
	sentBids := 0
	for _, bidder := range pbs_req.Bidders {
		if ex, ok := exchanges[bidder.BidderCode]; ok {
			// Make sure we have an independent label struct for each bidder. We don't want to run into issues with the goroutine below.
			blabels := pbsmetrics.AdapterLabels{
				Source:        labels.Source,
				RType:         labels.RType,
				Adapter:       openrtb_ext.BidderMap[bidder.BidderCode],
				PubID:         labels.PubID,
				Browser:       labels.Browser,
				CookieFlag:    labels.CookieFlag,
				AdapterStatus: pbsmetrics.AdapterStatusOK,
			}
			if blabels.Adapter == "" {
				// "districtm" is legal, but not in BidderMap. Other values will log errors in the go_metrics code
				blabels.Adapter = openrtb_ext.BidderName(bidder.BidderCode)
			}
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
					gdprApplies := pbs_req.ParseGDPR()
					consent := pbs_req.ParseConsent()
					if deps.shouldUsersync(ctx, openrtb_ext.BidderName(syncerCode), gdprApplies, consent) {
						bidder.UsersyncInfo = syncer.GetUsersyncInfo(gdprApplies, consent)
					}
					blabels.CookieFlag = pbsmetrics.CookieFlagNo
					if ex.SkipNoCookies() {
						continue
					}
				}
			}
			sentBids++
			go func(bidder *pbs.PBSBidder, blables pbsmetrics.AdapterLabels) {
				start := time.Now()
				bid_list, err := ex.Call(ctx, pbs_req, bidder)
				deps.metricsEngine.RecordAdapterTime(blabels, time.Since(start))
				bidder.ResponseTime = int(time.Since(start) / time.Millisecond)
				if err != nil {
					switch err {
					case context.DeadlineExceeded:
						blabels.AdapterStatus = pbsmetrics.AdapterStatusTimeout
						bidder.Error = "Timed out"
					case context.Canceled:
						fallthrough
					default:
						blabels.AdapterStatus = pbsmetrics.AdapterStatusErr
						bidder.Error = err.Error()
						if _, isBadInput := err.(*adapters.BadInputError); !isBadInput {
							if _, isBadServer := err.(*adapters.BadServerResponseError); !isBadServer {
								glog.Warningf("Error from bidder %v. Ignoring all bids: %v", bidder.BidderCode, err)
							}
						}
					}
				} else if bid_list != nil {
					bid_list = checkForValidBidSize(bid_list, bidder)
					bidder.NumBids = len(bid_list)
					for _, bid := range bid_list {
						var cpm = float64(bid.Price * 1000)
						deps.metricsEngine.RecordAdapterPrice(blables, cpm)
						switch bid.CreativeMediaType {
						case "banner":
							deps.metricsEngine.RecordAdapterBidReceived(blabels, openrtb_ext.BidTypeBanner, bid.Adm != "")
						case "video":
							deps.metricsEngine.RecordAdapterBidReceived(blabels, openrtb_ext.BidTypeVideo, bid.Adm != "")
						}
						bid.ResponseTime = bidder.ResponseTime
					}
				} else {
					bidder.NoBid = true
					blabels.AdapterStatus = pbsmetrics.AdapterStatusNoBid
				}

				ch <- bidResult{
					bidder:   bidder,
					bid_list: bid_list,
					// Bidder done, record bidder metrics
				}
				deps.metricsEngine.RecordAdapterRequest(blabels)
			}(bidder, blabels)

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
			labels.RequestStatus = pbsmetrics.RequestStatusErr
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
		cacheVideoOnly(pbs_resp.Bids, ctx, w, deps, &labels)
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
}

func (deps *auctionDeps) shouldUsersync(ctx context.Context, bidder openrtb_ext.BidderName, gdprApplies string, consent string) bool {
	switch gdprApplies {
	case "0":
		return true
	case "1":
		if consent == "" {
			return false
		}
		fallthrough
	default:
		if canSync, err := deps.gdprPerms.HostCookiesAllowed(ctx, consent); !canSync || err != nil {
			return false
		}
		canSync, err := deps.gdprPerms.BidderSyncAllowed(ctx, bidder, consent)
		return canSync && err == nil
	}
}

// cache video bids only for Web
func cacheVideoOnly(bids pbs.PBSBidSlice, ctx context.Context, w http.ResponseWriter, deps *auctionDeps, labels *pbsmetrics.Labels) {
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
		labels.RequestStatus = pbsmetrics.RequestStatusErr
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
func sortBidsAddKeywordsMobile(bids pbs.PBSBidSlice, pbs_req *pbs.PBSRequest, priceGranularitySetting string) {
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
			roundedCpm, err := exchange.GetCpmStringValue(bid.Price, openrtb_ext.PriceGranularityFromString(priceGranularitySetting))
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

	flag.Parse() // read glog settings from cmd line
}

func main() {
	v := viper.New()
	config.SetupViper(v)
	cfg, err := config.New(v)
	if err != nil {
		glog.Fatalf("Configuration could not be loaded or did not pass validation: %v", err)
	}

	if err := serve(cfg); err != nil {
		glog.Errorf("prebid-server failed: %v", err)
	}
}

func newExchangeMap(cfg *config.Configuration) map[string]adapters.Adapter {
	// These keys _must_ coincide with the bidder code in Prebid.js, if the adapter exists in both projects
	return map[string]adapters.Adapter{
		"appnexus":      appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["appnexus"].Endpoint),
		"districtm":     appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["appnexus"].Endpoint),
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

func serve(cfg *config.Configuration) error {
	router := httprouter.New()
	theClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        400,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     60 * time.Second,
			TLSClientConfig:     &tls.Config{RootCAs: ssl.GetRootCAPool()},
		},
	}
	fetcher, ampFetcher, db, shutdown := storedRequestsConf.NewStoredRequests(&cfg.StoredRequests, theClient, router)
	defer shutdown()

	if err := loadDataCache(cfg, db); err != nil {
		return fmt.Errorf("Prebid Server could not load data cache: %v", err)
	}

	pbsAnalytics := analyticsConf.NewPBSAnalytics(&cfg.Analytics)

	// Hack because of how legacy handles districtm
	bidderList := openrtb_ext.BidderList()
	bidderList = append(bidderList, openrtb_ext.BidderName("districtm"))

	metricsEngine := config.NewMetricsEngine(cfg, bidderList)

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

	paramsValidator, err := openrtb_ext.NewBidderParamsValidator(schemaDirectory)
	if err != nil {
		glog.Fatalf("Failed to create the bidder params validator. %v", err)
	}

	exchanges = newExchangeMap(cfg)
	theExchange := exchange.NewExchange(theClient, pbc.NewClient(&cfg.CacheURL), cfg, metricsEngine)

	openrtbEndpoint, err := openrtb2.NewEndpoint(theExchange, paramsValidator, fetcher, cfg, metricsEngine, pbsAnalytics)
	if err != nil {
		glog.Fatalf("Failed to create the openrtb endpoint handler. %v", err)
	}

	ampEndpoint, err := openrtb2.NewAmpEndpoint(theExchange, paramsValidator, ampFetcher, cfg, metricsEngine, pbsAnalytics)
	if err != nil {
		glog.Fatalf("Failed to create the amp endpoint handler. %v", err)
	}

	syncers := usersyncers.NewSyncerMap(cfg)
	gdprPerms := gdpr.NewPermissions(context.Background(), cfg.GDPR, usersyncers.GDPRAwareSyncerIDs(syncers), theClient)

	router.POST("/auction", (&auctionDeps{cfg, syncers, gdprPerms, metricsEngine}).auction)
	router.POST("/openrtb2/auction", openrtbEndpoint)
	router.GET("/openrtb2/amp", ampEndpoint)
	router.GET("/info/bidders", infoEndpoints.NewBiddersEndpoint())
	router.GET("/info/bidders/:bidderName", infoEndpoints.NewBidderDetailsEndpoint("./static/bidder-info", openrtb_ext.BidderList()))
	router.GET("/bidders/params", NewJsonDirectoryServer(paramsValidator))
	router.POST("/cookie_sync", endpoints.NewCookieSyncEndpoint(syncers, &(hostCookieSettings.OptOutCookie), gdprPerms, metricsEngine, pbsAnalytics))
	router.POST("/validate", validate)
	router.GET("/status", endpoints.NewStatusEndpoint(cfg.StatusResponse))
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
		MetricsEngine:      metricsEngine,
		PBSAnalytics:       pbsAnalytics,
	}

	router.GET("/getuids", userSyncDeps.GetUIDs)
	router.GET("/setuid", endpoints.NewSetUIDEndpoint(cfg.HostCookie, gdprPerms, pbsAnalytics, metricsEngine))
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

	server.Listen(cfg, noCacheHandler, metricsEngine)
	return nil
}
