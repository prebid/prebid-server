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
)

// NewStoredRequests returns four things:
//
// 1. A Fetcher which can be used to get Stored Requests for /openrtb2/auction
// 2. A Fetcher which can be used to get Stored Requests for /openrtb2/amp
// 3. A DB connection, if one was created. This may be nil.
// 4. A function which should be called on shutdown for graceful cleanups.
//
// If any errors occur, the program will exit with an error message.
// It probably means you have a bad config or networking issue.
//
// As a side-effect, it will add some endpoints to the router if the config calls for it.
// In the future we should look for ways to simplify this so that it's not doing two things.
func NewStoredRequests(cfg *config.StoredRequests, client *http.Client, router *httprouter.Router) (fetcher stored_requests.Fetcher, ampFetcher stored_requests.Fetcher, db *sql.DB, shutdown func()) {
	eventProducers, ampEventProducers := newEventProducers(cfg, client, router)
	cache := newCache(cfg)
	ampCache := newCache(cfg)
	fetcher, ampFetcher, db = newFetchers(cfg, client)

	fetcher = stored_requests.WithCache(fetcher, cache)
	ampFetcher = stored_requests.WithCache(ampFetcher, ampCache)

	shutdown1 := addListeners(cache, eventProducers)
	shutdown2 := addListeners(ampCache, ampEventProducers)
	shutdown = func() {
		shutdown1()
		shutdown2()
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

func newFetchers(cfg *config.StoredRequests, client *http.Client) (fetcher stored_requests.Fetcher, ampFetcher stored_requests.Fetcher, db *sql.DB) {
	idList := make(stored_requests.MultiFetcher, 0, 3)
	ampIDList := make(stored_requests.MultiFetcher, 0, 3)

	if cfg.Files {
		fFetcher := newFilesystem()
		idList = append(idList, fFetcher)
		ampIDList = append(ampIDList, fFetcher)
	}
	if cfg.Postgres != nil {
		pFetcher, pAmpFetcher, pDb := newPostgres(cfg)
		idList = append(idList, pFetcher)
		ampIDList = append(ampIDList, pAmpFetcher)
		db = pDb
	}
	if cfg.HTTP != nil {
		glog.Infof("Loading Stored Requests via HTTP. endpoint=%s, amp_endpoint=%s", cfg.HTTP.Endpoint, cfg.HTTP.AmpEndpoint)
		idList = append(idList, http_fetcher.NewFetcher(client, cfg.HTTP.Endpoint))
		ampIDList = append(ampIDList, http_fetcher.NewFetcher(client, cfg.HTTP.AmpEndpoint))
	}

	fetcher = consolidate(idList)
	ampFetcher = consolidate(ampIDList)
	return
}

func newCache(cfg *config.StoredRequests) stored_requests.Cache {
	if cfg.InMemoryCache == nil {
		glog.Info("No Stored Request cache configured. The Fetcher backend will be used for all Stored Requests.")
		return &nil_cache.NilCache{}
	}

	glog.Infof("Using a Stored Request in-memory cache. Max size for StoredRequests: %d bytes. Max size for Stored Imps: %d bytes. TTL: %d seconds.", cfg.InMemoryCache.RequestCacheSize, cfg.InMemoryCache.ImpCacheSize, cfg.InMemoryCache.TTL)
	return memory.NewCache(cfg.InMemoryCache)
}

func newEventProducers(cfg *config.StoredRequests, client *http.Client, router *httprouter.Router) (eventProducers []events.EventProducer, ampEventProducers []events.EventProducer) {
	if cfg.CacheEventsAPI {
		eventProducers = append(eventProducers, newEventsAPI(router, "/storedrequests/openrtb2"))
		ampEventProducers = append(ampEventProducers, newEventsAPI(router, "/storedrequests/amp"))
	}
	if cfg.HTTPEvents != nil {
		eventProducers = append(eventProducers, newHttpEvents(client, cfg.HTTPEvents.TimeoutDuration(), cfg.HTTPEvents.RefreshRateDuration(), cfg.HTTPEvents.Endpoint))
		ampEventProducers = append(ampEventProducers, newHttpEvents(client, cfg.HTTPEvents.TimeoutDuration(), cfg.HTTPEvents.RefreshRateDuration(), cfg.HTTPEvents.AmpEndpoint))
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

func newFilesystem() stored_requests.Fetcher {
	glog.Infof("Loading Stored Requests from filesystem at path %s", requestConfigPath)
	fetcher, err := file_fetcher.NewFileFetcher(requestConfigPath)
	if err != nil {
		glog.Fatalf("Failed to create a FileFetcher: %v", err)
	}
	return fetcher
}

func newPostgres(cfg *config.StoredRequests) (fetcher stored_requests.Fetcher, ampFetcher stored_requests.Fetcher, db *sql.DB) {
	if conn, err := db_fetcher.NewPostgresDb(cfg.Postgres); err != nil {
		glog.Fatalf("Failed to connect to postgres: %v", err)
	} else {
		db = conn
	}
	glog.Infof("Loading Stored Requests from Postgres. DB=%s, host=%s, port=%d, user=%s, query=%s", cfg.Postgres.Database, cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.Username, cfg.Postgres.QueryTemplate)

	fetcher = db_fetcher.NewFetcher(db, cfg.Postgres.MakeQuery)
	ampFetcher = db_fetcher.NewFetcher(db, cfg.Postgres.MakeAmpQuery)
	return
}

// consolidate returns a single Fetcher from an array of fetchers of any size.
func consolidate(fetchers []stored_requests.Fetcher) stored_requests.Fetcher {
	if len(fetchers) == 0 {
		glog.Warning("No Stored Request support configured. request.imp[i].ext.prebid.storedrequest will be ignored. If you need this, check your app config")
		return empty_fetcher.EmptyFetcher{}
	} else if len(fetchers) == 1 {
		return fetchers[0]
	} else {
		return stored_requests.MultiFetcher(fetchers)
	}
}

const requestConfigPath = "./stored_requests/data/by_id"
