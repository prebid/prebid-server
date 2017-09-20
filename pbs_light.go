package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sort"
	"sync"
	"time"

	"github.com/cloudfoundry/gosigar"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/cors"
	"github.com/spf13/viper"
	"github.com/vrischmann/go-metrics-influxdb"
	"github.com/xeipuuv/gojsonschema"
	"github.com/xojoc/useragent"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/cache/filecache"
	"github.com/prebid/prebid-server/cache/postgrescache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/prebid"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
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

	requireUUID2 bool
	cookieDomain string
)

var exchanges map[string]adapters.Adapter
var dataCache cache.Cache
var reqSchema *gojsonschema.Schema

type BidCache struct {
	Adm    string `json:"adm,omitempty"`
	NURL   string `json:"nurl,omitempty"`
	Width  uint64 `json:"width,omitempty"`
	Height uint64 `json:"height,omitempty"`
}

type bidResult struct {
	bidder   *pbs.PBSBidder
	bid_list pbs.PBSBidSlice
}

const defaultPriceGranularity = "med"

// Constant keys for ad server targeting for responses to Prebid Mobile
const hbpbConstantKey = "hb_pb"
const hbCreativeLoadMethodConstantKey = "hb_creative_loadtype"
const hbBidderConstantKey = "hb_bidder"
const hbCacheIdConstantKey = "hb_cache_id"

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
	UUID    string   `json:"uuid"`
	Bidders []string `json:"bidders"`
}

type cookieSyncResponse struct {
	UUID         string           `json:"uuid"`
	Status       string           `json:"status"`
	BidderStatus []*pbs.PBSBidder `json:"bidder_status"`
}

