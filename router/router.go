package router

import (
	"context"
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adform"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/appnexus"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/conversant"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/ix"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/lifestreet"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pubmatic"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/pulsepoint"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/rubicon"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/sovrn"
	"github.com/PubMatic-OpenWrap/prebid-server/analytics"
	analyticsConf "github.com/PubMatic-OpenWrap/prebid-server/analytics/config"
	"github.com/PubMatic-OpenWrap/prebid-server/cache"
	"github.com/PubMatic-OpenWrap/prebid-server/cache/dummycache"
	"github.com/PubMatic-OpenWrap/prebid-server/cache/filecache"
	"github.com/PubMatic-OpenWrap/prebid-server/cache/postgrescache"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/currencies"
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints"
	"github.com/PubMatic-OpenWrap/prebid-server/endpoints/openrtb2"
	"github.com/PubMatic-OpenWrap/prebid-server/exchange"
	"github.com/PubMatic-OpenWrap/prebid-server/gdpr"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics"
	metricsConf "github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics/config"
	pbc "github.com/PubMatic-OpenWrap/prebid-server/prebid_cache_client"
	"github.com/PubMatic-OpenWrap/prebid-server/ssl"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/backends/empty_fetcher"
	storedRequestsConf "github.com/PubMatic-OpenWrap/prebid-server/stored_requests/config"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync/usersyncers"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var dataCache cache.Cache
var exchanges map[string]adapters.Adapter
var (
	g_syncers           map[openrtb_ext.BidderName]usersync.Usersyncer
	g_cfg               *config.Configuration
	g_ex                exchange.Exchange
	g_paramsValidator   openrtb_ext.BidderParamValidator
	g_storedReqFetcher  stored_requests.Fetcher
	g_gdprPerms         gdpr.Permissions
	g_metrics           pbsmetrics.MetricsEngine
	g_analytics         analytics.PBSAnalyticsModule
	g_disabledBidders   map[string]string
	g_categoriesFetcher stored_requests.CategoryFetcher
	g_bidderMap         map[string]openrtb_ext.BidderName
	g_defReqJSON        []byte
)

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
func NewJsonDirectoryServer(schemaDirectory string, validator openrtb_ext.BidderParamValidator, aliases map[string]string) httprouter.Handle {
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

	// Add in any default aliases
	for aliasName, bidderName := range aliases {
		bidderData, ok := data[bidderName]
		if !ok {
			glog.Fatalf("Default alias (%s) exists referencing unknown bidder: %s", aliasName, bidderName)
		}
		data[aliasName] = bidderData
	}

	response, err := json.Marshal(data)
	if err != nil {
		glog.Fatalf("Failed to marshal bidder param JSON-schema: %v", err)
	}

	return func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		w.Header().Add("Content-Type", "application/json")
		w.Write(response)
	}
}

func serveIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "static/index.html")
}

type NoCache struct {
	Handler http.Handler
}

func (m NoCache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Add("Pragma", "no-cache")
	w.Header().Add("Expires", "0")
	m.Handler.ServeHTTP(w, r)
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

func newExchangeMap(cfg *config.Configuration) map[string]adapters.Adapter {
	// These keys _must_ coincide with the bidder code in Prebid.js, if the adapter exists in both projects
	return map[string]adapters.Adapter{
		"appnexus":   appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].PlatformID),
		"districtm":  appnexus.NewAppNexusAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].PlatformID),
		"ix":         ix.NewIxAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIx))].Endpoint),
		"pubmatic":   pubmatic.NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint),
		"pulsepoint": pulsepoint.NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPulsepoint)].Endpoint),
		"rubicon": rubicon.NewRubiconAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username, cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password, cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),
		"lifestreet": lifestreet.NewLifestreetAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderLifestreet)].Endpoint),
		"conversant": conversant.NewConversantAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderConversant)].Endpoint),
		"adform":     adform.NewAdformAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderAdform)].Endpoint),
		"sovrn":      sovrn.NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint),
	}
}

type Router struct {
	*httprouter.Router
	MetricsEngine   *metricsConf.DetailedMetricsEngine
	ParamsValidator openrtb_ext.BidderParamValidator
	Shutdown        func()
}

