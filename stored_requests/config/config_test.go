package config

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/backends/http_fetcher"
	"github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events"
	httpEvents "github.com/PubMatic-OpenWrap/prebid-server/stored_requests/events/http"
	"github.com/julienschmidt/httprouter"
)

func TestNewEmptyFetcher(t *testing.T) {
	fetcher := newFetcher(&config.StoredRequestsSlim{}, nil, nil)
	ampFetcher := newFetcher(&config.StoredRequestsSlim{}, nil, nil)
	if fetcher == nil || ampFetcher == nil {
		t.Errorf("The fetchers should be non-nil, even with an empty config.")
	}
	if _, ok := fetcher.(empty_fetcher.EmptyFetcher); !ok {
		t.Errorf("If the config is empty, and EmptyFetcher should be returned")
	}
	if _, ok := ampFetcher.(empty_fetcher.EmptyFetcher); !ok {
		t.Errorf("If the config is empty, and EmptyFetcher should be returned for AMP")
	}
}

func TestNewHTTPFetcher(t *testing.T) {
	fetcher := newFetcher(&config.StoredRequestsSlim{
		HTTP: config.HTTPFetcherConfigSlim{
			Endpoint: "stored-requests.prebid.com",
		},
	}, nil, nil)
	ampFetcher := newFetcher(&config.StoredRequestsSlim{
		HTTP: config.HTTPFetcherConfigSlim{
			Endpoint: "stored-requests.prebid.com?type=amp",
		},
	}, nil, nil)
	if httpFetcher, ok := fetcher.(*http_fetcher.HttpFetcher); ok {
		if httpFetcher.Endpoint != "stored-requests.prebid.com?" {
			t.Errorf("The HTTP fetcher is using the wrong endpoint. Expected %s, got %s", "stored-requests.prebid.com?", httpFetcher.Endpoint)
		}
	} else {
		t.Errorf("An HTTP Fetching config should return an HTTPFetcher. Got %v", ampFetcher)
	}
	if httpFetcher, ok := ampFetcher.(*http_fetcher.HttpFetcher); ok {
		if httpFetcher.Endpoint != "stored-requests.prebid.com?type=amp&" {
			t.Errorf("The AMP HTTP fetcher is using the wrong endpoint. Expected %s, got %s", "stored-requests.prebid.com?type=amp&", httpFetcher.Endpoint)
		}
	} else {
		t.Errorf("An HTTP Fetching config should return an HTTPFetcher. Got %v", ampFetcher)
	}
}

func TestNewHTTPFetcherNoAmp(t *testing.T) {
	fetcher := newFetcher(&config.StoredRequestsSlim{
		HTTP: config.HTTPFetcherConfigSlim{
			Endpoint: "stored-requests.prebid.com",
		},
	}, nil, nil)
	ampFetcher := newFetcher(&config.StoredRequestsSlim{
		HTTP: config.HTTPFetcherConfigSlim{
			Endpoint: "",
		},
	}, nil, nil)
	if httpFetcher, ok := fetcher.(*http_fetcher.HttpFetcher); ok {
		if httpFetcher.Endpoint != "stored-requests.prebid.com?" {
			t.Errorf("The HTTP fetcher is using the wrong endpoint. Expected %s, got %s", "stored-requests.prebid.com?", httpFetcher.Endpoint)
		}
	} else {
		t.Errorf("An HTTP Fetching config should return an HTTPFetcher. Got %v", ampFetcher)
	}
	if httpAmpFetcher, ok := ampFetcher.(*http_fetcher.HttpFetcher); ok && httpAmpFetcher == nil {
		t.Errorf("An HTTP Fetching config should not return an Amp HTTP fetcher in this case. Got %v (%v)", ampFetcher, httpAmpFetcher)
	}
}

