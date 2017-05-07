package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/pbs"

	"github.com/cloudfoundry/gosigar"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/rs/cors"
	"github.com/spf13/viper"
	"github.com/vrischmann/go-metrics-influxdb"
	"github.com/xeipuuv/gojsonschema"
	"github.com/xojoc/useragent"
)

type DomainMetrics struct {
	RequestMeter metrics.Meter
}

type AccountMetrics struct {
	RequestMeter      metrics.Meter
	BidsReceivedMeter metrics.Meter
	PriceHistogram    metrics.Histogram
}

type AdapterMetrics struct {
	NoCookieMeter  metrics.Meter
	ErrorMeter     metrics.Meter
	NoBidMeter     metrics.Meter
	TimeoutMeter   metrics.Meter
	RequestMeter   metrics.Meter
	RequestTimer   metrics.Timer
	PriceHistogram metrics.Histogram
}

var (
	metricsRegistry      metrics.Registry
	mRequestMeter        metrics.Meter
	mNoCookieMeter       metrics.Meter
	mSafariRequestMeter  metrics.Meter
	mSafariNoCookieMeter metrics.Meter
	mErrorMeter          metrics.Meter
	mInvalidMeter        metrics.Meter
	mRequestTimer        metrics.Timer

	adapterMetrics map[string]*AdapterMetrics

	accountMetrics        map[string]*AccountMetrics // FIXME -- this seems like an unbounded queue
	accountMetricsRWMutex sync.RWMutex

	requireUUID2 bool
	cookieDomain string
)

var exchanges map[string]adapters.Adapter
var dataCache cache.Cache
var reqSchema *gojsonschema.Schema

