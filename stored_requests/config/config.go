package config

import (
	"context"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v4/account"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/metrics"
	"github.com/prebid/prebid-server/v4/stored_requests"
	"github.com/prebid/prebid-server/v4/stored_requests/backends/db_fetcher"
	"github.com/prebid/prebid-server/v4/stored_requests/backends/db_provider"
	"github.com/prebid/prebid-server/v4/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v4/stored_requests/backends/file_fetcher"
	"github.com/prebid/prebid-server/v4/stored_requests/backends/http_fetcher"
	"github.com/prebid/prebid-server/v4/stored_requests/caches/memory"
	"github.com/prebid/prebid-server/v4/stored_requests/caches/nil_cache"
	"github.com/prebid/prebid-server/v4/stored_requests/events"
	apiEvents "github.com/prebid/prebid-server/v4/stored_requests/events/api"
	databaseEvents "github.com/prebid/prebid-server/v4/stored_requests/events/database"
	httpEvents "github.com/prebid/prebid-server/v4/stored_requests/events/http"
	"github.com/prebid/prebid-server/v4/util/task"
	"github.com/prebid/prebid-server/v4/util/timeutil"
)

// CreateStoredRequests returns three things:
//
// 1. A Fetcher which can be used to get Stored Requests
// 2. A function which should be called on shutdown for graceful cleanups.
//
// If any errors occur, the program will exit with an error message.
// It probably means you have a bad config or networking issue.
//
// As a side-effect, it will add some endpoints to the router if the config calls for it.
// In the future we should look for ways to simplify this so that it's not doing two things.
func CreateStoredRequests(cfg *config.StoredRequests, metricsEngine metrics.MetricsEngine, client *http.Client, router *httprouter.Router, provider db_provider.DbProvider) (fetcher stored_requests.AllFetcher, shutdown func()) {
	return createLegacyCachedStoredRequests(cfg, metricsEngine, client, router, provider)
}

func createStoredRequestSource(cfg *config.StoredRequests, client *http.Client, provider db_provider.DbProvider) stored_requests.AllFetcher {
	return newFetcher(cfg, client, provider)
}

func prepareStoredRequestsProvider(cfg *config.StoredRequests, provider db_provider.DbProvider) db_provider.DbProvider {
	if cfg.Database.ConnectionInfo.Database != "" {
		if provider == nil {
			logger.Infof("Connecting to Database for Stored %s. Driver=%s, DB=%s, host=%s, port=%d, user=%s",
				cfg.DataType(),
				cfg.Database.ConnectionInfo.Driver,
				cfg.Database.ConnectionInfo.Database,
				cfg.Database.ConnectionInfo.Host,
				cfg.Database.ConnectionInfo.Port,
				cfg.Database.ConnectionInfo.Username)
			provider = db_provider.NewDbProvider(cfg.DataType(), cfg.Database.ConnectionInfo)
		}

		// Error out if config is trying to use multiple database connections for different stored requests (not supported yet)
		if provider.Config() != cfg.Database.ConnectionInfo {
			logger.Fatalf("Multiple database connection settings found in config, only a single database connection is currently supported.")
		}
	}
	return provider
}

func createRawStoredRequests(cfg *config.StoredRequests, client *http.Client, provider db_provider.DbProvider) (fetcher stored_requests.AllFetcher, shutdown func()) {
	provider = prepareStoredRequestsProvider(cfg, provider)
	fetcher = createStoredRequestSource(cfg, client, provider)
	shutdown = func() {
		if provider == nil {
			return
		}

		if err := provider.Close(); err != nil {
			logger.Errorf("Error closing DB connection: %v", err)
		}
	}
	return
}

func createLegacyCachedStoredRequests(cfg *config.StoredRequests, metricsEngine metrics.MetricsEngine, client *http.Client, router *httprouter.Router, provider db_provider.DbProvider) (fetcher stored_requests.AllFetcher, shutdown func()) {
	provider = prepareStoredRequestsProvider(cfg, provider)

	eventProducers := newEventProducers(cfg, client, provider, metricsEngine, router)
	fetcher = createStoredRequestSource(cfg, client, provider)

	var shutdown1 func()

	if cfg.InMemoryCache.Type != "" {
		cache := newCache(cfg)
		fetcher = stored_requests.WithCache(fetcher, cache, metricsEngine)
		shutdown1 = addListeners(cache, eventProducers)
	}

	shutdown = func() {
		if shutdown1 != nil {
			shutdown1()
		}

		if provider == nil {
			return
		}

		if err := provider.Close(); err != nil {
			logger.Errorf("Error closing DB connection: %v", err)
		}
	}

	return
}