func TestResolveConfig(t *testing.T) {
	cfg := &config.Configuration{
		StoredRequests: config.StoredRequests{
			Files: true,
			Path:  "/test-path",
			Postgres: config.PostgresConfig{
				ConnectionInfo: config.PostgresConnection{
					Database: "db",
					Host:     "pghost",
					Port:     5,
					Username: "user",
					Password: "pass",
				},
				FetcherQueries: config.PostgresFetcherQueries{
					AmpQueryTemplate: "amp-fetcher-query",
				},
				CacheInitialization: config.PostgresCacheInitializer{
					AmpQuery: "amp-cache-init-query",
				},
				PollUpdates: config.PostgresUpdatePolling{
					AmpQuery: "amp-poll-query",
				},
			},
			HTTP: config.HTTPFetcherConfig{
				AmpEndpoint: "amp-http-fetcher-endpoint",
			},
			InMemoryCache: config.InMemoryCache{
				Type:             "none",
				TTL:              50,
				RequestCacheSize: 1,
				ImpCacheSize:     2,
			},
			CacheEventsAPI: true,
			HTTPEvents: config.HTTPEventsConfig{
				AmpEndpoint: "amp-http-events-endpoint",
			},
		},
	}

	cfg.StoredRequests.Postgres.FetcherQueries.QueryTemplate = "auc-fetcher-query"
	cfg.StoredRequests.Postgres.CacheInitialization.Query = "auc-cache-init-query"
	cfg.StoredRequests.Postgres.PollUpdates.Query = "auc-poll-query"
	cfg.StoredRequests.HTTP.Endpoint = "auc-http-fetcher-endpoint"
	cfg.StoredRequests.HTTPEvents.Endpoint = "auc-http-events-endpoint"

	auc, amp := resolvedStoredRequestsConfig(cfg)

	// Auction slim should have the non-amp values in it
	assertStringsEqual(t, auc.Postgres.FetcherQueries.QueryTemplate, cfg.StoredRequests.Postgres.FetcherQueries.QueryTemplate)
	assertStringsEqual(t, auc.Postgres.CacheInitialization.Query, cfg.StoredRequests.Postgres.CacheInitialization.Query)
	assertStringsEqual(t, auc.Postgres.PollUpdates.Query, cfg.StoredRequests.Postgres.PollUpdates.Query)
	assertStringsEqual(t, auc.HTTP.Endpoint, cfg.StoredRequests.HTTP.Endpoint)
	assertStringsEqual(t, auc.HTTPEvents.Endpoint, cfg.StoredRequests.HTTPEvents.Endpoint)
	assertStringsEqual(t, auc.CacheEvents.Endpoint, "/storedrequests/openrtb2")

	// Amp slim should have the amp values in it
	assertStringsEqual(t, amp.Postgres.FetcherQueries.QueryTemplate, cfg.StoredRequests.Postgres.FetcherQueries.AmpQueryTemplate)
	assertStringsEqual(t, amp.Postgres.CacheInitialization.Query, cfg.StoredRequests.Postgres.CacheInitialization.AmpQuery)
	assertStringsEqual(t, amp.Postgres.PollUpdates.Query, cfg.StoredRequests.Postgres.PollUpdates.AmpQuery)
	assertStringsEqual(t, amp.HTTP.Endpoint, cfg.StoredRequests.HTTP.AmpEndpoint)
	assertStringsEqual(t, amp.HTTPEvents.Endpoint, cfg.StoredRequests.HTTPEvents.AmpEndpoint)
	assertStringsEqual(t, amp.CacheEvents.Endpoint, "/storedrequests/amp")
}

func TestNewHTTPEvents(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	server1 := httptest.NewServer(http.HandlerFunc(handler))

	cfg := &config.StoredRequestsSlim{
		HTTPEvents: config.HTTPEventsConfigSlim{
			Endpoint:    server1.URL,
			RefreshRate: 100,
			Timeout:     1000,
		},
	}
	evProducers := newEventProducers(cfg, server1.Client(), nil, nil)
	assertSliceLength(t, evProducers, 1)
	assertHttpWithURL(t, evProducers[0], server1.URL)
}

func TestNewEmptyCache(t *testing.T) {
	cache := newCache(&config.StoredRequestsSlim{InMemoryCache: config.InMemoryCache{Type: "none"}})
	cache.Save(context.Background(), map[string]json.RawMessage{"foo": json.RawMessage("true")}, nil)
	reqs, _ := cache.Get(context.Background(), []string{"foo"}, nil)
	if len(reqs) != 0 {
		t.Errorf("The newCache method should return an empty cache if the config asks for it.")
	}
}

