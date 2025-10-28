package config

import (
	"context"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/v3/metrics"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/db_fetcher"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/db_provider"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/file_fetcher"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/http_fetcher"
	"github.com/prebid/prebid-server/v3/stored_requests/caches/memory"
	"github.com/prebid/prebid-server/v3/stored_requests/caches/nil_cache"
	"github.com/prebid/prebid-server/v3/stored_requests/events"
	apiEvents "github.com/prebid/prebid-server/v3/stored_requests/events/api"
	databaseEvents "github.com/prebid/prebid-server/v3/stored_requests/events/database"
	httpEvents "github.com/prebid/prebid-server/v3/stored_requests/events/http"
	"github.com/prebid/prebid-server/v3/util/task"
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
	// Create database connection if given options for one
	if cfg.Database.ConnectionInfo.Database != "" {
		if provider == nil {
			glog.Infof("Connecting to Database for Stored %s. Driver=%s, DB=%s, host=%s, port=%d, user=%s",
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
			glog.Fatal("Multiple database connection settings found in config, only a single database connection is currently supported.")
		}
	}

	eventProducers := newEventProducers(cfg, client, provider, metricsEngine, router)
	fetcher = newFetcher(cfg, client, provider)

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
			glog.Errorf("Error closing DB connection: %v", err)
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
	fetcher5, shutdown5 := CreateStoredRequests(&cfg.Accounts, metricsEngine, client, router, provider)
	fetcher6, shutdown6 := CreateStoredRequests(&cfg.StoredResponses, metricsEngine, client, router, provider)

	fetcher = fetcher1.(stored_requests.Fetcher)
	ampFetcher = fetcher2.(stored_requests.Fetcher)
	categoriesFetcher = fetcher3.(stored_requests.CategoryFetcher)
	videoFetcher = fetcher4.(stored_requests.Fetcher)
	accountsFetcher = fetcher5.(stored_requests.AccountFetcher)
	storedRespFetcher = fetcher6.(stored_requests.Fetcher)

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
		glog.Infof("Loading Stored %s data via Database.\nQuery: %s", cfg.DataType(), cfg.Database.FetcherQueries.QueryTemplate)
		idList = append(idList, db_fetcher.NewFetcher(provider,
			cfg.Database.FetcherQueries.QueryTemplate, cfg.Database.FetcherQueries.QueryTemplate))
	} else if cfg.Database.CacheInitialization.Query != "" && cfg.Database.PollUpdates.Query != "" {
		//in this case data will be loaded to cache via poll for updates event
		idList = append(idList, empty_fetcher.EmptyFetcher{})
	}
	if cfg.HTTP.Endpoint != "" {
		glog.Infof("Loading Stored %s data via HTTP. endpoint=%s", cfg.DataType(), cfg.HTTP.Endpoint)
		idList = append(idList, http_fetcher.NewFetcher(client, cfg.HTTP.Endpoint, cfg.HTTP.UseRfcCompliantBuilder))
	}

	fetcher = consolidate(cfg.DataType(), idList)
	return
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
		glog.Warningf("No %s cache configured. The %s Fetcher backend will be used for all data requests", cfg.DataType(), cfg.DataType())
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
	glog.Infof("Loading Stored %s data from filesystem at path %s", dataType, configPath)
	fetcher, err := file_fetcher.NewFileFetcher(configPath)
	if err != nil {
		glog.Fatalf("Failed to create a %s FileFetcher: %v", dataType, err)
	}
	return fetcher
}

// consolidate returns a single Fetcher from an array of fetchers of any size.
func consolidate(dataType config.DataType, fetchers []stored_requests.AllFetcher) stored_requests.AllFetcher {
	if len(fetchers) == 0 {
		switch dataType {
		case config.RequestDataType:
			glog.Warning("No Stored Request support configured. request.imp[i].ext.prebid.storedrequest will be ignored. If you need this, check your app config")
		default:
			glog.Warningf("No Stored %s support configured. If you need this, check your app config", dataType)
		}
		return empty_fetcher.EmptyFetcher{}
	} else if len(fetchers) == 1 {
		return fetchers[0]
	} else {
		return stored_requests.MultiFetcher(fetchers)
	}
}