// NewStoredRequests returns:
//
// 1. A function which should be called on shutdown for graceful cleanups.
// 2. A Fetcher which can be used to get Stored Requests for /openrtb2/auction
// 3. A Fetcher which can be used to get Stored Requests for /openrtb2/amp
// 4. A Fetcher which can be used to get Account data
// 5. A Fetcher which can be used to get Category Mapping data
// 6. A Fetcher which can be used to get Stored Requests for /openrtb2/video
//
// If any errors occur, the program will exit with an error message.
// It probably means you have a bad config or networking issue.
//
// As a side-effect, it will add some endpoints to the router if the config calls for it.
// In the future we should look for ways to simplify this so that it's not doing two things.
func NewStoredRequests(cfg *config.Configuration, metricsEngine metrics.MetricsEngine, client *http.Client, router *httprouter.Router) (shutdown func(),
	fetcher stored_requests.Fetcher,
	ampFetcher stored_requests.Fetcher,
	accountsFetcher stored_requests.AccountFetcher,
	categoriesFetcher stored_requests.CategoryFetcher,
	videoFetcher stored_requests.Fetcher,
	storedRespFetcher stored_requests.Fetcher) {

	var provider db_provider.DbProvider

	fetcher1, shutdown1 := CreateStoredRequests(&cfg.StoredRequests, metricsEngine, client, router, provider)
	fetcher2, shutdown2 := CreateStoredRequests(&cfg.StoredRequestsAMP, metricsEngine, client, router, provider)
	fetcher3, shutdown3 := CreateStoredRequests(&cfg.CategoryMapping, metricsEngine, client, router, provider)
	fetcher4, shutdown4 := CreateStoredRequests(&cfg.StoredVideo, metricsEngine, client, router, provider)
	var fetcher5 stored_requests.AllFetcher
	var shutdown5 func()
	var accountSource account.Source
	if cfg.Accounts.V2Enabled {
		accountSource, shutdown5 = createAccountSource(&cfg.Accounts, client, provider)
	} else {
		fetcher5, shutdown5 = CreateStoredRequests(&cfg.Accounts, metricsEngine, client, router, provider)
	}
	fetcher6, shutdown6 := CreateStoredRequests(&cfg.StoredResponses, metricsEngine, client, router, provider)

	fetcher = fetcher1.(stored_requests.Fetcher)
	ampFetcher = fetcher2.(stored_requests.Fetcher)
	categoriesFetcher = fetcher3.(stored_requests.CategoryFetcher)
	videoFetcher = fetcher4.(stored_requests.Fetcher)
	if !cfg.Accounts.V2Enabled {
		accountsFetcher = fetcher5.(stored_requests.AccountFetcher)
	}
	storedRespFetcher = fetcher6.(stored_requests.Fetcher)

	// Fetchers 2.0: when enabled, use the new typed account source/fetcher path.
	// With v2_enabled=false the legacy byte-cache path is used unchanged.
	if cfg.Accounts.V2Enabled {
		v2Accounts, err := account.NewFetcherAccountFetcher(accountSource, cfg.Accounts.CacheV2, cfg.AccountDefaultsJSON(), &timeutil.RealTime{}, metricsEngine)
		if err != nil {
			logger.Fatalf("Failed to initialize Fetchers 2.0 account fetcher: %v", err)
		}
		accountsFetcher = v2Accounts
	}

	shutdown = func() {
		shutdown1()
		shutdown2()
		shutdown3()
		shutdown4()
		shutdown5()
		shutdown6()
	}

	return
}