type bidResult struct {
	bidder   *pbs.PBSBidder
	bid_list pbs.PBSBidSlice
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
		glog.Errorf("Error marshalling error: %s", err)
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
		accountMetrics[id] = am
	}
	accountMetricsRWMutex.Unlock()

	return am
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

	if requireUUID2 {
		if _, err := r.Cookie("uuid2"); err != nil {
			mNoCookieMeter.Mark(1)
			if isSafari {
				mSafariNoCookieMeter.Mark(1)
			}
			resp := pbs.PBSResponse{Status: "no_cookie"}
			b, _ := json.Marshal(&resp)
			c := http.Cookie{
				Name:    "uuid2",
				Value:   fmt.Sprintf("%d", rand.Int63()),
				Domain:  cookieDomain,
				Expires: time.Now().Add(180 * 24 * time.Hour),
			}
			http.SetCookie(w, &c)
			w.Write(b)
			return
		}
	}

	pbs_req, err := pbs.ParsePBSRequest(r, dataCache)
	if err != nil {
		glog.Info("error parsing request", err)
		writeAuctionError(w, "Error parsing request", err)
		mErrorMeter.Mark(1)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pbs_req.TimeoutMillis))
	defer cancel()

	if glog.V(1) {
		glog.Infof("Request for %d ad units on url %s by account %s", len(pbs_req.AdUnits), pbs_req.Url, pbs_req.AccountID)
	}

	_, err = dataCache.GetAccount(pbs_req.AccountID)
	if err != nil {
		glog.Info("Invalid account id: ", err)
		writeAuctionError(w, "Unknown account id", fmt.Errorf("Unknown account"))
		mErrorMeter.Mark(1)
		return
	}

	am := getAccountMetrics(pbs_req.AccountID)
	am.RequestMeter.Mark(1)

	pbs_resp := pbs.PBSResponse{
		Status:       "OK",
		TID:          pbs_req.Tid,
		BidderStatus: pbs_req.Bidders,
	}

	ch := make(chan bidResult)
	sentBids := 0
	for _, bidder := range pbs_req.Bidders {
		if ex, ok := exchanges[bidder.BidderCode]; ok {
			ametrics := adapterMetrics[bidder.BidderCode]
			ametrics.RequestMeter.Mark(1)
			if pbs_req.GetUserID(ex.FamilyName()) == "" {
				bidder.NoCookie = true
				bidder.UsersyncInfo = ex.GetUsersyncInfo()
				ametrics.NoCookieMeter.Mark(1)
				continue
			}
			sentBids++
			go func(bidder *pbs.PBSBidder) {
				start := time.Now()
				bid_list, err := ex.Call(ctx, pbs_req, bidder)
				bidder.ResponseTime = int(time.Since(start) / time.Millisecond)
				ametrics.RequestTimer.UpdateSince(start)
				if err != nil {
					switch err {
					case context.DeadlineExceeded:
						ametrics.TimeoutMeter.Mark(1)
						bidder.Error = "Timed out"
					case context.Canceled:
						fallthrough
					default:
						ametrics.ErrorMeter.Mark(1)
						bidder.Error = err.Error()
					}
				} else if bid_list != nil {
					bidder.NumBids = len(bid_list)
					am.BidsReceivedMeter.Mark(int64(bidder.NumBids))
					for _, bid := range bid_list {
						ametrics.PriceHistogram.Update(int64(bid.Price * 1000))
						am.PriceHistogram.Update(int64(bid.Price * 1000))
					}
				} else {
					bidder.NoBid = true
					ametrics.NoBidMeter.Mark(1)
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

	/*
		    // record bids by code
		    // code_bids := make(map[string]PBSBidSlice)

		        for _, bid :=  range result.bid_list {
		            code_bids[bid.AdUnitCode] = append(code_bids[bid.AdUnitCode], bid)
		        }

			// loop through ad units to find top bid
			for adunit := range pbs_req.AdUnits {
				bar := code_bids[adunit.Code]

				if len(bar) == 0 {
					if glog.V(1) {
						glog.Infof("No bids for ad unit '%s'", code)
					}
					continue
				}
				sort.Sort(bar)

				if glog.V(1) {
					glog.Infof("Ad unit %s got %d bids. Highest CPM $%.2f, second CPM $%.2f, from bidder %s", code, len(bar), bar[0].Price.First,
						bar[0].Price.Second, bar[0].BidderCode)
				}
			}
	*/

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	//enc.SetIndent("", "  ")
	enc.Encode(pbs_resp)
	mRequestTimer.UpdateSince(pbs_req.Start)
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
		fmt.Fprintf(w, "User Agent: %v", ua)
	}
	ip, port, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		//return nil, fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)

		fmt.Fprintf(w, "userip: %q is not IP:port", req.RemoteAddr)

	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
		//return nil, fmt.Errorf("userip: %q is not IP:port", req.RemoteAddr)
		fmt.Fprintf(w, "userip: %q is not IP:port", req.RemoteAddr)
		return

	}

	// This will only be defined when site is accessed via non-anonymous proxy
	// and takes precedence over RemoteAddr
	// Header.Get is case-insensitive
	forward := req.Header.Get("X-Forwarded-For")

	fmt.Fprintf(w, "<p>IP: %s</p>", ip)
	fmt.Fprintf(w, "<p>Port: %s</p>", port)
	fmt.Fprintf(w, "<p>Forwarded for: %s</p>", forward)

	for k, v := range req.Header {
		fmt.Fprintf(w, "<p>%s: %s</p>", k, v)
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
		fmt.Fprintf(w, "Validation chema not loaded\n")
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

func loadPostgresDataCache() (cache.Cache, error) {
	mem := sigar.Mem{}
	mem.Get()

	cfg := cache.PostgresDataCacheConfig{
		Dbname:   viper.GetString("datacache.dbname"),
		Host:     viper.GetString("datacache.host"),
		User:     viper.GetString("datacache.user"),
		Password: viper.GetString("datacache.password"),
		Size:     viper.GetInt("datacache.cache_size"),
		TTL:      viper.GetInt("datacache.ttl_seconds"),
	}

	return cache.NewPostgresDataCache(&cfg)

}

func loadDataCache() {
	var err error

	cacheType := viper.GetString("datacache.type")
	switch cacheType {
	case "dummy":
		dataCache = cache.NewDummyCache()

	case "postgres":
		dataCache, err = loadPostgresDataCache()
		if err != nil {
			glog.Fatalf("Postgres cache not configured: %s", err.Error())
		}

	case "filecache":
		dataCache, err = cache.NewFileCache(viper.GetString("datacache.filename"))
		if err != nil {
			glog.Fatalf("Failed to load filecach: %s", err.Error())
		}

	default:
		log.Fatalf("Unknown datacache.type: %s", cacheType)
	}
}

func main() {
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

	viper.SetDefault("pubmatic_endpoint", "http://openbid-useast.pubmatic.com/translator?")
	viper.SetDefault("rubicon_endpoint", "http://staged-by.rubiconproject.com/a/api/exchange.json")
	viper.ReadInConfig()

	flag.Parse() // read glog settings from cmd line

	externalURL := viper.GetString("external_url")
	requireUUID2 = viper.GetBool("require_uuid2")

	loadDataCache()

	exchanges = map[string]adapters.Adapter{
		"appnexus":      adapters.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, externalURL),
		"districtm":     adapters.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, externalURL),
		"indexExchange": adapters.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, externalURL),
		"pubmatic":      adapters.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, viper.GetString("pubmatic_endpoint"), externalURL),
		"rubicon": adapters.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, viper.GetString("rubicon_endpoint"),
			viper.GetString("rubicon_xapi_username"), viper.GetString("rubicon_xapi_password"), externalURL),
		"audienceNetwork": adapters.NewFacebookAdapter(adapters.DefaultHTTPAdapterConfig, viper.GetString("facebook_platform_id"), externalURL),
	}

	metricsRegistry = metrics.NewPrefixedRegistry("prebidserver.")
	mRequestMeter = metrics.GetOrRegisterMeter("requests", metricsRegistry)
	mNoCookieMeter = metrics.GetOrRegisterMeter("no_cookie_requests", metricsRegistry)
	mSafariRequestMeter = metrics.GetOrRegisterMeter("safari_requests", metricsRegistry)
	mSafariNoCookieMeter = metrics.GetOrRegisterMeter("safari_no_cookie_requests", metricsRegistry)
	mErrorMeter = metrics.GetOrRegisterMeter("error_requests", metricsRegistry)
	mInvalidMeter = metrics.GetOrRegisterMeter("invalid_requests", metricsRegistry)
	mRequestTimer = metrics.GetOrRegisterTimer("request_time", metricsRegistry)

	accountMetrics = make(map[string]*AccountMetrics)

	adapterMetrics = make(map[string]*AdapterMetrics)
	for exchange := range exchanges {
		a := AdapterMetrics{}
		a.NoCookieMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.no_cookie_requests", exchange), metricsRegistry)
		a.ErrorMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.error_requests", exchange), metricsRegistry)
		a.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.requests", exchange), metricsRegistry)
		a.NoBidMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.no_bid_requests", exchange), metricsRegistry)
		a.TimeoutMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.timeout_requests", exchange), metricsRegistry)
		a.RequestTimer = metrics.GetOrRegisterTimer(fmt.Sprintf("adapter.%s.request_time", exchange), metricsRegistry)
		a.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("adapter.%s.prices", exchange), metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
		adapterMetrics[exchange] = &a
	}

	if viper.Get("metrics") != nil {
		go influxdb.InfluxDB(
			metricsRegistry,                     // metrics registry
			time.Second*10,                      // interval
			viper.GetString("metrics.host"),     // the InfluxDB url
			viper.GetString("metrics.database"), // your InfluxDB database
			viper.GetString("metrics.username"), // your InfluxDB user
			viper.GetString("metrics.password"), // your InfluxDB password
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
		adminURI := fmt.Sprintf("%s:%s", viper.GetString("host"), viper.GetString("admin_port"))
		fmt.Println("Admin running on: ", adminURI)
		glog.Fatal(http.ListenAndServe(adminURI, nil))
	}()

	router := httprouter.New()
	router.POST("/auction", auction)
	router.POST("/validate", validate)
	router.GET("/status", status)
	router.GET("/", serveIndex)
	router.GET("/ip", getIP)
	router.ServeFiles("/static/*filepath", http.Dir("static"))

	cookieDomain = viper.GetString("cookie_domain")

	pbs.InitUsersyncHandlers(router, metricsRegistry, cookieDomain, externalURL, viper.GetString("recaptcha_secret"))

	// Add CORS middleware
	c := cors.New(cors.Options{AllowCredentials: true})
	corsRouter := c.Handler(router)

	// Add no cache headers
	noCacheHandler := NoCache{corsRouter}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", viper.GetString("host"), viper.GetString("port")),
		Handler:      noCacheHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	fmt.Println("Server running on: ", server.Addr)
	glog.Fatal(server.ListenAndServe())
}