func New(cfg *config.Configuration, rateConvertor *currencies.RateConverter) (r *Router, err error) {

	const schemaDirectory = "/home/http/GO_SERVER/dmhbserver/static/bidder-params"
	const infoDirectory = "/home/http/GO_SERVER/dmhbserver/static/bidder-info"

	r = &Router{
		Router: httprouter.New(),
	}

	// For bid processing, we need both the hardcoded certificates and the certificates found in container's
	// local file system
	certPool := ssl.GetRootCAPool()
	var readCertErr error
	certPool, readCertErr = ssl.AppendPEMFileToRootCAPool(certPool, cfg.PemCertsFile)
	if readCertErr != nil {
		glog.Infof("Could not read certificates file: %s \n", readCertErr.Error())
	}

	theClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        cfg.Client.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.Client.MaxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(cfg.Client.IdleConnTimeout) * time.Second,
			TLSClientConfig:     &tls.Config{RootCAs: certPool},
		},
	}
	// Hack because of how legacy handles districtm
	legacyBidderList := openrtb_ext.BidderList()
	legacyBidderList = append(legacyBidderList, openrtb_ext.BidderName("districtm"))

	g_cfg = cfg
	var db *sql.DB
	// Metrics engine
	r.MetricsEngine = metricsConf.NewMetricsEngine(cfg, legacyBidderList)
	db, _, g_storedReqFetcher, _, g_categoriesFetcher, _ = storedRequestsConf.NewStoredRequests(cfg, r.MetricsEngine, theClient, r.Router)

	// todo(zachbadgett): better shutdown
	//r.Shutdown = shutdown
	if err := loadDataCache(cfg, db); err != nil {
		return nil, fmt.Errorf("Prebid Server could not load data cache: %v", err)
	}

	g_analytics = analyticsConf.NewPBSAnalytics(&cfg.Analytics)

	// Metrics engine
	g_metrics = metricsConf.NewMetricsEngine(cfg, legacyBidderList)

	g_paramsValidator, err = openrtb_ext.NewBidderParamsValidator(schemaDirectory)
	if err != nil {
		glog.Fatalf("Failed to create the bidder params validator. %v", err)
	}

	g_disabledBidders = map[string]string{
		"indexExchange": "Bidder \"indexExchange\" has been deprecated and is no longer available. Please use bidder \"ix\" and note that the bidder params have changed.",
	}
	//var bidderList []openrtb_ext.BidderName

	p, _ := filepath.Abs(infoDirectory)
	bidderInfos := adapters.ParseBidderInfos(cfg.Adapters, p, openrtb_ext.BidderList())

	g_bidderMap = exchange.DisableBidders(bidderInfos, g_disabledBidders)

	_, g_defReqJSON = readDefaultRequest(cfg.DefReqConfig)

	g_syncers = usersyncers.NewSyncerMap(cfg)
	g_gdprPerms = gdpr.NewPermissions(context.Background(), cfg.GDPR, adapters.GDPRAwareSyncerIDs(g_syncers), theClient)

	exchanges = newExchangeMap(cfg)

	g_ex = exchange.NewExchange(theClient, pbc.NewClient(&cfg.CacheURL, &cfg.ExtCacheURL, r.MetricsEngine), cfg, g_metrics, bidderInfos, g_gdprPerms, rateConvertor)

	/*
			openrtbEndpoint, err := openrtb2.NewEndpoint(theExchange, paramsValidator, fetcher, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, bidderMap, categoriesFetcher)

			if err != nil {
				glog.Fatalf("Failed to create the openrtb endpoint handler. %v", err)
			}

			ampEndpoint, err := openrtb2.NewAmpEndpoint(theExchange, paramsValidator, ampFetcher, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, bidderMap, categoriesFetcher)

			if err != nil {
				glog.Fatalf("Failed to create the amp endpoint handler. %v", err)
			}

			videoEndpoint, err := openrtb2.NewVideoEndpoint(theExchange, paramsValidator, fetcher, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, bidderMap, categoriesFetcher)
			if err != nil {
				glog.Fatalf("Failed to create the video endpoint handler. %v", err)
			}

			r.POST("/auction", endpoints.Auction(cfg, syncers, gdprPerms, r.MetricsEngine, dataCache, exchanges))
			r.POST("/openrtb2/auction", openrtbEndpoint)
			r.POST("/openrtb2/video", videoEndpoint)
			r.GET("/openrtb2/amp", ampEndpoint)
			r.GET("/info/bidders", infoEndpoints.NewBiddersEndpoint(defaultAliases))
			r.GET("/info/bidders/:bidderName", infoEndpoints.NewBidderDetailsEndpoint(bidderInfos, defaultAliases))
			r.GET("/bidders/params", NewJsonDirectoryServer(schemaDirectory, paramsValidator, defaultAliases))
			r.POST("/cookie_sync", endpoints.NewCookieSyncEndpoint(syncers, cfg, gdprPerms, r.MetricsEngine, pbsAnalytics))
			r.GET("/status", endpoints.NewStatusEndpoint(cfg.StatusResponse))
			r.GET("/", serveIndex)
			r.ServeFiles("/static/*filepath", http.Dir("static"))

		userSyncDeps := &pbs.UserSyncDeps{
			HostCookieConfig: &(cfg.HostCookie),
			ExternalUrl:      cfg.ExternalURL,
			RecaptchaSecret:  cfg.RecaptchaSecret,
			MetricsEngine:    r.MetricsEngine,
			PBSAnalytics:     pbsAnalytics,
		}

			r.GET("/setuid", endpoints.NewSetUIDEndpoint(cfg.HostCookie, syncers, gdprPerms, pbsAnalytics, r.MetricsEngine))
			r.GET("/getuids", endpoints.NewGetUIDsEndpoint(cfg.HostCookie))
			r.POST("/optout", userSyncDeps.OptOut)
			r.GET("/optout", userSyncDeps.OptOut)
	*/
	return r, nil
}