func createAccountSource(cfg *config.StoredRequests, client *http.Client, provider db_provider.DbProvider) (source account.Source, shutdown func()) {
	provider = prepareStoredRequestsProvider(cfg, provider)
	source = newAccountSource(cfg, client)
	shutdown = func() {
		if provider == nil {
			return
		}
		if err := provider.Close(); err != nil {
			logger.Errorf("Error closing DB connection: %v", err)
		}
	}
	return
}

func addListeners(cache stored_requests.Cache, eventProducers []events.EventProducer) (shutdown func()) {
	listeners := make([]*events.EventListener, 0, len(eventProducers))

	for _, ep := range eventProducers {
		listener := events.SimpleEventListener()
		go listener.Listen(cache, ep)
		listeners = append(listeners, listener)
	}

	return func() {
		for _, l := range listeners {
			l.Stop()
		}
	}
}

func newFetcher(cfg *config.StoredRequests, client *http.Client, provider db_provider.DbProvider) (fetcher stored_requests.AllFetcher) {
	idList := make(stored_requests.MultiFetcher, 0, 3)

	if cfg.Files.Enabled {
		fFetcher := newFilesystem(cfg.DataType(), cfg.Files.Path)
		idList = append(idList, fFetcher)
	}
	if cfg.Database.FetcherQueries.QueryTemplate != "" {
		logger.Infof("Loading Stored %s data via Database.\nQuery: %s", cfg.DataType(), cfg.Database.FetcherQueries.QueryTemplate)
		idList = append(idList, db_fetcher.NewFetcher(provider, cfg.Database.FetcherQueries.QueryTemplate, cfg.Database.FetcherQueries.QueryTemplate))
	} else if cfg.Database.CacheInitialization.Query != "" && cfg.Database.PollUpdates.Query != "" {
		//in this case data will be loaded to cache via poll for updates event
		idList = append(idList, empty_fetcher.EmptyFetcher{})
	}
	if cfg.HTTP.Endpoint != "" {
		logger.Infof("Loading Stored %s data via HTTP. endpoint=%s", cfg.DataType(), cfg.HTTP.Endpoint)
		idList = append(idList, http_fetcher.NewFetcher(client, cfg.HTTP.Endpoint, cfg.HTTP.UseRfcCompliantBuilder))
	}

	fetcher = consolidate(cfg.DataType(), idList)
	return
}

func newAccountSource(cfg *config.StoredRequests, client *http.Client) account.Source {
	var source account.Source
	if cfg.Files.Enabled {
		logger.Infof("Loading Fetchers 2.0 Account data from filesystem at path %s", cfg.Files.Path)
		fileSource, err := account.NewFileSource(cfg.Files.Path)
		if err != nil {
			logger.Fatalf("Failed to create a Fetchers 2.0 Account FileSource: %v", err)
		}
		source = fileSource
	}
	if cfg.Database.ConnectionInfo.Database != "" {
		logger.Fatalf("Fetchers 2.0 account database source is not supported. Use accounts.filesystem or accounts.http.")
	}
	if cfg.HTTP.Endpoint != "" {
		if source != nil {
			logger.Fatalf("Fetchers 2.0 account source supports exactly one backend. Configure either accounts.filesystem or accounts.http, not both.")
		}
		httpSource, err := account.NewHTTPSource(client, cfg.HTTP.Endpoint, cfg.HTTP.UseRfcCompliantBuilder)
		if err != nil {
			logger.Fatalf("Failed to create a Fetchers 2.0 Account HTTPSource: %v", err)
		}
		source = httpSource
	}
	if source == nil {
		logger.Warnf("No Stored %s support configured. If you need this, check your app config", cfg.DataType())
		return account.EmptySource{}
	}
	return source
}

