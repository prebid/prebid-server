package config

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/prebid/prebid-server/metrics"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests"
	"github.com/prebid/prebid-server/stored_requests/backends/db_fetcher"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/stored_requests/backends/file_fetcher"
	"github.com/prebid/prebid-server/stored_requests/backends/http_fetcher"
	"github.com/prebid/prebid-server/stored_requests/caches/memory"
	"github.com/prebid/prebid-server/stored_requests/caches/nil_cache"
	"github.com/prebid/prebid-server/stored_requests/events"
	apiEvents "github.com/prebid/prebid-server/stored_requests/events/api"
	httpEvents "github.com/prebid/prebid-server/stored_requests/events/http"
	postgresEvents "github.com/prebid/prebid-server/stored_requests/events/postgres"
	"github.com/prebid/prebid-server/util/task"
)

// This gets set to the connection string used when a database connection is made. We only support a single
// database currently, so all fetchers need to share the same db connection for now.
type dbConnection struct {
	conn string
	db   *sql.DB
}

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
func CreateStoredRequests(cfg *config.StoredRequests, metricsEngine metrics.MetricsEngine, client *http.Client, router *httprouter.Router, dbc *dbConnection) (fetcher stored_requests.AllFetcher, shutdown func()) {
	// Create database connection if given options for one
	if cfg.Postgres.ConnectionInfo.Database != "" {
		conn := cfg.Postgres.ConnectionInfo.ConnString()

		if dbc.conn == "" {
			glog.Infof("Connecting to Postgres for Stored %s. DB=%s, host=%s, port=%d, user=%s",
				cfg.DataType(),
				cfg.Postgres.ConnectionInfo.Database,
				cfg.Postgres.ConnectionInfo.Host,
				cfg.Postgres.ConnectionInfo.Port,
				cfg.Postgres.ConnectionInfo.Username)
			db := newPostgresDB(cfg.DataType(), cfg.Postgres.ConnectionInfo)
			dbc.conn = conn
			dbc.db = db
		}

		// Error out if config is trying to use multiple database connections for different stored requests (not supported yet)
		if conn != dbc.conn {
			glog.Fatal("Multiple database connection settings found in config, only a single database connection is currently supported.")
		}
	}

	eventProducers := newEventProducers(cfg, client, dbc.db, metricsEngine, router)
	fetcher = newFetcher(cfg, client, dbc.db)

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
		if dbc.db != nil {
			db := dbc.db
			dbc.db = nil
			dbc.conn = ""
			if err := db.Close(); err != nil {
				glog.Errorf("Error closing DB connection: %v", err)
			}
		}
	}

	return
}

