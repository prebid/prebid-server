package config

import (
	"context"
	"database/sql"
	"net/http"
	"time"

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
)

// NewStoredRequests returns five things:
//
// 1. A Fetcher which can be used to get Stored Requests for /openrtb2/auction
// 2. A Fetcher which can be used to get Stored Requests for /openrtb2/amp
// 3. A DB connection, if one was created. This may be nil.
// 4. A function which should be called on shutdown for graceful cleanups.
// 5. A Fetcher which can be used to get Category mapping for any /openrtb2 endpoint
//
// If any errors occur, the program will exit with an error message.
// It probably means you have a bad config or networking issue.
//
// As a side-effect, it will add some endpoints to the router if the config calls for it.
// In the future we should look for ways to simplify this so that it's not doing two things.
func NewStoredRequests(cfg *config.Configuration, client *http.Client, router *httprouter.Router) (fetcher stored_requests.Fetcher, ampFetcher stored_requests.Fetcher, db *sql.DB, shutdown func(), categoriesFetcher stored_requests.CategoryFetcher) {
	if cfg.StoredRequests.Postgres.ConnectionInfo.Database != "" {
		glog.Infof("Connecting to Postgres for Stored Requests. DB=%s, host=%s, port=%d, user=%s",
			cfg.StoredRequests.Postgres.ConnectionInfo.Database,
			cfg.StoredRequests.Postgres.ConnectionInfo.Host,
			cfg.StoredRequests.Postgres.ConnectionInfo.Port,
			cfg.StoredRequests.Postgres.ConnectionInfo.Username)
		db = newPostgresDB(cfg.StoredRequests.Postgres.ConnectionInfo)
	}
	eventProducers, ampEventProducers := newEventProducers(&cfg.StoredRequests, client, db, router)
	cache := newCache(&cfg.StoredRequests)
	ampCache := newCache(&cfg.StoredRequests)
	fetcher = newFetcher(&cfg.StoredRequests, client, db, false)
	ampFetcher = newFetcher(&cfg.StoredRequests, client, db, true)
	categoriesFetcher = newFetcher(&cfg.CategoryMapping, client, db, false)

	fetcher = stored_requests.WithCache(fetcher, cache)
	ampFetcher = stored_requests.WithCache(ampFetcher, ampCache)

	shutdown1 := addListeners(cache, eventProducers)
	shutdown2 := addListeners(ampCache, ampEventProducers)
	shutdown = func() {
		shutdown1()
		shutdown2()
		if db != nil {
			if err := db.Close(); err != nil {
				glog.Errorf("Error closing DB connection: %v", err)
			}
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

func newFetcher(cfg *config.StoredRequests, client *http.Client, db *sql.DB, isAmp bool) (fetcher stored_requests.AllFetcher) {
	idList := make(stored_requests.MultiFetcher, 0, 3)

	if cfg.Files {
		fFetcher := newFilesystem(cfg.Path)
		idList = append(idList, fFetcher)
	}
	if cfg.Postgres.FetcherQueries.QueryTemplate != "" {
		glog.Infof("Loading Stored Requests via Postgres.\nQuery: %s\nAMP Query: %s", cfg.Postgres.FetcherQueries.QueryTemplate, cfg.Postgres.FetcherQueries.AmpQueryTemplate)
		idList = append(idList, db_fetcher.NewFetcher(db, cfg.Postgres.FetcherQueries.MakeQuery))
	}
	if cfg.HTTP.Endpoint != "" && !isAmp {
		glog.Infof("Loading Stored Requests via HTTP. endpoint=%s", cfg.HTTP.Endpoint)
		idList = append(idList, http_fetcher.NewFetcher(client, cfg.HTTP.Endpoint))
	}
	if cfg.HTTP.AmpEndpoint != "" && isAmp {
		glog.Infof("Loading Stored Requests via HTTP. amp_endpoint=%s", cfg.HTTP.AmpEndpoint)
		idList = append(idList, http_fetcher.NewFetcher(client, cfg.HTTP.AmpEndpoint))
	}

	fetcher = consolidate(idList)
	return
}

func newCache(cfg *config.StoredRequests) stored_requests.Cache {
	if cfg.InMemoryCache.Type == "none" {
		glog.Info("No Stored Request cache configured. The Fetcher backend will be used for all Stored Requests.")
		return &nil_cache.NilCache{}
	}

	return memory.NewCache(&cfg.InMemoryCache)
}

func newEventProducers(cfg *config.StoredRequests, client *http.Client, db *sql.DB, router *httprouter.Router) (eventProducers []events.EventProducer, ampEventProducers []events.EventProducer) {
	if cfg.CacheEventsAPI {
		eventProducers = append(eventProducers, newEventsAPI(router, "/storedrequests/openrtb2"))
		ampEventProducers = append(ampEventProducers, newEventsAPI(router, "/storedrequests/amp"))
	}
	if cfg.HTTPEvents.RefreshRate != 0 {
		if cfg.HTTPEvents.Endpoint != "" {
			eventProducers = append(eventProducers, newHttpEvents(client, cfg.HTTPEvents.TimeoutDuration(), cfg.HTTPEvents.RefreshRateDuration(), cfg.HTTPEvents.Endpoint))
		}
		if cfg.HTTPEvents.AmpEndpoint != "" {
			ampEventProducers = append(ampEventProducers, newHttpEvents(client, cfg.HTTPEvents.TimeoutDuration(), cfg.HTTPEvents.RefreshRateDuration(), cfg.HTTPEvents.AmpEndpoint))
		}
	}
	if cfg.Postgres.CacheInitialization.Query != "" {
		// Make sure we don't miss any updates in between the initial fetch and the "update" polling.
		updateStartTime := time.Now()
		timeout := time.Duration(cfg.Postgres.CacheInitialization.Timeout) * time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		eventProducers = append(eventProducers, postgresEvents.LoadAll(ctx, db, cfg.Postgres.CacheInitialization.Query))
		cancel()

		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		ampEventProducers = append(ampEventProducers, postgresEvents.LoadAll(ctx, db, cfg.Postgres.CacheInitialization.AmpQuery))
		cancel()

		if cfg.Postgres.PollUpdates.Query != "" {
			eventProducers = append(eventProducers, newPostgresPolling(cfg.Postgres.PollUpdates, db, updateStartTime, false))
			ampEventProducers = append(ampEventProducers, newPostgresPolling(cfg.Postgres.PollUpdates, db, updateStartTime, true))
		}
	}
	return
}

func newPostgresPolling(cfg config.PostgresUpdatePolling, db *sql.DB, startTime time.Time, forAmp bool) events.EventProducer {
	timeout := time.Duration(cfg.Timeout) * time.Millisecond
	ctxProducer := func() (ctx context.Context, canceller func()) {
		return context.WithTimeout(context.Background(), timeout)
	}

	if forAmp {
		return postgresEvents.PollForUpdates(ctxProducer, db, cfg.AmpQuery, startTime, time.Duration(cfg.RefreshRate)*time.Second)
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
