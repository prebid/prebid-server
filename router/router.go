package router

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/prebid/prebid-server/analytics"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prometheus/client_golang/prometheus"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/endpoints"
	"github.com/prebid/prebid-server/endpoints/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/server/ssl"
	storedRequestsConf "github.com/prebid/prebid-server/stored_requests/config"
	"github.com/prebid/prebid-server/util/sliceutil"
	"github.com/prebid/prebid-server/util/uuidutil"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var (
	g_syncers           map[string]usersync.Syncer
	g_cfg               *config.Configuration
	g_ex                exchange.Exchange
	g_accounts          stored_requests.AccountFetcher
	g_paramsValidator   openrtb_ext.BidderParamValidator
	g_storedReqFetcher  stored_requests.Fetcher
	g_storedRespFetcher stored_requests.Fetcher
	g_gdprPerms         gdpr.Permissions
	g_metrics           metrics.MetricsEngine
	g_analytics         analytics.PBSAnalyticsModule
	g_disabledBidders   map[string]string
	g_categoriesFetcher stored_requests.CategoryFetcher
	g_videoFetcher      stored_requests.Fetcher
	g_activeBidders     map[string]openrtb_ext.BidderName
	g_defReqJSON        []byte
	g_cacheClient       pbc.Client
	g_transport         *http.Transport
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

type Router struct {
	*httprouter.Router
	MetricsEngine   *metricsConf.DetailedMetricsEngine
	ParamsValidator openrtb_ext.BidderParamValidator
	Shutdown        func()
}

var SchemaDirectory = "/home/http/GO_SERVER/dmhbserver/static/bidder-params"
var InfoDirectory = "/home/http/GO_SERVER/dmhbserver/static/bidder-info"

func New(cfg *config.Configuration, rateConvertor *currency.RateConverter) (r *Router, err error) {
	schemaDirectory := SchemaDirectory
	infoDirectory := InfoDirectory
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

	g_transport = getTransport(cfg, certPool)
	generalHttpClient := &http.Client{
		Transport: g_transport,
	}

	/*
		* Add Dialer:
		* Add TLSHandshakeTimeout:
		* MaxConnsPerHost: Max value should be QPS
		* MaxIdleConnsPerHost:
		* ResponseHeaderTimeout: Max Timeout from OW End
		* No Need for MaxIdleConns:
		*

		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			Dial: (&net.Dialer{
				Timeout:   100 * time.Millisecond,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   10 * time.Second,
			MaxIdleConnsPerHost:   (maxIdleConnsPerHost / size), // ideal needs to be defined diff?
			MaxConnsPerHost:       (maxConnPerHost / size),
			ResponseHeaderTimeout: responseHdrTimeout,
		}
	*/

	cacheHttpClient := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost:     cfg.CacheClient.MaxConnsPerHost,
			MaxIdleConns:        cfg.CacheClient.MaxIdleConns,
			MaxIdleConnsPerHost: cfg.CacheClient.MaxIdleConnsPerHost,
			IdleConnTimeout:     time.Duration(cfg.CacheClient.IdleConnTimeout) * time.Second,
		},
	}

	p, _ := filepath.Abs(infoDirectory)
	bidderInfos, err := config.LoadBidderInfoFromDisk(p, cfg.Adapters, openrtb_ext.BuildBidderStringSlice())
	if err != nil {
		return nil, err
	}

	if err := applyBidderInfoConfigOverrides(bidderInfos, cfg.Adapters); err != nil {
		return nil, err
	}

	if err := checkSupportedUserSyncEndpoints(bidderInfos); err != nil {
		return nil, err
	}

	var errs []error
	g_syncers, errs = usersync.BuildSyncers(cfg, bidderInfos)
	if len(errs) > 0 {
		return nil, errortypes.NewAggregateError("user sync", errs)
	}

	syncerKeys := make([]string, 0, len(g_syncers))
	syncerKeysHashSet := map[string]struct{}{}
	for _, syncer := range g_syncers {
		syncerKeysHashSet[syncer.Key()] = struct{}{}
	}
	for k := range syncerKeysHashSet {
		syncerKeys = append(syncerKeys, k)
	}

	g_cfg = cfg
	// Metrics engine
	g_metrics = metricsConf.NewMetricsEngine(cfg, openrtb_ext.CoreBidderNames(), syncerKeys)
	_, g_storedReqFetcher, _, g_accounts, g_categoriesFetcher, g_videoFetcher, g_storedRespFetcher = storedRequestsConf.NewStoredRequests(cfg, g_metrics, generalHttpClient, r.Router)
	// todo(zachbadgett): better shutdown
	// r.Shutdown = shutdown

	g_analytics = analyticsConf.NewPBSAnalytics(&cfg.Analytics)

	g_paramsValidator, err = openrtb_ext.NewBidderParamsValidator(schemaDirectory)
	if err != nil {
		glog.Fatalf("Failed to create the bidder params validator. %v", err)
	}

	g_activeBidders = exchange.GetActiveBidders(bidderInfos)
	g_disabledBidders = exchange.GetDisabledBiddersErrorMessages(bidderInfos)

	defaultAliases, defReqJSON := readDefaultRequest(cfg.DefReqConfig)
	if err := validateDefaultAliases(defaultAliases); err != nil {
		return nil, err
	}

	g_defReqJSON = defReqJSON

	gvlVendorIDs := bidderInfos.ToGVLVendorIDMap()
	g_gdprPerms = gdpr.NewPermissions(context.Background(), cfg.GDPR, gvlVendorIDs, generalHttpClient)

	if cfg.VendorListScheduler.Enabled {
		vendorListScheduler, err := gdpr.GetVendorListScheduler(cfg.VendorListScheduler.Interval, cfg.VendorListScheduler.Timeout, generalHttpClient)
		if err != nil {
			glog.Fatal(err)
		}
		vendorListScheduler.Start()
	}

	g_cacheClient = pbc.NewClient(cacheHttpClient, &cfg.CacheURL, &cfg.ExtCacheURL, g_metrics)

	adapters, adaptersErrs := exchange.BuildAdapters(generalHttpClient, cfg, bidderInfos, g_metrics)
	if len(adaptersErrs) > 0 {
		errs := errortypes.NewAggregateError("Failed to initialize adapters", adaptersErrs)
		return nil, errs
	}

	g_ex = exchange.NewExchange(adapters, g_cacheClient, cfg, g_syncers, g_metrics, bidderInfos, g_gdprPerms, rateConvertor, g_categoriesFetcher)
	/*var uuidGenerator uuidutil.UUIDRandomGenerator
	openrtbEndpoint, err := openrtb2.NewEndpoint(uuidGenerator, theExchange, paramsValidator, fetcher, accounts, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, activeBidders, storedRespFetcher)
	if err != nil {
		glog.Fatalf("Failed to create the openrtb2 endpoint handler. %v", err)
	}

	ampEndpoint, err := openrtb2.NewAmpEndpoint(uuidGenerator, theExchange, paramsValidator, ampFetcher, accounts, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, activeBidders)
	if err != nil {
		glog.Fatalf("Failed to create the amp endpoint handler. %v", err)
	}

	videoEndpoint, err := openrtb2.NewVideoEndpoint(uuidGenerator, theExchange, paramsValidator, fetcher, videoFetcher, accounts, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, activeBidders, cacheClient)
	if err != nil {
		glog.Fatalf("Failed to create the video endpoint handler. %v", err)
	}

	requestTimeoutHeaders := config.RequestTimeoutHeaders{}
	if cfg.RequestTimeoutHeaders != requestTimeoutHeaders {
		videoEndpoint = aspects.QueuedRequestTimeout(videoEndpoint, cfg.RequestTimeoutHeaders, r.MetricsEngine, metrics.ReqTypeVideo)
	}

	r.POST("/openrtb2/auction", openrtbEndpoint)
	r.POST("/openrtb2/video", videoEndpoint)
	r.GET("/openrtb2/amp", ampEndpoint)
	r.GET("/info/bidders", infoEndpoints.NewBiddersEndpoint(bidderInfos, defaultAliases))
	r.GET("/info/bidders/:bidderName", infoEndpoints.NewBiddersDetailEndpoint(bidderInfos, cfg.Adapters, defaultAliases))
	r.GET("/bidders/params", NewJsonDirectoryServer(schemaDirectory, paramsValidator, defaultAliases))
	r.POST("/cookie_sync", endpoints.NewCookieSyncEndpoint(syncersByBidder, cfg, gdprPerms, r.MetricsEngine, pbsAnalytics, activeBidders).Handle)
	r.GET("/status", endpoints.NewStatusEndpoint(cfg.StatusResponse))
	r.GET("/", serveIndex)
	r.Handler("GET", "/version", endpoints.NewVersionEndpoint(version.Ver, version.Rev))
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
	}

	r.GET("/setuid", endpoints.NewSetUIDEndpoint(cfg.HostCookie, syncersByBidder, gdprPerms, pbsAnalytics, r.MetricsEngine))
	r.GET("/getuids", endpoints.NewGetUIDsEndpoint(cfg.HostCookie))
	r.POST("/optout", userSyncDeps.OptOut)
	r.GET("/optout", userSyncDeps.OptOut)*/

	return r, nil
}