func cookieSync(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	mCookieSyncMeter.Mark(1)
	userSyncCookie := pbs.ParsePBSCookieFromRequest(r)
	if !userSyncCookie.AllowSyncs() {
		http.Error(w, "User has opted out", http.StatusUnauthorized)
		return
	}

	defer r.Body.Close()

	csReq := &cookieSyncRequest{}
	err := json.NewDecoder(r.Body).Decode(&csReq)
	if err != nil {
		if glog.V(2) {
			glog.Infof("Failed to parse /cookie_sync request body: %v", err)
		}
		http.Error(w, "JSON parse failed", http.StatusBadRequest)
		return
	}

	csResp := cookieSyncResponse{
		UUID:         csReq.UUID,
		BidderStatus: make([]*pbs.PBSBidder, 0, len(csReq.Bidders)),
	}
	if _, err := r.Cookie("uuid2"); (requireUUID2 && err != nil) || userSyncCookie.LiveSyncCount() == 0 {
		csResp.Status = "no_cookie"
	} else {
		csResp.Status = "ok"
	}

	for _, bidder := range csReq.Bidders {
		if ex, ok := exchanges[bidder]; ok {
			if !userSyncCookie.HasLiveSync(ex.FamilyName()) {
				b := pbs.PBSBidder{
					BidderCode:   bidder,
					NoCookie:     true,
					UsersyncInfo: ex.GetUsersyncInfo(),
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

func auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Add("Content-Type", "application/json")

	mRequestMeter.Mark(1)

	isSafari := false
	if ua := useragent.Parse(r.Header.Get("User-Agent")); ua != nil {
		if ua.Type == useragent.Browser && ua.Name == "Safari" {
			isSafari = true
			mSafariRequestMeter.Mark(1)
		}
	}

	pbs_req, err := pbs.ParsePBSRequest(r, dataCache)
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
		if requireUUID2 {
			uuid2 := fmt.Sprintf("%d", rand.Int63())
			c := http.Cookie{
				Name:    "uuid2",
				Value:   uuid2,
				Domain:  cookieDomain,
				Expires: time.Now().Add(180 * 24 * time.Hour),
			}
			http.SetCookie(w, &c)
			pbs_req.Cookie.TrySync("adnxs", uuid2)
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
				uid, _, _ := pbs_req.Cookie.GetUID(ex.FamilyName())
				if uid == "" {
					bidder.NoCookie = true
					bidder.UsersyncInfo = ex.GetUsersyncInfo()
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
			bc := BidCache{
				Adm:    bid.Adm,
				NURL:   bid.NURL,
				Width:  bid.Width,
				Height: bid.Height,
			}
			buf := new(bytes.Buffer)
			enc := json.NewEncoder(buf)
			enc.SetEscapeHTML(false)
			enc.Encode(bc)
			cobjs[i] = &pbc.CacheObject{
				Value: buf.String(),
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
			bid.NURL = ""
			bid.Adm = ""
		}
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
			priceBucketStringMap := pbs.GetPriceBucketString(bid.Price)
			roundedCpm := priceBucketStringMap[priceGranularitySetting]

			hbPbBidderKey := hbpbConstantKey + "_" + bid.BidderCode
			hbBidderBidderKey := hbBidderConstantKey + "_" + bid.BidderCode
			hbCacheIdBidderKey := hbCacheIdConstantKey + "_" + bid.BidderCode
			if pbs_req.MaxKeyLength != 0 {
				hbPbBidderKey = hbPbBidderKey[:min(len(hbPbBidderKey), int(pbs_req.MaxKeyLength))]
				hbBidderBidderKey = hbBidderBidderKey[:min(len(hbBidderBidderKey), int(pbs_req.MaxKeyLength))]
				hbCacheIdBidderKey = hbCacheIdBidderKey[:min(len(hbCacheIdBidderKey), int(pbs_req.MaxKeyLength))]
			}
			pbs_kvs := map[string]string{
				hbPbBidderKey:      roundedCpm,
				hbBidderBidderKey:  bid.BidderCode,
				hbCacheIdBidderKey: bid.CacheID,
			}
			// For the top bid, we want to add the following additional keys
			if i == 0 {
				pbs_kvs[hbpbConstantKey] = roundedCpm
				pbs_kvs[hbBidderConstantKey] = bid.BidderCode
				pbs_kvs[hbCacheIdConstantKey] = bid.CacheID
				if bid.BidderCode == "audienceNetwork" {
					pbs_kvs[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodDemandSDK
				} else {
					pbs_kvs[hbCreativeLoadMethodConstantKey] = hbCreativeLoadMethodHTML
				}
			}
			bid.AdServerTargeting = pbs_kvs
		}
	}
}

func status(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// could add more logic here, but doing nothing means 200 OK
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

// https://blog.golang.org/context/userip/userip.go
func getIP(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	if ua := useragent.Parse(req.Header.Get("User-Agent")); ua != nil {
		fmt.Fprintf(w, "User Agent: %v\n", ua)
	}
	ip, port, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		fmt.Fprintf(w, "userip: %q is not IP:port\n", req.RemoteAddr)
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		//return nil, fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)
		fmt.Fprintf(w, "userip: %q is not IP:port\n", req.RemoteAddr)
		return
	}

	forwardedIP := prebid.GetForwardedIP(req)
	realIP := prebid.GetIP(req)

	fmt.Fprintf(w, "IP: %s\n", ip)
	fmt.Fprintf(w, "Port: %s\n", port)
	fmt.Fprintf(w, "Forwarded IP: %s\n", forwardedIP)
	fmt.Fprintf(w, "Real IP: %s\n", realIP)

	for k, v := range req.Header {
		fmt.Fprintf(w, "%s: %s\n", k, v)
	}

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

func loadPostgresDataCache(cfg *config.Configuration) (cache.Cache, error) {
	mem := sigar.Mem{}
	mem.Get()

	return postgrescache.New(postgrescache.PostgresConfig{
		Dbname:   cfg.DataCache.Database,
		Host:     cfg.DataCache.Host,
		User:     cfg.DataCache.Username,
		Password: cfg.DataCache.Password,
		Size:     cfg.DataCache.CacheSize,
		TTL:      cfg.DataCache.TTLSeconds,
	})

}

func loadDataCache(cfg *config.Configuration) (err error) {

	switch cfg.DataCache.Type {
	case "dummy":
		dataCache, err = dummycache.New()
		if err != nil {
			glog.Fatalf("Dummy cache not configured: %s", err.Error())
		}

	case "postgres":
		dataCache, err = loadPostgresDataCache(cfg)
		if err != nil {
			return fmt.Errorf("PostgresCache Error: %s", err.Error())
		}

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
	viper.SetDefault("datacache.type", "dummy")
	// no metrics configured by default (metrics{host|database|username|password})

	viper.SetDefault("adapters.pubmatic.endpoint", "http://openbid.pubmatic.com/translator?source=prebid-server")
	viper.SetDefault("adapters.rubicon.endpoint", "http://staged-by.rubiconproject.com/a/api/exchange.json")
	viper.SetDefault("adapters.rubicon.usersync_url", "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid")
	viper.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	viper.SetDefault("adapters.index.usersync_url", "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=https%3A%2F%2Fprebid.adnxs.com%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3DindexExchange%26uid%3D")
	viper.ReadInConfig()

	flag.Parse() // read glog settings from cmd line
}

func main() {
	cfg, err := config.New()
	if err != nil {
		glog.Errorf("Viper was unable to read configurations: %v", err)
	}
	// we need to set this global variable so it can be used by other methods
	requireUUID2 = cfg.RequireUUID2
	cookieDomain = cfg.CookieDomain
	if err := serve(cfg); err != nil {
		glog.Fatalf("PreBid Server encountered an error: %v", err)
	}
}

func setupExchanges(cfg *config.Configuration) {
	exchanges = map[string]adapters.Adapter{
		"appnexus":      adapters.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
		"districtm":     adapters.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
		"indexExchange": adapters.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["indexexchange"].Endpoint, cfg.Adapters["indexexchange"].UserSyncURL),
		"pubmatic":      adapters.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pubmatic"].Endpoint, cfg.ExternalURL),
		"pulsepoint":    adapters.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pulsepoint"].Endpoint, cfg.ExternalURL),
		"rubicon": adapters.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["rubicon"].Endpoint,
			cfg.Adapters["rubicon"].XAPI.Username, cfg.Adapters["rubicon"].XAPI.Password, cfg.Adapters["rubicon"].XAPI.Tracker, cfg.Adapters["rubicon"].UserSyncURL),
		"audienceNetwork": adapters.NewFacebookAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["facebook"].PlatformID, cfg.Adapters["facebook"].UserSyncURL),
	}

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
	if err := loadDataCache(cfg); err != nil {
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

	/* Run admin on different port thats not exposed */
	go func() {
		// Todo -- make configurable
		adminURI := fmt.Sprintf("%s:%d", cfg.Host, cfg.AdminPort)
		fmt.Println("Admin running on: ", adminURI)
		glog.Fatal(http.ListenAndServe(adminURI, nil))
	}()

	router := httprouter.New()
	router.POST("/auction", auction)
	router.POST("/cookie_sync", cookieSync)
	router.POST("/validate", validate)
	router.GET("/status", status)
	router.GET("/", serveIndex)
	router.GET("/ip", getIP)
	router.ServeFiles("/static/*filepath", http.Dir("static"))

	userSyncDeps := &pbs.UserSyncDeps{
		Cookie_domain:    cfg.CookieDomain,
		External_url:     cfg.ExternalURL,
		Recaptcha_secret: cfg.RecaptchaSecret,
		Metrics:          metricsRegistry,
	}

	router.GET("/getuids", userSyncDeps.GetUIDs)
	router.GET("/setuid", userSyncDeps.SetUID)
	router.POST("/optout", userSyncDeps.OptOut)
	router.GET("/optout", userSyncDeps.OptOut)

	pbc.InitPrebidCache(cfg.CacheURL)

	// Add CORS middleware
	c := cors.New(cors.Options{AllowCredentials: true})
	corsRouter := c.Handler(router)

	// Add no cache headers
	noCacheHandler := NoCache{corsRouter}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      noCacheHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	fmt.Printf("Server running on: %s\n", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		return err
	}
	return nil
}
