package endpoints

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"sort"
	"strconv"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/mssola/user_agent"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/pbsmetrics"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/usersync"
)

type bidResult struct {
	bidder  *pbs.PBSBidder
	bidList pbs.PBSBidSlice
}

const defaultPriceGranularity = "med"

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

type auction struct {
	cfg           *config.Configuration
	syncers       map[openrtb_ext.BidderName]usersync.Usersyncer
	gdprPerms     gdpr.Permissions
	metricsEngine pbsmetrics.MetricsEngine
	dataCache     cache.Cache
	exchanges     map[string]adapters.Adapter
}

func Auction(cfg *config.Configuration, syncers map[openrtb_ext.BidderName]usersync.Usersyncer, gdprPerms gdpr.Permissions, metricsEngine pbsmetrics.MetricsEngine, dataCache cache.Cache, exchanges map[string]adapters.Adapter) httprouter.Handle {
	a := &auction{
		cfg:           cfg,
		syncers:       syncers,
		gdprPerms:     gdprPerms,
		metricsEngine: metricsEngine,
		dataCache:     dataCache,
		exchanges:     exchanges,
	}
	return a.auction
}

func (a *auction) auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
	req, err := pbs.ParsePBSRequest(r, &a.cfg.AuctionTimeouts, a.dataCache, &(a.cfg.HostCookie))
	// Defer here because we need req defined.
	defer func() {
		if req == nil {
			a.metricsEngine.RecordRequest(labels)
			a.metricsEngine.RecordLegacyImps(labels, 0)
		} else {
			// handles the case that ParsePBSRequest returns an error, so req.Start is not defined
			a.metricsEngine.RecordRequest(labels)
			a.metricsEngine.RecordLegacyImps(labels, len(req.AdUnits))
			a.metricsEngine.RecordRequestTime(labels, time.Since(req.Start))
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
	if req.App != nil {
		labels.Source = pbsmetrics.DemandApp
	} else {
		labels.Source = pbsmetrics.DemandWeb
		if req.Cookie.LiveSyncCount() == 0 {
			labels.CookieFlag = pbsmetrics.CookieFlagNo
			status = "no_cookie"
		} else {
			labels.CookieFlag = pbsmetrics.CookieFlagYes
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(req.TimeoutMillis))
	defer cancel()
	account, err := a.dataCache.Accounts().Get(req.AccountID)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Invalid account id: %v", err)
		}
		writeAuctionError(w, "Unknown account id", fmt.Errorf("Unknown account"))
		labels.RequestStatus = pbsmetrics.RequestStatusBadInput
		return
	}
	labels.PubID = req.AccountID
	resp := pbs.PBSResponse{
		Status:       status,
		TID:          req.Tid,
		BidderStatus: req.Bidders,
	}
	s.AdapterBidPresent,
			}
			if blabels.Adapter == "" {
				// "districtm" is legal, but not in BidderMap. Other values will log errors in the go_metrics code
				blabels.Adapter = openrtb_ext.BidderName(bidder.BidderCode)
			}
			if req.App == nil {
				// If exchanges[bidderCode] exists, then a.syncers[bidderCode] exists *except for districtm*.
				// OpenRTB handles aliases differently, so this hack will keep legacy code working. For all other
				// bidderCodes, a.syncers[bidderCode] will exist if exchanges[bidderCode] also does.
				// This is guaranteed by the following unit tests, which compare these maps to the (source of truth) openrtb_ext.BidderMap:
				//   1. TestSyncers inside usersync/usersync_test.go
				//   2. TestExchangeMap inside pbs_light_test.go
				syncerCode := bidder.BidderCode
				if syncerCode == "districtm" {
					syncerCode = "appnexus"
				}
				syncer := a.syncers[openrtb_ext.BidderName(syncerCode)]
				uid, _, _ := req.Cookie.GetUID(syncer.FamilyName())
				if uid == "" {
					bidder.NoCookie = true
					gdprApplies := req.ParseGDPR()
					consent := req.ParseConsent()
					if a.shouldUsersync(ctx, openrtb_ext.BidderName(syncerCode), gdprApplies, consent) {
						syncInfo, err := syncer.GetUsersyncInfo(gdprApplies, consent)
						if err == nil {
							bidder.UsersyncInfo = syncInfo
						} else {
							glog.Errorf("Failed to get usersync info for %s: %v", syncerCode, err)
						}
					}
					blabels.CookieFlag = pbsmetrics.CookieFlagNo
					if ex.SkipNoCookies() {
						continue
					}
				}
			}
			sentBids++
			bidderRunner := a.recoverSafely(func(bidder *pbs.PBSBidder, aLabels pbsmetrics.AdapterLabels) {

				start := time.Now()
				bidList, err := ex.Call(ctx, req, bidder)
				a.metricsEngine.RecordAdapterTime(aLabels, time.Since(start))
				bidder.ResponseTime = int(time.Since(start) / time.Millisecond)
				if err != nil {
					var s struct{}
					switch err {
					case context.DeadlineExceeded:
						aLabels.AdapterErrors = map[pbsmetrics.AdapterError]struct{}{pbsmetrics.AdapterErrorTimeout: s}
						bidder.Error = "Timed out"
					case context.Canceled:
						fallthrough
					default:
						bidder.Error = err.Error()
						switch err.(type) {
						case *errortypes.BadInput:
							aLabels.AdapterErrors = map[pbsmetrics.AdapterError]struct{}{pbsmetrics.AdapterErrorBadInput: s}
						case *errortypes.BadServerResponse:
							aLabels.AdapterErrors = map[pbsmetrics.AdapterError]struct{}{pbsmetrics.AdapterErrorBadServerResponse: s}
						default:
							glog.Warningf("Error from bidder %v. Ignoring all bids: %v", bidder.BidderCode, err)
							aLabels.AdapterErrors = map[pbsmetrics.AdapterError]struct{}{pbsmetrics.AdapterErrorUnknown: s}
						}
					}
				} else if bidList != nil {
					bidList = checkForValidBidSize(bidList, bidder)
					bidder.NumBids = len(bidList)
					for _, bid := range bidList {
						var cpm = float64(bid.Price * 1000)
						a.metricsEngine.RecordAdapterPrice(aLabels, cpm)
						switch bid.CreativeMediaType {
						case "banner":
							a.metricsEngine.RecordAdapterBidReceived(aLabels, openrtb_ext.BidTypeBanner, bid.Adm != "")
						case "video":
							a.metricsEngine.RecordAdapterBidReceived(aLabels, openrtb_ext.BidTypeVideo, bid.Adm != "")
						}
						bid.ResponseTime = bidder.ResponseTime
					}
				} else {
					bidder.NoBid = true
					aLabels.AdapterBids = pbsmetrics.AdapterBidNone
				}

				ch <- bidResult{
					bidder:  bidder,
					bidList: bidList,
					// Bidder done, record bidder metrics
				}
				a.metricsEngine.RecordAdapterRequest(aLabels)
			})

			go bidderRunner(bidder, blabels)

		} else {
			bidder.Error = "Unsupported bidder"
		}
	}
	for i := 0; i < sentBids; i++ {
		result := <-ch
		for _, bid := range result.bidList {
			resp.Bids = append(resp.Bids, bid)
		}
	}
	if req.CacheMarkup == 1 {
		cobjs := make([]*pbc.CacheObject, len(resp.Bids))
		for i, bid := range resp.Bids {
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
		if err := pbc.Put(ctx, cobjs); err != nil {
			writeAuctionError(w, "Prebid cache failed", err)
			labels.RequestStatus = pbsmetrics.RequestStatusErr
			return
		}
		for i, bid := range resp.Bids {
			bid.CacheID = cobjs[i].UUID
			bid.CacheURL = a.cfg.GetCachedAssetURL(bid.CacheID)
			bid.NURL = ""
			bid.Adm = ""
		}
	}
	if req.CacheMarkup == 2 {
		cacheVideoOnly(resp.Bids, ctx, w, a, &labels)
	}
	if req.SortBids == 1 {
		sortBidsAddKeywordsMobile(resp.Bids, req, account.PriceGranularity)
	}
	if glog.V(2) {
		glog.Infof("Request for %d ad units on url %s by account %s got %d bids", len(req.AdUnits), req.Url, req.AccountID, len(resp.Bids))
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(resp)
}

func (a *auction) recoverSafely(inner func(*pbs.PBSBidder, pbsmetrics.AdapterLabels)) func(*pbs.PBSBidder, pbsmetrics.AdapterLabels) {
	return func(bidder *pbs.PBSBidder, labels pbsmetrics.AdapterLabels) {
		defer func() {
			if r := recover(); r != nil {
				if bidder == nil {
					glog.Errorf("Legacy auction recovered panic: %v. Stack trace is: %v", r, string(debug.Stack()))
				} else {
					glog.Errorf("Legacy auction recovered panic from Bidder %s: %v. Stack trace is: %v", bidder.BidderCode, r, string(debug.Stack()))
				}
				a.metricsEngine.RecordAdapterPanic(labels)
			}
		}()
		inner(bidder, labels)
	}
}

func (a *auction) shouldUsersync(ctx context.Context, bidder openrtb_ext.BidderName, gdprApplies string, consent string) bool {
	switch gdprApplies {
	case "0":
		return true
	case "1":
		if consent == "" {
			return false
		}
		fallthrough
	default:
		if canSync, err := a.gdprPerms.HostCookiesAllowed(ctx, consent); !canSync || err != nil {
			return false
		}
		canSync, err := a.gdprPerms.BidderSyncAllowed(ctx, bidder, consent)
		return canSync && err == nil
	}
}

// cache video bids only for Web
func cacheVideoOnly(bids pbs.PBSBidSlice, ctx context.Context, w http.ResponseWriter, deps *auction, labels *pbsmetrics.Labels) {
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

			hbPbBidderKey := string(openrtb_ext.HbpbConstantKey) + "_" + bid.BidderCode
			hbBidderBidderKey := string(openrtb_ext.HbBidderConstantKey) + "_" + bid.BidderCode
			hbCacheIDBidderKey := string(openrtb_ext.HbCacheKey) + "_" + bid.BidderCode
			hbDealIDBidderKey := string(openrtb_ext.HbDealIDConstantKey) + "_" + bid.BidderCode
			hbSizeBidderKey := string(openrtb_ext.HbSizeConstantKey) + "_" + bid.BidderCode
			if pbs_req.MaxKeyLength != 0 {
				hbPbBidderKey = hbPbBidderKey[:min(len(hbPbBidderKey), int(pbs_req.MaxKeyLength))]
				hbBidderBidderKey = hbBidderBidderKey[:min(len(hbBidderBidderKey), int(pbs_req.MaxKeyLength))]
				hbCacheIDBidderKey = hbCacheIDBidderKey[:min(len(hbCacheIDBidderKey), int(pbs_req.MaxKeyLength))]
				hbDealIDBidderKey = hbDealIDBidderKey[:min(len(hbDealIDBidderKey), int(pbs_req.MaxKeyLength))]
				hbSizeBidderKey = hbSizeBidderKey[:min(len(hbSizeBidderKey), int(pbs_req.MaxKeyLength))]
			}

			// fixes #288 where map was being overwritten instead of updated
			if bid.AdServerTargeting == nil {
				bid.AdServerTargeting = make(map[string]string)
			}
			kvs := bid.AdServerTargeting

			kvs[hbPbBidderKey] = roundedCpm
			kvs[hbBidderBidderKey] = bid.BidderCode
			kvs[hbCacheIDBidderKey] = bid.CacheID

			if hbSize != "" {
				kvs[hbSizeBidderKey] = hbSize
			}
			if bid.DealId != "" {
				kvs[hbDealIDBidderKey] = bid.DealId
			}
			// For the top bid, we want to add the following additional keys
			if i == 0 {
				kvs[string(openrtb_ext.HbpbConstantKey)] = roundedCpm
				kvs[string(openrtb_ext.HbBidderConstantKey)] = bid.BidderCode
				kvs[string(openrtb_ext.HbCacheKey)] = bid.CacheID
				if bid.DealId != "" {
					kvs[string(openrtb_ext.HbDealIDConstantKey)] = bid.DealId
				}
				if hbSize != "" {
					kvs[string(openrtb_ext.HbSizeConstantKey)] = hbSize
				}
			}
		}
	}
}