func getTransport(cfg *config.Configuration, certPool *x509.CertPool) *http.Transport {
	transport := &http.Transport{
		MaxConnsPerHost: cfg.Client.MaxConnsPerHost,
		IdleConnTimeout: time.Duration(cfg.Client.IdleConnTimeout) * time.Second,
		TLSClientConfig: &tls.Config{RootCAs: certPool},
	}

	if cfg.Client.DialTimeout > 0 {
		transport.Dial = (&net.Dialer{
			Timeout:   time.Duration(cfg.Client.DialTimeout) * time.Millisecond,
			KeepAlive: time.Duration(cfg.Client.DialKeepAlive) * time.Second,
		}).Dial
	}

	if cfg.Client.TLSHandshakeTimeout > 0 {
		transport.TLSHandshakeTimeout = time.Duration(cfg.Client.TLSHandshakeTimeout) * time.Second
	}

	if cfg.Client.ResponseHeaderTimeout > 0 {
		transport.ResponseHeaderTimeout = time.Duration(cfg.Client.ResponseHeaderTimeout) * time.Second
	}

	if cfg.Client.MaxIdleConns > 0 {
		transport.MaxIdleConns = cfg.Client.MaxIdleConns
	}

	if cfg.Client.MaxIdleConnsPerHost > 0 {
		transport.MaxIdleConnsPerHost = cfg.Client.MaxIdleConnsPerHost
	}

	return transport
}