func OrtbAuctionEndpointWrapper(w http.ResponseWriter, r *http.Request) error {
	ortbAuctionEndpoint, err := openrtb2.NewEndpoint(g_ex, g_paramsValidator, g_storedReqFetcher, g_categoriesFetcher, g_cfg, g_metrics, g_analytics, g_disabledBidders, g_defReqJSON, g_bidderMap)
	if err != nil {
		return err
	}
	ortbAuctionEndpoint(w, r, nil)
	return nil
}

func VideoAuctionEndpointWrapper(w http.ResponseWriter, r *http.Request) error {
	videoAuctionEndpoint, err := openrtb2.NewCTVEndpoint(g_ex, g_paramsValidator, g_storedReqFetcher, empty_fetcher.EmptyFetcher{}, g_categoriesFetcher, g_cfg, g_metrics, g_analytics, g_disabledBidders, g_defReqJSON, g_bidderMap)
	if err != nil {
		return err
	}
	videoAuctionEndpoint(w, r, nil)
	return nil
}

func AuctionWrapper(w http.ResponseWriter, r *http.Request) {
	auction := endpoints.Auction(g_cfg, g_syncers, g_gdprPerms, g_metrics, dataCache, exchanges)
	auction(w, r, nil)
}

func GetUIDSWrapper(w http.ResponseWriter, r *http.Request) {
	getUID := endpoints.NewGetUIDsEndpoint(g_cfg.HostCookie)
	getUID(w, r, nil)
}

func SetUIDSWrapper(w http.ResponseWriter, r *http.Request) {
	setUID := endpoints.NewSetUIDEndpoint(g_cfg.HostCookie, g_syncers, g_gdprPerms, g_analytics, g_metrics)
	setUID(w, r, nil)
}

func CookieSync(w http.ResponseWriter, r *http.Request) {
	cookiesync := endpoints.NewCookieSyncEndpoint(g_syncers, g_cfg, g_gdprPerms, g_metrics, g_analytics)
	cookiesync(w, r, nil)
}

func SyncerMap() map[openrtb_ext.BidderName]usersync.Usersyncer {
	return g_syncers
}

// Fixes #648
//
// These CORS options pose a security risk... but it's a calculated one.
// People _must_ call us with "withCredentials" set to "true" because that's how we use the cookie sync info.
// We also must allow all origins because every site on the internet _could_ call us.
//
// This is an inherent security risk. However, PBS doesn't use cookies for authorization--just identification.
// We only store the User's ID for each Bidder, and each Bidder has already exposed a public cookie sync endpoint
// which returns that data anyway.
//
// For more info, see:
//
// - https://github.com/rs/cors/issues/55
// - https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS/Errors/CORSNotSupportingCredentials
// - https://portswigger.net/blog/exploiting-cors-misconfigurations-for-bitcoins-and-bounties
func SupportCORS(handler http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowCredentials: true,
		AllowOriginFunc: func(string) bool {
			return true
		},
		AllowedHeaders: []string{"Origin", "X-Requested-With", "Content-Type", "Accept"}})
	return c.Handler(handler)
}

type defReq struct {
	Ext defExt `json:"ext"`
}
type defExt struct {
	Prebid defaultAliases `json:"prebid"`
}
type defaultAliases struct {
	Aliases map[string]string `json:"aliases"`
}

func readDefaultRequest(defReqConfig config.DefReqConfig) (map[string]string, []byte) {
	defReq := &defReq{}
	aliases := make(map[string]string)
	if defReqConfig.Type == "file" {
		if len(defReqConfig.FileSystem.FileName) == 0 {
			return aliases, []byte{}
		}
		defReqJSON, err := ioutil.ReadFile(defReqConfig.FileSystem.FileName)
		if err != nil {
			glog.Fatalf("error reading aliases from file %s: %v", defReqConfig.FileSystem.FileName, err)
			return aliases, []byte{}
		}

		if err := json.Unmarshal(defReqJSON, defReq); err != nil {
			// we might not have aliases defined, but will atleast show that the JSON file is parsable.
			glog.Fatalf("error parsing alias json in file %s: %v", defReqConfig.FileSystem.FileName, err)
			return aliases, []byte{}
		}

		// Read in the alias map if we want to populate the info endpoints with aliases.
		if defReqConfig.AliasInfo {
			aliases = defReq.Ext.Prebid.Aliases
		}
		return aliases, defReqJSON
	}
	return aliases, []byte{}
}
