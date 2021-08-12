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

	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/endpoints/events"
	"github.com/prebid/prebid-server/errortypes"

	"github.com/prebid/prebid-server/metrics"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adform"
	"github.com/prebid/prebid-server/adapters/appnexus"
	"github.com/prebid/prebid-server/adapters/conversant"
	"github.com/prebid/prebid-server/adapters/ix"
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
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/router/aspects"
	"github.com/prebid/prebid-server/server/ssl"
	storedRequestsConf "github.com/prebid/prebid-server/stored_requests/config"
	"github.com/prebid/prebid-server/usersync/usersyncers"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var dataCache cache.Cache
var exchanges map[string]adapters.Adapter

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

	bidderMap := openrtb_ext.BuildBidderMap()

	data := make(map[string]json.RawMessage, len(files))
	for _, file := range files {
		bidder := strings.TrimSuffix(file.Name(), ".json")
		bidderName, isValid := bidderMap[bidder]
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
		"appnexus":   appnexus.NewAppNexusLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].PlatformID),
		"districtm":  appnexus.NewAppNexusLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].PlatformID),
		"ix":         ix.NewIxLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIx))].Endpoint),
		"pubmatic":   pubmatic.NewPubmaticLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint),
		"pulsepoint": pulsepoint.NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderPulsepoint)].Endpoint),
		"rubicon": rubicon.NewRubiconLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint,
			cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username, cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password, cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Tracker),
		"conversant": conversant.NewConversantLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderConversant)].Endpoint),
		"adform":     adform.NewAdformLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderAdform)].Endpoint),
		"sovrn":      sovrn.NewSovrnLegacyAdapter(adapters.DefaultHTTPAdapterConfig, cfg.Adapters[string(openrtb_ext.BidderSovrn)].Endpoint),
	}
}

type Router struct {
	*httprouter.Router
	MetricsEngine   *metricsConf.DetailedMetricsEngine
	ParamsValidator openrtb_ext.BidderParamValidator
	Shutdown        func()
}

