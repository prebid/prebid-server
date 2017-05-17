package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/cloudfoundry/gosigar"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/rcrowley/go-metrics"
	"github.com/rs/cors"
	influxdb "github.com/vrischmann/go-metrics-influxdb"
	"github.com/xeipuuv/gojsonschema"
	"github.com/xojoc/useragent"

	// internal prebid-server libs
	"github.com/prebid/prebid-server"
	"github.com/prebid/prebid-server/cache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
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

type BidCache struct {
	Adm  string `json:"adm,omitempty"`
	NURL string `json:"nurl,omitempty"`
}

type bidResult struct {
	bidder   *pbs.PBSBidder
	bid_list pbs.PBSBidSlice
}

func (s *Server) writeAuctionError(w http.ResponseWriter, status string, err error) {
	resp := pbs.PBSResponse{Status: status}
	if err != nil {
		resp.Status = fmt.Sprintf("%s: %v", status, err)
	}
	b, err := json.Marshal(&resp)
	if err != nil {
		glog.Errorf("Error marshalling error: %s", err)
		return
	}
	w.Write(b)
}

func (s *Server) auction(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Add("Content-Type", "application/json")

	s.mRequestMeter.Mark(1)

	isSafari := false
	if ua := useragent.Parse(r.Header.Get("User-Agent")); ua != nil {
		if ua.Type == useragent.Browser && ua.Name == "Safari" {
			isSafari = true
			s.mSafariRequestMeter.Mark(1)
		}
	}

	pbs_req, err := pbs.ParsePBSRequest(r, s.dataCache, s.defaultTimeoutMS)
	if err != nil {
		glog.Info("error parsing request", err)
		s.writeAuctionError(w, "Error parsing request", err)
		s.mErrorMeter.Mark(1)
		return
	}

	if pbs_req.App != nil {
		s.mAppRequestMeter.Mark(1)
	} else if s.requireUUID2 {
		if _, err := r.Cookie("uuid2"); err != nil {
			s.mNoCookieMeter.Mark(1)
			if isSafari {
				s.mSafariNoCookieMeter.Mark(1)
			}
			b, _ := json.Marshal(pbs.PBSResponse{Status: "no_cookie"})
			c := http.Cookie{
				Name:    "uuid2",
				Value:   fmt.Sprintf("%d", rand.Int63()),
				Domain:  s.cookieDomain,
				Expires: time.Now().Add(180 * 24 * time.Hour),
			}
			http.SetCookie(w, &c)
			w.Write(b)
			return
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*time.Duration(pbs_req.TimeoutMillis))
	defer cancel()

	if glog.V(1) {
		glog.Infof("Request for %d ad units on url %s by account %s", len(pbs_req.AdUnits), pbs_req.Url, pbs_req.AccountID)
	}

	if _, err = s.dataCache.Accounts().Get(pbs_req.AccountID); err != nil {
		glog.Info("Invalid account id: ", err)
		s.writeAuctionError(w, "Unknown account id", fmt.Errorf("Unknown account"))
		s.mErrorMeter.Mark(1)
		return
	}

	am := s.getAccountMetrics(pbs_req.AccountID)
	am.RequestMeter.Mark(1)

	pbs_resp := pbs.PBSResponse{
		Status:       "OK",
		TID:          pbs_req.Tid,
		BidderStatus: pbs_req.Bidders,
	}

	ch := make(chan *bidResult)
	sentBids := 0
	for _, bidder := range pbs_req.Bidders {
		if ex, ok := s.exchanges[bidder.BidderCode]; ok {
			ametrics := s.adapterMetrics[bidder.BidderCode]
			ametrics.RequestMeter.Mark(1)
			if pbs_req.App == nil && pbs_req.GetUserID(ex.FamilyName()) == "" {
				bidder.NoCookie = true
				bidder.UsersyncInfo = ex.GetUsersyncInfo()
				ametrics.NoCookieMeter.Mark(1)
				continue
			}
			sentBids++
			go s.requestBids(ch, ex, ametrics, am, ctx, pbs_req, bidder)
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
		if err = pbc.Put(ctx, cobjs); err != nil {
			s.writeAuctionError(w, "Prebid cache failed", err)
			s.mErrorMeter.Mark(1)
			return
		}
		for i, bid := range pbs_resp.Bids {
			bid.CacheID = cobjs[i].UUID
			bid.NURL = ""
			bid.Adm = ""
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
	s.mRequestTimer.UpdateSince(pbs_req.Start)
}

func (s *Server) requestBids(ch chan *bidResult, ex prebid.Adapter, ametrics *AdapterMetrics, am *AccountMetrics, ctx context.Context, pbs_req *pbs.PBSRequest, bidder *pbs.PBSBidder) {
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

	ch <- &bidResult{
		bidder:   bidder,
		bid_list: bid_list,
	}
}

func (s *Server) status(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// could add more logic here, but doing nothing means 200 OK
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
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
func (s *Server) getIP(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	if ua := useragent.Parse(req.Header.Get("User-Agent")); ua != nil {
		fmt.Fprintf(w, "User Agent: %v", ua)
	}
	ip, port, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		fmt.Fprintf(w, "userip: %q is not IP:port", req.RemoteAddr)
		return
	}

	userIP := net.ParseIP(ip)
	if userIP == nil {
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

func (s *Server) validate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	w.Header().Add("Content-Type", "text/plain")
	defer r.Body.Close()
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Unable to read body\n")
		return
	}

	if s.reqSchema == nil {
		fmt.Fprintf(w, "Validation schema not loaded\n")
		return
	}

	js := gojsonschema.NewStringLoader(string(b))
	result, err := s.reqSchema.Validate(js)
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

func (s *Server) loadPostgresDataCache(c *config.Configuration) (cache.Cache, error) {
	mem := sigar.Mem{}
	mem.Get()
	//
	// cfg := cache.Configuration{
	// 	Dbname:   c.DataCache.Database,
	// 	Host:     c.DataCache.Host,
	// 	User:     c.DataCache.Username,
	// 	Password: c.DataCache.Password,
	// 	Size:     c.DataCache.CacheSize,
	// 	TTL:      c.DataCache.TTLSeconds,
	// }
	// return cache.NewPostgresDataCache(&cfg)
	return nil, nil
}

// Server is the prebid-server
type Server struct {
	externalURL      string
	defaultTimeoutMS uint64

	mRequestTimer        metrics.Timer
	metricsRegistry      metrics.Registry
	mRequestMeter        metrics.Meter
	mAppRequestMeter     metrics.Meter
	mNoCookieMeter       metrics.Meter
	mSafariRequestMeter  metrics.Meter
	mSafariNoCookieMeter metrics.Meter
	mErrorMeter          metrics.Meter
	mInvalidMeter        metrics.Meter
	adapterMetrics       map[string]*AdapterMetrics

	accountMetrics        map[string]*AccountMetrics // FIXME -- this seems like an unbounded queue
	accountMetricsRWMutex sync.RWMutex

	requireUUID2 bool
	cookieDomain string

	exchanges map[string]prebid.Adapter
	dataCache cache.Cache
	reqSchema *gojsonschema.Schema
}

func (s *Server) getAccountMetrics(id string) *AccountMetrics {
	if am := s.getAccountMetricsFromGlobal(id); am != nil {
		return am
	}
	return s.newAccountMetrics(id)
}

func (s *Server) getAccountMetricsFromGlobal(id string) *AccountMetrics {
	s.accountMetricsRWMutex.RLock()
	defer s.accountMetricsRWMutex.RUnlock()
	if am, ok := s.accountMetrics[id]; ok {
		return am
	}
	return nil
}

func (s *Server) newAccountMetrics(id string) *AccountMetrics {
	s.accountMetricsRWMutex.Lock()
	defer s.accountMetricsRWMutex.Unlock()

	am := &AccountMetrics{}
	am.RequestMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.requests", id), s.metricsRegistry)
	am.BidsReceivedMeter = metrics.GetOrRegisterMeter(fmt.Sprintf("account.%s.bids_received", id), s.metricsRegistry)
	am.PriceHistogram = metrics.GetOrRegisterHistogram(fmt.Sprintf("account.%s.prices", id), s.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015))
	s.accountMetrics[id] = am
	return am
}

func NewServer(c *config.Configuration, dataCache cache.Cache) (*Server, error) {
	metricsRegistry := metrics.NewPrefixedRegistry("prebidserver.")

	s := &Server{
		externalURL:          c.ExternalURL,
		requireUUID2:         c.RequireUUID2,
		dataCache:            dataCache,
		accountMetrics:       make(map[string]*AccountMetrics),
		adapterMetrics:       make(map[string]*AdapterMetrics),
		metricsRegistry:      metricsRegistry,
		mRequestMeter:        metrics.GetOrRegisterMeter("requests", metricsRegistry),
		mAppRequestMeter:     metrics.GetOrRegisterMeter("app_requests", metricsRegistry),
		mNoCookieMeter:       metrics.GetOrRegisterMeter("no_cookie_requests", metricsRegistry),
		mSafariRequestMeter:  metrics.GetOrRegisterMeter("safari_requests", metricsRegistry),
		mSafariNoCookieMeter: metrics.GetOrRegisterMeter("safari_no_cookie_requests", metricsRegistry),
		mErrorMeter:          metrics.GetOrRegisterMeter("error_requests", metricsRegistry),
		mInvalidMeter:        metrics.GetOrRegisterMeter("invalid_requests", metricsRegistry),
		mRequestTimer:        metrics.GetOrRegisterTimer("request_time", metricsRegistry),
	}

	s.exchanges = map[string]prebid.Adapter{
	// "appnexus":      appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL),
	// "districtm":     appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL),
	// "indexExchange": index.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL),
	// "pubmatic":      pubmatic.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, viper.GetString("pubmatic_endpoint"), s.externalURL),
	// "pulsepoint":    pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, viper.GetString("pulsepoint_endpoint"), s.externalURL),
	// "rubicon": rubicon.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, viper.GetString("rubicon_endpoint"),
	// 	viper.GetString("rubicon_xapi_username"), viper.GetString("rubicon_xapi_password"), viper.GetString("rubicon_usersync_url")),
	// "audienceNetwork": facebook.NewFacebookAdapter(adapters.DefaultHTTPAdapterConfig, viper.GetString("facebook_platform_id"), viper.GetString("facebook_usersync_url")),
	}

	for exchange := range s.exchanges {
		a := &AdapterMetrics{
			NoCookieMeter:  metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.no_cookie_requests", exchange), s.metricsRegistry),
			ErrorMeter:     metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.error_requests", exchange), s.metricsRegistry),
			RequestMeter:   metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.requests", exchange), s.metricsRegistry),
			NoBidMeter:     metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.no_bid_requests", exchange), s.metricsRegistry),
			TimeoutMeter:   metrics.GetOrRegisterMeter(fmt.Sprintf("adapter.%s.timeout_requests", exchange), s.metricsRegistry),
			RequestTimer:   metrics.GetOrRegisterTimer(fmt.Sprintf("adapter.%s.request_time", exchange), s.metricsRegistry),
			PriceHistogram: metrics.GetOrRegisterHistogram(fmt.Sprintf("adapter.%s.prices", exchange), s.metricsRegistry, metrics.NewExpDecaySample(1028, 0.015)),
		}
		s.adapterMetrics[exchange] = a
	}
	return s, nil
}

// handleMetrics requires a hostname to send data to influxdb
func (s *Server) handleMetrics(c *config.Configuration) {
	if c.Metrics.Host == "" {
		// if there is no hostname then don't emit data to influxdb
		return
	}
	go influxdb.InfluxDB(
		s.metricsRegistry,  // metrics registry
		time.Second*10,     // interval
		c.Metrics.Host,     // the InfluxDB url
		c.Metrics.Database, // your InfluxDB database
		c.Metrics.Username, // your InfluxDB user
		c.Metrics.Password, // your InfluxDB password
	)
}

func (s *Server) loadPBSSchema() {
	b, err := ioutil.ReadFile("static/pbs_request.json")
	if err != nil {
		glog.Errorf("Unable to open pbs_request.json: %v", err)
		return
	}
	sl := gojsonschema.NewStringLoader(string(b))
	s.reqSchema, err = gojsonschema.NewSchema(sl)
	if err != nil {
		glog.Errorf("Unable to load request schema: %v", err)
	}
}

func (s *Server) Run(c *config.Configuration) {
	// first handle any metrics to influxdb
	s.handleMetrics(c)

	// load pbs schema
	s.loadPBSSchema()

	// run admin server in its own goroutine
	go s.runAdmin(c)

	router := s.newHTTPRouter()

	s.cookieDomain = c.CookieDomain

	pbs.InitUsersyncHandlers(router, s.metricsRegistry, s.cookieDomain, s.externalURL, c.RecaptchaSecret)

	pbc.InitPrebidCache(c.CacheURL)

	// Add CORS middleware
	corsRouter := cors.New(cors.Options{AllowCredentials: true}).Handler(router)

	// Add no cache headers
	noCacheHandler := NoCache{corsRouter}

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", c.Host, c.Port),
		Handler:      noCacheHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	fmt.Println("Server running on: ", server.Addr)
	glog.Fatal(server.ListenAndServe())
}

// runAdmin will run admin on a different port that's not exposed
func (s *Server) runAdmin(c *config.Configuration) {
	/* Run admin on different port thats not exposed */
	adminURI := fmt.Sprintf("%s:%d", c.Host, c.AdminPort)
	fmt.Println("Admin running on: ", adminURI)
	glog.Fatal(http.ListenAndServe(adminURI, nil))
}

func (s *Server) newHTTPRouter() *httprouter.Router {
	router := httprouter.New()
	router.POST("/auction", s.auction)
	router.POST("/validate", s.validate)
	router.GET("/status", s.status)
	router.GET("/", s.serveIndex)
	router.GET("/ip", s.getIP)
	router.ServeFiles("/static/*filepath", http.Dir("static"))

	return router
}

//
// func newadaptersConfigurations() []adapters.Configuration {
// 	adapters = make([]adapters.Configuration, 0)
// 	adapters = append(adapters, adapters.Configuration{"appnexus", "", "", "", ""})
// 	adapters = append(adapters, adapters.Configuration{"districtm", "", "", "", ""})
// 	adapters = append(adapters, adapters.Configuration{"indexExchange", "", "", "", ""})
// 	adapters = append(adapters, adapters.Configuration{"pubmatic", viper.GetString("pubmatic_endpoint"), "", "", ""})
// 	adapters = append(adapters, adapters.Configuration{"pulsepoint", viper.GetString("pulsepoint_endpoint"), "", "", ""})
// 	adapters = append(adapters, adapters.Configuration{"rubicon", viper.GetString("rubicon_endpoint"), viper.GetString("rubicon_xapi_username"), viper.GetString("rubicon_xapi_password"), viper.GetString("rubicon_usersync_url")})
// 	adapters = append(adapters, adapters.Configuration{"audienceNetwork", "", viper.GetString("facebook_platform_id"), "", viper.GetString("facebook_usersync_url")})
//
// 	return adapters
// }

func (s *Server) handleExchanges(c *config.Configuration) {
	// for _, a := range c.Adapters {
	// 	switch a.Name {
	// 	case "appnexus":
	// 		s.exchanges["appnexus"] = appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL, a)
	// 	case "districtm":
	// 		s.exchanges["districtm"] = appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL, a)
	// 	case "indexExchange":
	// 		s.exchanges["indexExchange"] = index.NewIndexAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL, a)
	// 	case "pubmatic":
	// 		s.exchanges["pubmatic"] = pubmatic.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL, a)
	// 	case "pulsepoint":
	// 		s.exchanges["pulsepoint"] = pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL, a)
	// 	case "rubicon":
	// 		s.exchanges["rubicon"] = rubicon.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL, a)
	// 	case "audienceNetwork":
	// 		s.exchanges["audienceNetwork"] = facebook.NewFacebookAdapter(adapters.DefaultHTTPAdapterConfig, s.externalURL, a)
	// 	default:
	// 		panic("unknown adapter")
	// 	}
	// }
}