func newCache(cfg *config.StoredRequests) stored_requests.Cache {
	cache := stored_requests.Cache{
		Requests:  &nil_cache.NilCache{},
		Imps:      &nil_cache.NilCache{},
		Responses: &nil_cache.NilCache{},
		Accounts:  &nil_cache.NilCache{},
	}
	switch {
	case cfg.InMemoryCache.Type == "none":
		logger.Warnf("No %s cache configured. The %s Fetcher backend will be used for all data requests", cfg.DataType(), cfg.DataType())
	case cfg.DataType() == config.AccountDataType:
		cache.Accounts = memory.NewCache(cfg.InMemoryCache.Size, cfg.InMemoryCache.TTL, "Accounts")
	default:
		cache.Requests = memory.NewCache(cfg.InMemoryCache.RequestCacheSize, cfg.InMemoryCache.TTL, "Requests")
		cache.Imps = memory.NewCache(cfg.InMemoryCache.ImpCacheSize, cfg.InMemoryCache.TTL, "Imps")
		cache.Responses = memory.NewCache(cfg.InMemoryCache.RespCacheSize, cfg.InMemoryCache.TTL, "Responses")
	}
	return cache
}

func newEventProducers(cfg *config.StoredRequests, client *http.Client, provider db_provider.DbProvider, metricsEngine metrics.MetricsEngine, router *httprouter.Router) (eventProducers []events.EventProducer) {
	if cfg.CacheEvents.Enabled {
		eventProducers = append(eventProducers, newEventsAPI(router, cfg.CacheEvents.Endpoint))
	}
	if cfg.HTTPEvents.RefreshRate != 0 && cfg.HTTPEvents.Endpoint != "" {
		eventProducers = append(eventProducers, newHttpEvents(client, cfg.HTTPEvents.TimeoutDuration(), cfg.HTTPEvents.RefreshRateDuration(), cfg.HTTPEvents.Endpoint))
	}
	if cfg.Database.CacheInitialization.Query != "" {
		dbEventCfg := databaseEvents.DatabaseEventProducerConfig{
			Provider:           provider,
			RequestType:        cfg.DataType(),
			CacheInitQuery:     cfg.Database.CacheInitialization.Query,
			CacheInitTimeout:   time.Duration(cfg.Database.CacheInitialization.Timeout) * time.Millisecond,
			CacheUpdateQuery:   cfg.Database.PollUpdates.Query,
			CacheUpdateTimeout: time.Duration(cfg.Database.PollUpdates.Timeout) * time.Millisecond,
			MetricsEngine:      metricsEngine,
		}
		dbEventProducer := databaseEvents.NewDatabaseEventProducer(dbEventCfg)
		fetchInterval := time.Duration(cfg.Database.PollUpdates.RefreshRate) * time.Second
		dbEventTickerTask := task.NewTickerTask(fetchInterval, dbEventProducer)
		dbEventTickerTask.Start()
		eventProducers = append(eventProducers, dbEventProducer)
	}
	return
}

func newEventsAPI(router *httprouter.Router, endpoint string) events.EventProducer {
	producer, handler := apiEvents.NewEventsAPI()
	router.POST(endpoint, handler)
	router.DELETE(endpoint, handler)
	return producer
}

func newHttpEvents(client *http.Client, timeout time.Duration, refreshRate time.Duration, endpoint string) events.EventProducer {
	ctxProducer := func() (ctx context.Context, canceller func()) {
		return context.WithTimeout(context.Background(), timeout)
	}
	return httpEvents.NewHTTPEvents(client, endpoint, ctxProducer, refreshRate)
}

func newFilesystem(dataType config.DataType, configPath string) stored_requests.AllFetcher {
	logger.Infof("Loading Stored %s data from filesystem at path %s", dataType, configPath)
	fetcher, err := file_fetcher.NewFileFetcher(configPath)
	if err != nil {
		logger.Fatalf("Failed to create a %s FileFetcher: %v", dataType, err)
	}
	return fetcher
}

// consolidate returns a single Fetcher from an array of fetchers of any size.
func consolidate(dataType config.DataType, fetchers []stored_requests.AllFetcher) stored_requests.AllFetcher {
	if len(fetchers) == 0 {
		switch dataType {
		case config.RequestDataType:
			logger.Warnf("No Stored Request support configured. request.imp[i].ext.prebid.storedrequest will be ignored. If you need this, check your app config")
		default:
			logger.Warnf("No Stored %s support configured. If you need this, check your app config", dataType)
		}
		return empty_fetcher.EmptyFetcher{}
	} else if len(fetchers) == 1 {
		return fetchers[0]
	} else {
		return stored_requests.MultiFetcher(fetchers)
	}
}