func New(cfg *config.Configuration, rateConvertor *currency.RateConverter) (r *Router, err error) {
	const schemaDirectory = "./static/bidder-params"
	const infoDirectory = "./static/bidder-info"

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

	generalHttpClient := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:     cfg.Client.MaxConnsPerHost,
			MaxIdleConns:        cfg.Client.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.Client.MaxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(cfg.Client.IdleConnTimeout) * time.Second,
			TLSClientConfig:     &tls.Config{RootCAs: certPool},
		},
	}

	cacheHttpClient := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:     cfg.CacheClient.MaxConnsPerHost,
			MaxIdleConns:        cfg.CacheClient.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.CacheClient.MaxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(cfg.CacheClient.IdleConnTimeout) * time.Second,
		},
	}

	// Hack because of how legacy handles districtm
	legacyBidderList := openrtb_ext.CoreBidderNames()
	legacyBidderList = append(legacyBidderList, openrtb_ext.BidderName("districtm"))

	// Metrics engine
	r.MetricsEngine = metricsConf.NewMetricsEngine(cfg, legacyBidderList)
	db, shutdown, fetcher, ampFetcher, accounts, categoriesFetcher, videoFetcher := storedRequestsConf.NewStoredRequests(cfg, r.MetricsEngine, generalHttpClient, r.Router)
	// todo(zachbadgett): better shutdown
	r.Shutdown = shutdown
	if err := loadDataCache(cfg, db); err != nil {
		return nil, fmt.Errorf("Prebid Server could not load data cache: %v", err)
	}

	pbsAnalytics := analyticsConf.NewPBSAnalytics(&cfg.Analytics)

	paramsValidator, err := openrtb_ext.NewBidderParamsValidator(schemaDirectory)
	if err != nil {
		glog.Fatalf("Failed to create the bidder params validator. %v", err)
	}

	p, _ := filepath.Abs(infoDirectory)
	bidderInfos, err := config.LoadBidderInfoFromDisk(p, cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		glog.Fatal(err)
	}

	activeBidders := exchange.GetActiveBidders(bidderInfos)
	disabledBidders := exchange.GetDisabledBiddersErrorMessages(bidderInfos)

	defaultAliases, defReqJSON := readDefaultRequest(cfg.DefReqConfig)
	if err := validateDefaultAliases(defaultAliases); err != nil {
		glog.Fatal(err)
	}

	syncers := usersyncers.NewSyncerMap(cfg)
	gvlVendorIDs := bidderInfos.ToGVLVendorIDMap()
	gdprPerms := gdpr.NewPermissions(context.Background(), cfg.GDPR, gvlVendorIDs, generalHttpClient)

	exchanges = newExchangeMap(cfg)
	cacheClient := pbc.NewClient(cacheHttpClient, &cfg.CacheURL, &cfg.ExtCacheURL, r.MetricsEngine)

	adapters, adaptersErrs := exchange.BuildAdapters(generalHttpClient, cfg, bidderInfos, r.MetricsEngine)
	if len(adaptersErrs) > 0 {
		errs := errortypes.NewAggregateError("Failed to initialize adapters", adaptersErrs)
		glog.Fatalf("%v", errs)
	}

	theExchange := exchange.NewExchange(adapters, cacheClient, cfg, r.MetricsEngine, bidderInfos, gdprPerms, rateConvertor, categoriesFetcher)

	openrtbEndpoint, err := openrtb2.NewEndpoint(theExchange, paramsValidator, fetcher, accounts, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, activeBidders)
	if err != nil {
		glog.Fatalf("Failed to create the openrtb2 endpoint handler. %v", err)
	}

	ampEndpoint, err := openrtb2.NewAmpEndpoint(theExchange, paramsValidator, ampFetcher, accounts, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, activeBidders)
	if err != nil {
		glog.Fatalf("Failed to create the amp endpoint handler. %v", err)
	}

	videoEndpoint, err := openrtb2.NewVideoEndpoint(theExchange, paramsValidator, fetcher, videoFetcher, accounts, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, activeBidders, cacheClient)
	if err != nil {
		glog.Fatalf("Failed to create the video endpoint handler. %v", err)
	}

	requestTimeoutHeaders := config.RequestTimeoutHeaders{}
	if cfg.RequestTimeoutHeaders != requestTimeoutHeaders {
		videoEndpoint = aspects.QueuedRequestTimeout(videoEndpoint, cfg.RequestTimeoutHeaders, r.MetricsEngine, metrics.ReqTypeVideo)
	}

	r.POST("/auction", endpoints.Auction(cfg, syncers, gdprPerms, r.MetricsEngine, dataCache, exchanges))
	r.POST("/openrtb2/auction", openrtbEndpoint)
	r.POST("/openrtb2/video", videoEndpoint)
	r.GET("/openrtb2/amp", ampEndpoint)
	r.GET("/info/bidders", infoEndpoints.NewBiddersEndpoint(bidderInfos, defaultAliases))
	r.GET("/info/bidders/:bidderName", infoEndpoints.NewBiddersDetailEndpoint(bidderInfos, cfg.Adapters, defaultAliases))
	r.GET("/bidders/params", NewJsonDirectoryServer(schemaDirectory, paramsValidator, defaultAliases))
	r.POST("/cookie_sync", endpoints.NewCookieSyncEndpoint(syncers, cfg, gdprPerms, r.MetricsEngine, pbsAnalytics, activeBidders))
	r.GET("/status", endpoints.NewStatusEndpoint(cfg.StatusResponse))
	r.GET("/", serveIndex)
	r.ServeFiles("/static/*filepath", http.Dir("static"))

	// vtrack endpoint
	if cfg.VTrack.Enabled {
		vtrackEndpoint := events.NewVTrackEndpoint(cfg, accounts, cacheClient, bidderInfos)
		r.POST("/vtrack", vtrackEndpoint)
	}

	// event endpoint
	eventEndpoint := events.NewEventEndpoint(cfg, accounts, pbsAnalytics)
	r.GET("/event", eventEndpoint)

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

	return r, nil
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

func validateDefaultAliases(aliases map[string]string) error {
	var errs []error

	for alias := range aliases {
		if openrtb_ext.IsBidderNameReserved(alias) {
			errs = append(errs, fmt.Errorf("alias %s is a reserved bidder name and cannot be used", alias))
		}
	}

	if len(errs) > 0 {
		return errortypes.NewAggregateError("default request alias errors", errs)
	}

	return nil
}