func applyBidderInfoConfigOverrides(bidderInfos config.BidderInfos, adaptersCfg map[string]config.Adapter) error {
	for bidderName, bidderInfo := range bidderInfos {
		// bidder name from bidderInfos is case-sensitive, but bidder name from adaptersCfg
		// is always expressed as lower case. need to adapt for the difference here.
		if adapterCfg, exists := adaptersCfg[strings.ToLower(bidderName)]; exists {
			bidderInfo.Syncer = adapterCfg.Syncer.Override(bidderInfo.Syncer)

			// validate and try to apply the legacy usersync_url configuration in attempt to provide
			// an easier upgrade path. be warned, this will break if the bidder adds a second syncer
			// type and will eventually be removed after we've given hosts enough time to upgrade to
			// the new config.
			if adapterCfg.UserSyncURL != "" {
				if bidderInfo.Syncer == nil {
					return fmt.Errorf("adapters.%s.usersync_url cannot be applied, bidder does not define a user sync", strings.ToLower(bidderName))
				}

				endpointsCount := 0
				if bidderInfo.Syncer.IFrame != nil {
					bidderInfo.Syncer.IFrame.URL = adapterCfg.UserSyncURL
					endpointsCount++
				}
				if bidderInfo.Syncer.Redirect != nil {
					bidderInfo.Syncer.Redirect.URL = adapterCfg.UserSyncURL
					endpointsCount++
				}

				// use Supports as a hint if there are no good defaults provided
				if endpointsCount == 0 {
					if sliceutil.ContainsStringIgnoreCase(bidderInfo.Syncer.Supports, "iframe") {
						bidderInfo.Syncer.IFrame = &config.SyncerEndpoint{URL: adapterCfg.UserSyncURL}
						endpointsCount++
					}
					if sliceutil.ContainsStringIgnoreCase(bidderInfo.Syncer.Supports, "redirect") {
						bidderInfo.Syncer.Redirect = &config.SyncerEndpoint{URL: adapterCfg.UserSyncURL}
						endpointsCount++
					}
				}

				if endpointsCount == 0 {
					return fmt.Errorf("adapters.%s.usersync_url cannot be applied, bidder does not define user sync endpoints and does not define supported endpoints", strings.ToLower(bidderName))
				}

				// if the bidder defines both an iframe and redirect endpoint, we can't be sure which config value to
				// override, and  it wouldn't be both. this is a fatal configuration error.
				if endpointsCount > 1 {
					return fmt.Errorf("adapters.%s.usersync_url cannot be applied, bidder defines multiple user sync endpoints or supports multiple endpoints", strings.ToLower(bidderName))
				}

				// provide a warning that this compatibility layer is temporary
				glog.Warningf("adapters.%s.usersync_url is deprecated and will be removed in a future version, please update to the latest user sync config values", strings.ToLower(bidderName))
			}

			bidderInfos[bidderName] = bidderInfo
		}
	}
	return nil
}

