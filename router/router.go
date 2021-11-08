package router

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	analyticsConf "github.com/prebid/prebid-server/analytics/config"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/currency"
	"github.com/prebid/prebid-server/endpoints"
	"github.com/prebid/prebid-server/endpoints/events"
	infoEndpoints "github.com/prebid/prebid-server/endpoints/info"
	"github.com/prebid/prebid-server/endpoints/openrtb2"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/exchange"
	"github.com/prebid/prebid-server/gdpr"
	"github.com/prebid/prebid-server/metrics"
	metricsConf "github.com/prebid/prebid-server/metrics/config"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/prebid/prebid-server/pbs"
	pbc "github.com/prebid/prebid-server/prebid_cache_client"
	"github.com/prebid/prebid-server/router/aspects"
	"github.com/prebid/prebid-server/server/ssl"
	storedRequestsConf "github.com/prebid/prebid-server/stored_requests/config"
	"github.com/prebid/prebid-server/usersync"
	"github.com/prebid/prebid-server/util/sliceutil"
	"github.com/prebid/prebid-server/util/uuidutil"
	"github.com/prebid/prebid-server/version"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
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

	syncersByBidder, errs := usersync.BuildSyncers(cfg, bidderInfos)
	if len(errs) > 0 {
		return nil, errortypes.NewAggregateError("user sync", errs)
	}

	syncerKeys := make([]string, 0, len(syncersByBidder))
	syncerKeysHashSet := map[string]struct{}{}
	for _, syncer := range syncersByBidder {
		syncerKeysHashSet[syncer.Key()] = struct{}{}
	}
	for k := range syncerKeysHashSet {
		syncerKeys = append(syncerKeys, k)
	}

	// Metrics engine
	r.MetricsEngine = metricsConf.NewMetricsEngine(cfg, openrtb_ext.CoreBidderNames(), syncerKeys)
	shutdown, fetcher, ampFetcher, accounts, categoriesFetcher, videoFetcher := storedRequestsConf.NewStoredRequests(cfg, r.MetricsEngine, generalHttpClient, r.Router)
	// todo(zachbadgett): better shutdown
	r.Shutdown = shutdown

	pbsAnalytics := analyticsConf.NewPBSAnalytics(&cfg.Analytics)

	paramsValidator, err := openrtb_ext.NewBidderParamsValidator(schemaDirectory)
	if err != nil {
		glog.Fatalf("Failed to create the bidder params validator. %v", err)
	}

	activeBidders := exchange.GetActiveBidders(bidderInfos)
	disabledBidders := exchange.GetDisabledBiddersErrorMessages(bidderInfos)

	defaultAliases, defReqJSON := readDefaultRequest(cfg.DefReqConfig)
	if err := validateDefaultAliases(defaultAliases); err != nil {
		return nil, err
	}

	gvlVendorIDs := bidderInfos.ToGVLVendorIDMap()
	gdprPerms := gdpr.NewPermissions(context.Background(), cfg.GDPR, gvlVendorIDs, generalHttpClient)

	cacheClient := pbc.NewClient(cacheHttpClient, &cfg.CacheURL, &cfg.ExtCacheURL, r.MetricsEngine)

	adapters, adaptersErrs := exchange.BuildAdapters(generalHttpClient, cfg, bidderInfos, r.MetricsEngine)
	if len(adaptersErrs) > 0 {
		errs := errortypes.NewAggregateError("Failed to initialize adapters", adaptersErrs)
		return nil, errs
	}

	theExchange := exchange.NewExchange(adapters, cacheClient, cfg, syncersByBidder, r.MetricsEngine, bidderInfos, gdprPerms, rateConvertor, categoriesFetcher)
	var uuidGenerator uuidutil.UUIDRandomGenerator
	openrtbEndpoint, err := openrtb2.NewEndpoint(uuidGenerator, theExchange, paramsValidator, fetcher, accounts, cfg, r.MetricsEngine, pbsAnalytics, disabledBidders, defReqJSON, activeBidders)
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
	r.GET("/optout", userSyncDeps.OptOut)

	return r, nil
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
