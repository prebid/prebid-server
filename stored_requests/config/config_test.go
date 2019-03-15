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
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/stored_requests/backends/http_fetcher"
	"github.com/prebid/prebid-server/stored_requests/events"
	httpEvents "github.com/prebid/prebid-server/stored_requests/events/http"
)

func TestNewEmptyFetcher(t *testing.T) {
	fetcher := newFetcher(&config.StoredRequests{}, nil, nil, false)
	ampFetcher := newFetcher(&config.StoredRequests{}, nil, nil, true)
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
	fetcher := newFetcher(&config.StoredRequests{
		HTTP: config.HTTPFetcherConfig{
			Endpoint:    "stored-requests.prebid.com",
			AmpEndpoint: "stored-requests.prebid.com?type=amp",
		},
	}, nil, nil, false)
	ampFetcher := newFetcher(&config.StoredRequests{
		HTTP: config.HTTPFetcherConfig{
			Endpoint:    "stored-requests.prebid.com",
			AmpEndpoint: "stored-requests.prebid.com?type=amp",
		},
	}, nil, nil, true)
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
	fetcher := newFetcher(&config.StoredRequests{
		HTTP: config.HTTPFetcherConfig{
			Endpoint:    "stored-requests.prebid.com",
			AmpEndpoint: "",
		},
	}, nil, nil, false)
	ampFetcher := newFetcher(&config.StoredRequests{
		HTTP: config.HTTPFetcherConfig{
			Endpoint:    "stored-requests.prebid.com",
			AmpEndpoint: "",
		},
	}, nil, nil, true)
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
func TestNewHTTPEvents(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	server1 := httptest.NewServer(http.HandlerFunc(handler))
	server2 := httptest.NewServer(http.HandlerFunc(handler))

	cfg := &config.StoredRequests{
		HTTPEvents: config.HTTPEventsConfig{
			Endpoint:    server1.URL,
			AmpEndpoint: server2.URL,
			RefreshRate: 100,
			Timeout:     1000,
		},
	}
	evProducers, ampProducers := newEventProducers(cfg, server1.Client(), nil, nil)
	assertSliceLength(t, evProducers, 1)
	assertSliceLength(t, ampProducers, 1)
	assertHttpWithURL(t, evProducers[0], server1.URL)
	assertHttpWithURL(t, ampProducers[0], server2.URL)
}

func TestNewEmptyCache(t *testing.T) {
	cache := newCache(&config.StoredRequests{InMemoryCache: config.InMemoryCache{Type: "none"}})
	cache.Save(context.Background(), map[string]json.RawMessage{"foo": json.RawMessage("true")}, nil)
	reqs, _ := cache.Get(context.Background(), []string{"foo"}, nil)
	if len(reqs) != 0 {
		t.Errorf("The newCache method should return an empty cache if the config asks for it.")
	}
}

func TestNewInMemoryCache(t *testing.T) {
	cache := newCache(&config.StoredRequests{
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
	cfg := &config.StoredRequests{
		Postgres: config.PostgresConfig{
			CacheInitialization: config.PostgresCacheInitializer{
				Timeout:  50,
				Query:    "SELECT id, requestData, type FROM stored_data",
				AmpQuery: "SELECT id, requestData, type FROM stored_amp_data",
			},
			PollUpdates: config.PostgresUpdatePolling{
				RefreshRate: 20,
				Timeout:     50,
				Query:       "SELECT id, requestData, type FROM stored_data WHERE last_updated > $1",
				AmpQuery:    "SELECT id, requestData, type FROM stored_amp_data WHERE last_updated > $1",
			},
		},
	}
	client := &http.Client{}
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	mock.ExpectQuery("^" + regexp.QuoteMeta(cfg.Postgres.CacheInitialization.Query) + "$").WillReturnError(errors.New("Query failed"))
	mock.ExpectQuery("^" + regexp.QuoteMeta(cfg.Postgres.CacheInitialization.AmpQuery) + "$").WillReturnError(errors.New("Query failed"))

	evProducers, ampEvProducers := newEventProducers(cfg, client, db, nil)
	assertExpectationsMet(t, mock)
	assertProducerLength(t, evProducers, 2)
	assertProducerLength(t, ampEvProducers, 2)
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
