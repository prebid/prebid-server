package config

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/PubMatic-OpenWrap/prebid-server/pbsmetrics"

	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/backends/db_fetcher"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/backends/file_fetcher"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/backends/http_fetcher"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/caches/memory"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/caches/nil_cache"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events"
	apiEvents "github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events/api"
	httpEvents "github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events/http"
	postgresEvents "github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events/postgres"
	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
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
func CreateStoredRequests(cfg *config.StoredRequestsSlim, metricsEngine pbsmetrics.MetricsEngine, client *http.Client, router *httprouter.Router, dbc *dbConnection) (fetcher stored_requests.AllFetcher, shutdown func()) {
	// Create database connection if given options for one
	if cfg.Postgres.ConnectionInfo.Database != "" {
		conn := cfg.Postgres.ConnectionInfo.ConnString()

		if dbc.conn == "" {
			glog.Infof("Connecting to Postgres for Stored Requests. DB=%s, host=%s, port=%d, user=%s",
				cfg.Postgres.ConnectionInfo.Database,
				cfg.Postgres.ConnectionInfo.Host,
				cfg.Postgres.ConnectionInfo.Port,
				cfg.Postgres.ConnectionInfo.Username)
			db := newPostgresDB(cfg.Postgres.ConnectionInfo)
			dbc.conn = conn
			dbc.db = db
		}

		// Error out if config is trying to use multiple database connections for different stored requests (not supported yet)
		if conn != dbc.conn {
			glog.Fatal("Multiple database connection settings found in Stored Requests config, only a single database connection is currently supported.")
		}
	}

	eventProducers := newEventProducers(cfg, client, dbc.db, router)
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
func NewStoredRequests(cfg *config.Configuration, metricsEngine pbsmetrics.MetricsEngine, client *http.Client, router *httprouter.Router) (db *sql.DB, shutdown func(), fetcher stored_requests.Fetcher, ampFetcher stored_requests.Fetcher, categoriesFetcher stored_requests.CategoryFetcher, videoFetcher stored_requests.Fetcher) {
	// Build individual slim options from combined config struct
	slimAuction, slimAmp := resolvedStoredRequestsConfig(cfg)

	// TODO: Switch this to be set in config defaults
	//if cfg.CategoryMapping.CacheEvents.Enabled && cfg.CategoryMapping.CacheEvents.Endpoint == "" {
	//	cfg.CategoryMapping.CacheEvents.Endpoint = "/storedrequest/categorymapping"
	//}

	var dbc dbConnection

	fetcher1, shutdown1 := CreateStoredRequests(&slimAuction, metricsEngine, client, router, &dbc)
	fetcher2, shutdown2 := CreateStoredRequests(&slimAmp, metricsEngine, client, router, &dbc)
	fetcher3, shutdown3 := CreateStoredRequests(&cfg.CategoryMapping, metricsEngine, client, router, &dbc)
	fetcher4, shutdown4 := CreateStoredRequests(&cfg.StoredVideo, metricsEngine, client, router, &dbc)

	db = dbc.db

	fetcher = fetcher1.(stored_requests.Fetcher)
	ampFetcher = fetcher2.(stored_requests.Fetcher)
	categoriesFetcher = fetcher3.(stored_requests.CategoryFetcher)
	videoFetcher = fetcher4.(stored_requests.Fetcher)

	shutdown = func() {
		shutdown1()
		shutdown2()
		shutdown3()
		shutdown4()
	}

	return
}

func resolvedStoredRequestsConfig(cfg *config.Configuration) (auc, amp config.StoredRequestsSlim) {
	sr := &cfg.StoredRequests

	// Auction endpoint uses non-Amp fields so can just copy the slin data
	auc.Files.Enabled = sr.Files
	auc.Files.Path = sr.Path
	auc.Postgres.ConnectionInfo = sr.Postgres.ConnectionInfo
	auc.Postgres.FetcherQueries.QueryTemplate = sr.Postgres.FetcherQueries.QueryTemplate
	auc.Postgres.CacheInitialization.Timeout = sr.Postgres.CacheInitialization.Timeout
	auc.Postgres.CacheInitialization.Query = sr.Postgres.CacheInitialization.Query
	auc.Postgres.PollUpdates.RefreshRate = sr.Postgres.PollUpdates.RefreshRate
	auc.Postgres.PollUpdates.Timeout = sr.Postgres.PollUpdates.Timeout
	auc.Postgres.PollUpdates.Query = sr.Postgres.PollUpdates.Query
	auc.HTTP.Endpoint = sr.HTTP.Endpoint
	auc.InMemoryCache = sr.InMemoryCache
	auc.CacheEvents.Enabled = sr.CacheEventsAPI
	auc.CacheEvents.Endpoint = "/storedrequests/openrtb2"
	auc.HTTPEvents.RefreshRate = sr.HTTPEvents.RefreshRate
	auc.HTTPEvents.Timeout = sr.HTTPEvents.Timeout
	auc.HTTPEvents.Endpoint = sr.HTTPEvents.Endpoint

	// Amp endpoint uses all the slim data but some fields get replacyed by Amp* version of similar fields
	amp.Files.Enabled = sr.Files
	amp.Files.Path = sr.Path
	amp.Postgres.ConnectionInfo = sr.Postgres.ConnectionInfo
	amp.Postgres.FetcherQueries.QueryTemplate = sr.Postgres.FetcherQueries.AmpQueryTemplate
	amp.Postgres.CacheInitialization.Timeout = sr.Postgres.CacheInitialization.Timeout
	amp.Postgres.CacheInitialization.Query = sr.Postgres.CacheInitialization.AmpQuery
	amp.Postgres.PollUpdates.RefreshRate = sr.Postgres.PollUpdates.RefreshRate
	amp.Postgres.PollUpdates.Timeout = sr.Postgres.PollUpdates.Timeout
	amp.Postgres.PollUpdates.Query = sr.Postgres.PollUpdates.AmpQuery
	amp.HTTP.Endpoint = sr.HTTP.AmpEndpoint
	amp.InMemoryCache = sr.InMemoryCache
	amp.CacheEvents.Enabled = sr.CacheEventsAPI
	amp.CacheEvents.Endpoint = "/storedrequests/amp"
	amp.HTTPEvents.RefreshRate = sr.HTTPEvents.RefreshRate
	amp.HTTPEvents.Timeout = sr.HTTPEvents.Timeout
	amp.HTTPEvents.Endpoint = sr.HTTPEvents.AmpEndpoint

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

func newFetcher(cfg *config.StoredRequestsSlim, client *http.Client, db *sql.DB) (fetcher stored_requests.AllFetcher) {
	idList := make(stored_requests.MultiFetcher, 0, 3)

	if cfg.Files.Enabled {
		fFetcher := newFilesystem(cfg.Files.Path)
		idList = append(idList, fFetcher)
	}
	if cfg.Postgres.FetcherQueries.QueryTemplate != "" {
		glog.Infof("Loading Stored Requests via Postgres.\nQuery: %s", cfg.Postgres.FetcherQueries.QueryTemplate)
		idList = append(idList, db_fetcher.NewFetcher(db, cfg.Postgres.FetcherQueries.MakeQuery))
	}
	if cfg.HTTP.Endpoint != "" {
		glog.Infof("Loading Stored Requests via HTTP. endpoint=%s", cfg.HTTP.Endpoint)
		idList = append(idList, http_fetcher.NewFetcher(client, cfg.HTTP.Endpoint))
	}

	fetcher = consolidate(idList)
	return
}

func newCache(cfg *config.StoredRequestsSlim) stored_requests.Cache {
	if cfg.InMemoryCache.Type == "none" {
		glog.Info("No Stored Request cache configured. The Fetcher backend will be used for all Stored Requests.")
		return &nil_cache.NilCache{}
	}

	return memory.NewCache(&cfg.InMemoryCache)
}

func newEventProducers(cfg *config.StoredRequestsSlim, client *http.Client, db *sql.DB, router *httprouter.Router) (eventProducers []events.EventProducer) {
	if cfg.CacheEvents.Enabled {
		eventProducers = append(eventProducers, newEventsAPI(router, cfg.CacheEvents.Endpoint))
	}
	if cfg.HTTPEvents.RefreshRate != 0 && cfg.HTTPEvents.Endpoint != "" {
		eventProducers = append(eventProducers, newHttpEvents(client, cfg.HTTPEvents.TimeoutDuration(), cfg.HTTPEvents.RefreshRateDuration(), cfg.HTTPEvents.Endpoint))
	}
	if cfg.Postgres.CacheInitialization.Query != "" {
		// Make sure we don't miss any updates in between the initial fetch and the "update" polling.
		updateStartTime := time.Now()
		timeout := time.Duration(cfg.Postgres.CacheInitialization.Timeout) * time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		eventProducers = append(eventProducers, postgresEvents.LoadAll(ctx, db, cfg.Postgres.CacheInitialization.Query))
		cancel()

		if cfg.Postgres.PollUpdates.Query != "" {
			eventProducers = append(eventProducers, newPostgresPolling(cfg.Postgres.PollUpdates, db, updateStartTime))
		}
	}
	return
}

func newPostgresPolling(cfg config.PostgresUpdatePollingSlim, db *sql.DB, startTime time.Time) events.EventProducer {
	timeout := time.Duration(cfg.Timeout) * time.Millisecond
	ctxProducer := func() (ctx context.Context, canceller func()) {
		return context.WithTimeout(context.Background(), timeout)
	}
	return postgresEvents.PollForUpdates(ctxProducer, db, cfg.Query, startTime, time.Duration(cfg.RefreshRate)*time.Second)
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

func newFilesystem(configPath string) stored_requests.AllFetcher {
	glog.Infof("Loading Stored Requests from filesystem at path %s", configPath)
	fetcher, err := file_fetcher.NewFileFetcher(configPath)
	if err != nil {
		glog.Fatalf("Failed to create a FileFetcher: %v", err)
	}
	return fetcher
}

func newPostgresDB(cfg config.PostgresConnection) *sql.DB {
	db, err := sql.Open("postgres", cfg.ConnString())
	if err != nil {
		glog.Fatalf("Failed to open postgres connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		glog.Fatalf("Failed to ping postgres: %v", err)
	}

	return db
}

// consolidate returns a single Fetcher from an array of fetchers of any size.
func consolidate(fetchers []stored_requests.AllFetcher) stored_requests.AllFetcher {
	if len(fetchers) == 0 {
		glog.Warning("No Stored Request support configured. request.imp[i].ext.prebid.storedrequest will be ignored. If you need this, check your app config")
		return empty_fetcher.EmptyFetcher{}
	} else if len(fetchers) == 1 {
		return fetchers[0]
	} else {
		return stored_requests.MultiFetcher(fetchers)
	}
}