func TestNewInMemoryCache(t *testing.T) {
	cache := newCache(&config.StoredRequestsSlim{
		InMemoryCache: config.InMemoryCache{
			TTL:              60,
			RequestCacheSize: 100,
			ImpCacheSize:     100,
		},
	})
	cache.Save(context.Background(), map[string]json.RawMessage{"foo": json.RawMessage("true")}, nil)
	reqs, _ := cache.Get(context.Background(), []string{"foo"}, nil)
	if len(reqs) != 1 {
		t.Errorf("The newCache method should return an in-memory cache if the config asks for it.")
	}
}

func TestNewPostgresEventProducers(t *testing.T) {
	cfg := &config.StoredRequestsSlim{
		Postgres: config.PostgresConfigSlim{
			CacheInitialization: config.PostgresCacheInitializerSlim{
				Timeout: 50,
				Query:   "SELECT id, requestData, type FROM stored_data",
			},
			PollUpdates: config.PostgresUpdatePollingSlim{
				RefreshRate: 20,
				Timeout:     50,
				Query:       "SELECT id, requestData, type FROM stored_data WHERE last_updated > $1",
			},
		},
	}
	ampCfg := &config.StoredRequestsSlim{
		Postgres: config.PostgresConfigSlim{
			CacheInitialization: config.PostgresCacheInitializerSlim{
				Timeout: 50,
				Query:   "SELECT id, requestData, type FROM stored_amp_data",
			},
			PollUpdates: config.PostgresUpdatePollingSlim{
				RefreshRate: 20,
				Timeout:     50,
				Query:       "SELECT id, requestData, type FROM stored_amp_data WHERE last_updated > $1",
			},
		},
	}
	client := &http.Client{}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	mock.ExpectQuery("^" + regexp.QuoteMeta(cfg.Postgres.CacheInitialization.Query) + "$").WillReturnError(errors.New("Query failed"))
	mock.ExpectQuery("^" + regexp.QuoteMeta(ampCfg.Postgres.CacheInitialization.Query) + "$").WillReturnError(errors.New("Query failed"))

	evProducers := newEventProducers(cfg, client, db, nil)
	assertProducerLength(t, evProducers, 2)

	ampEvProducers := newEventProducers(ampCfg, client, db, nil)
	assertProducerLength(t, ampEvProducers, 2)

	assertExpectationsMet(t, mock)
}

func TestNewEventsAPI(t *testing.T) {
	router := httprouter.New()
	newEventsAPI(router, "/test-endpoint")
	if handle, _, _ := router.Lookup("POST", "/test-endpoint"); handle == nil {
		t.Error("The newEventsAPI method didn't add a POST /test-endpoint route")
	}
	if handle, _, _ := router.Lookup("DELETE", "/test-endpoint"); handle == nil {
		t.Error("The newEventsAPI method didn't add a DELETE /test-endpoint route")
	}
}

func assertProducerLength(t *testing.T, producers []events.EventProducer, expectedLength int) {
	t.Helper()
	if len(producers) != expectedLength {
		t.Errorf("Expected %d producers, but got %d", expectedLength, len(producers))
	}
}

func assertExpectationsMet(t *testing.T, mock sqlmock.Sqlmock) {
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("sqlmock expectations were not met: %v", err)
	}
}

func assertHttpWithURL(t *testing.T, ev events.EventProducer, url string) {
	if casted, ok := ev.(*httpEvents.HTTPEvents); ok {
		assertStringsEqual(t, casted.Endpoint, url)
	} else {
		t.Errorf("The EventProducer was not a *HTTPEvents")
	}
}

func assertSliceLength(t *testing.T, producers []events.EventProducer, expected int) {
	t.Helper()

	if len(producers) != expected {
		t.Fatalf("Expected %d EventProducers. Got: %v", expected, producers)
	}
}

func assertStringsEqual(t *testing.T, actual string, expected string) {
	t.Helper()

	if actual != expected {
		t.Fatalf("String %s did not match expected %s", actual, expected)
	}
}