func checkSupportedUserSyncEndpoints(bidderInfos config.BidderInfos) error {
	for name, info := range bidderInfos {
		if info.Syncer == nil {
			continue
		}

		for _, endpoint := range info.Syncer.Supports {
			endpointLower := strings.ToLower(endpoint)
			switch endpointLower {
			case "iframe":
				if info.Syncer.IFrame == nil {
					glog.Warningf("bidder %s supports iframe user sync, but doesn't have a default and must be configured by the host", name)
				}
			case "redirect":
				if info.Syncer.Redirect == nil {
					glog.Warningf("bidder %s supports redirect user sync, but doesn't have a default and must be configured by the host", name)
				}
			default:
				return fmt.Errorf("failed to load bidder info for %s, user sync supported endpoint '%s' is unrecognized", name, endpoint)
			}
		}
	}
	return nil
}

func GetCacheClient() *pbc.Client {
	return &g_cacheClient
}

func GetPrebidCacheURL() string {
	return g_cfg.ExternalURL
}

//OrtbAuctionEndpointWrapper Openwrap wrapper method for calling /openrtb2/auction endpoint
func OrtbAuctionEndpointWrapper(w http.ResponseWriter, r *http.Request) error {
	ortbAuctionEndpoint, err := openrtb2.NewEndpoint(uuidutil.UUIDRandomGenerator{}, g_ex, g_paramsValidator, g_storedReqFetcher, g_accounts, g_cfg, g_metrics, g_analytics, g_disabledBidders, g_defReqJSON, g_activeBidders, g_storedRespFetcher)
	if err != nil {
		return err
	}
	ortbAuctionEndpoint(w, r, nil)
	return nil
}

//VideoAuctionEndpointWrapper Openwrap wrapper method for calling /openrtb2/video endpoint
func VideoAuctionEndpointWrapper(w http.ResponseWriter, r *http.Request) error {
	videoAuctionEndpoint, err := openrtb2.NewCTVEndpoint(g_ex, g_paramsValidator, g_storedReqFetcher, g_videoFetcher, g_accounts, g_cfg, g_metrics, g_analytics, g_disabledBidders, g_defReqJSON, g_activeBidders)
	if err != nil {
		return err
	}
	videoAuctionEndpoint(w, r, nil)
	return nil
}

//GetUIDSWrapper Openwrap wrapper method for calling /getuids endpoint
func GetUIDSWrapper(w http.ResponseWriter, r *http.Request) {
	getUID := endpoints.NewGetUIDsEndpoint(g_cfg.HostCookie)
	getUID(w, r, nil)
}

//SetUIDSWrapper Openwrap wrapper method for calling /setuid endpoint
func SetUIDSWrapper(w http.ResponseWriter, r *http.Request) {
	setUID := endpoints.NewSetUIDEndpoint(g_cfg.HostCookie, g_syncers, g_gdprPerms, g_analytics, g_metrics)
	setUID(w, r, nil)
}

//CookieSync Openwrap wrapper method for calling /cookie_sync endpoint
func CookieSync(w http.ResponseWriter, r *http.Request) {
	cookiesync := endpoints.NewCookieSyncEndpoint(g_syncers, g_cfg, g_gdprPerms, g_metrics, g_analytics, g_activeBidders)
	cookiesync.Handle(w, r, nil)
}

//SyncerMap Returns map of bidder and its usersync info
func SyncerMap() map[string]usersync.Syncer {
	return g_syncers
}

func GetPrometheusGatherer() *prometheus.Registry {
	mEngine, ok := g_metrics.(*metricsConf.DetailedMetricsEngine)
	if !ok || mEngine == nil || mEngine.PrometheusMetrics == nil {
		return nil
	}

	return mEngine.PrometheusMetrics.Gatherer
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