// NewStoredRequests returns five things:
//
// 1. A DB connection, if one was created. This may be nil.
// 2. A function which should be called on shutdown for graceful cleanups.
// 3. A Fetcher which can be used to get Stored Requests for /openrtb2/auction
// 4. A Fetcher which can be used to get Stored Requests for /openrtb2/amp
// 5. A Fetcher which can be used to get Category Mapping data
// 6. A Fetcher which can be used to get Stored Requests for /openrtb2/video
//
// If any errors occur, the program will exit with an error message.
// It probably means you have a bad config or networking issue.
//
// As a side-effect, it will add some endpoints to the router if the config calls for it.
// In the future we should look for ways to simplify this so that it's not doing two things.
func NewStoredRequests(cfg *config.Configuration, metricsEngine metrics.MetricsEngine, client *http.Client, router *httprouter.Router) (db *sql.DB, shutdown func(), fetcher stored_requests.Fetcher, ampFetcher stored_requests.Fetcher, accountsFetcher stored_requests.AccountFetcher, categoriesFetcher stored_requests.CategoryFetcher, videoFetcher stored_requests.Fetcher) {
	// TODO: Switch this to be set in config defaults
	//if cfg.CategoryMapping.CacheEvents.Enabled && cfg.CategoryMapping.CacheEvents.Endpoint == "" {
	//	cfg.CategoryMapping.CacheEvents.Endpoint = "/storedrequest/categorymapping"
	//}

	var dbc dbConnection

	fetcher1, shutdown1 := CreateStoredRequests(&cfg.StoredRequests, metricsEngine, client, router, &dbc)
	fetcher2, shutdown2 := CreateStoredRequests(&cfg.StoredRequestsAMP, metricsEngine, client, router, &dbc)
	fetcher3, shutdown3 := CreateStoredRequests(&cfg.CategoryMapping, metricsEngine, client, router, &dbc)
	fetcher4, shutdown4 := CreateStoredRequests(&cfg.StoredVideo, metricsEngine, client, router, &dbc)
	fetcher5, shutdown5 := CreateStoredRequests(&cfg.Accounts, metricsEngine, client, router, &dbc)

	db = dbc.db

	fetcher = fetcher1.(stored_requests.Fetcher)
	ampFetcher = fetcher2.(stored_requests.Fetcher)
	categoriesFetcher = fetcher3.(stored_requests.CategoryFetcher)
	videoFetcher = fetcher4.(stored_requests.Fetcher)
	accountsFetcher = fetcher5.(stored_requests.AccountFetcher)

	shutdown = func() {
		shutdown1()
		shutdown2()
		shutdown3()
		shutdown4()
		shutdown5()
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

func newFetcher(cfg *config.StoredRequests, client *http.Client, db *sql.DB) (fetcher stored_requests.AllFetcher) {
	idList := make(stored_requests.MultiFetcher, 0, 3)

	if cfg.Files.Enabled {
		fFetcher := newFilesystem(cfg.DataType(), cfg.Files.Path)
		idList = append(idList, fFetcher)
	}
	if cfg.Postgres.FetcherQueries.QueryTemplate != "" {
		glog.Infof("Loading Stored %s data via Postgres.\nQuery: %s", cfg.DataType(), cfg.Postgres.FetcherQueries.QueryTemplate)
		idList = append(idList, db_fetcher.NewFetcher(db, cfg.Postgres.FetcherQueries.MakeQuery))
	} else if cfg.Postgres.CacheInitialization.Query != "" && cfg.Postgres.PollUpdates.Query != "" {
		//in this case data will be loaded to cache via poll for updates event
		idList = append(idList, empty_fetcher.EmptyFetcher{})
	}
	if cfg.HTTP.Endpoint != "" {
		glog.Infof("Loading Stored %s data via HTTP. endpoint=%s", cfg.DataType(), cfg.HTTP.Endpoint)
		idList = append(idList, http_fetcher.NewFetcher(client, cfg.HTTP.Endpoint))
	}

	fetcher = consolidate(cfg.DataType(), idList)
	return
}

func newCache(cfg *config.StoredRequests) stored_requests.Cache {
	cache := stored_requests.Cache{&nil_cache.NilCache{}, &nil_cache.NilCache{}, &nil_cache.NilCache{}}
	switch {
	case cfg.InMemoryCache.Type == "none":
		glog.Warningf("No %s cache configured. The %s Fetcher backend will be used for all data requests", cfg.DataType(), cfg.DataType())
	case cfg.DataType() == config.AccountDataType:
		cache.Accounts = memory.NewCache(cfg.InMemoryCache.Size, cfg.InMemoryCache.TTL, "Accounts")
	default:
		cache.Requests = memory.NewCache(cfg.InMemoryCache.RequestCacheSize, cfg.InMemoryCache.TTL, "Requests")
		cache.Imps = memory.NewCache(cfg.InMemoryCache.ImpCacheSize, cfg.InMemoryCache.TTL, "Imps")
	}
	return cache
}

func newEventProducers(cfg *config.StoredRequests, client *http.Client, db *sql.DB, metricsEngine metrics.MetricsEngine, router *httprouter.Router) (eventProducers []events.EventProducer) {
	if cfg.CacheEvents.Enabled {
		eventProducers = append(eventProducers, newEventsAPI(router, cfg.CacheEvents.Endpoint))
	}
	if cfg.HTTPEvents.RefreshRate != 0 && cfg.HTTPEvents.Endpoint != "" {
		eventProducers = append(eventProducers, newHttpEvents(client, cfg.HTTPEvents.TimeoutDuration(), cfg.HTTPEvents.RefreshRateDuration(), cfg.HTTPEvents.Endpoint))
	}
	if cfg.Postgres.CacheInitialization.Query != "" {
		pgEventCfg := postgresEvents.PostgresEventProducerConfig{
			DB:                 db,
			RequestType:        cfg.DataType(),
			CacheInitQuery:     cfg.Postgres.CacheInitialization.Query,
			CacheInitTimeout:   time.Duration(cfg.Postgres.CacheInitialization.Timeout) * time.Millisecond,
			CacheUpdateQuery:   cfg.Postgres.PollUpdates.Query,
			CacheUpdateTimeout: time.Duration(cfg.Postgres.PollUpdates.Timeout) * time.Millisecond,
			MetricsEngine:      metricsEngine,
		}
		pgEventProducer := postgresEvents.NewPostgresEventProducer(pgEventCfg)
		fetchInterval := time.Duration(cfg.Postgres.PollUpdates.RefreshRate) * time.Second
		pgEventTickerTask := task.NewTickerTask(fetchInterval, pgEventProducer)
		pgEventTickerTask.Start()
		eventProducers = append(eventProducers, pgEventProducer)
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

func newPostgresDB(dataType config.DataType, cfg config.PostgresConnection) *sql.DB {
	db, err := sql.Open("postgres", cfg.ConnString())
	if err != nil {
		glog.Fatalf("Failed to open %s postgres connection: %v", dataType, err)
	}

	if err := db.Ping(); err != nil {
		glog.Fatalf("Failed to ping %s postgres: %v", dataType, err)
	}

	return db
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
