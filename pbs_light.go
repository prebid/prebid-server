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
	mAppRequestMeter     metrics.Meter
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

type BidCache struct {
	Adm  string `json:"adm,omitempty"`
	NURL string `json:"nurl,omitempty"`
}

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

	pbs_req, err := pbs.ParsePBSRequest(r, dataCache)
	if err != nil {
		glog.Info("error parsing request", err)
		writeAuctionError(w, "Error parsing request", err)
		mErrorMeter.Mark(1)
		return
	}

	status := "OK"
	if pbs_req.App != nil {
		mAppRequestMeter.Mark(1)
	} else if requireUUID2 {
		if _, err := r.Cookie("uuid2"); err != nil {
			mNoCookieMeter.Mark(1)
			if isSafari {
				mSafariNoCookieMeter.Mark(1)
			}
			status = "no_cookie"
			uuid2 := fmt.Sprintf("%d", rand.Int63())
			c := http.Cookie{
				Name:    "uuid2",
				Value:   uuid2,
				Domain:  cookieDomain,
				Expires: time.Now().Add(180 * 24 * time.Hour),
			}
			http.SetCookie(w, &c)
			pbs_req.UserIDs["adnxs"] = uuid2
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pbs_req.TimeoutMillis))
	defer cancel()

	_, err = dataCache.Accounts().Get(pbs_req.AccountID)
	if err != nil {
		glog.Info("Invalid account id: ", err)
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
			ametrics.RequestMeter.Mark(1)
			if pbs_req.App == nil && pbs_req.GetUserID(ex.FamilyName()) == "" {
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

	if pbs_req.CacheMarkup == 1 {
		cobjs := make([]*pbc.CacheObject, len(pbs_resp.Bids))
		for i, bid := range pbs_resp.Bids {
			bc := BidCache{
				Adm:  bid.Adm,
				NURL: bid.NURL,
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

	if glog.V(1) {
		glog.Infof("Request for %d ad units on url %s by account %s got %d bids", len(pbs_req.AdUnits), pbs_req.Url, pbs_req.AccountID, len(pbs_resp.Bids))
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

	viper.SetDefault("adapters.pubmatic.endpoint", "http://openbid-useast.pubmatic.com/translator?")
	viper.SetDefault("adapters.rubicon.endpoint", "http://staged-by.rubiconproject.com/a/api/exchange.json")
	viper.SetDefault("adapters.rubicon.usersync_url", "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid")
	viper.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
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
	if err := serve(cfg); err != nil {
		glog.Fatalf("PreBid Server encountered an error: %v", err)
	}
}

func serve(cfg *config.Configuration) error {
	if err := loadDataCache(cfg); err != nil {
		return fmt.Errorf("Prebid Server could not load data cache: %v", err)
	}

	exchanges = map[string]adapters.Adapter{
		"appnexus":      adapters.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
		"districtm":     adapters.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
		"indexExchange": adapters.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, cfg.ExternalURL),
		"pubmatic":      adapters.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pubmatic"].Endpoint, cfg.ExternalURL),
		"pulsepoint":    adapters.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["pulsepoint"].Endpoint, cfg.ExternalURL),
		"rubicon": adapters.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters["rubicon"].Endpoint,
			cfg.Adapters["rubicon"].XAPI.Username, cfg.Adapters["rubicon"].XAPI.Password, cfg.Adapters["rubicon"].UserSyncURL),
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
	router.POST("/validate", validate)
	router.GET("/status", status)
	router.GET("/", serveIndex)
	router.GET("/ip", getIP)
	router.ServeFiles("/static/*filepath", http.Dir("static"))

	pbs.InitUsersyncHandlers(router, metricsRegistry, cfg.CookieDomain, cfg.ExternalURL, cfg.RecaptchaSecret)

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
