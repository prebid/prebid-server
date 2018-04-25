package config

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/stored_requests/backends/http_fetcher"
	"github.com/prebid/prebid-server/stored_requests/events"

	"github.com/prebid/prebid-server/stored_requests/backends/empty_fetcher"

	"github.com/prebid/prebid-server/config"
)

func TestNewEmptyFetcher(t *testing.T) {
	fetcher, ampFetcher := newFetchers(&config.StoredRequests{}, nil, nil)
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
	fetcher, ampFetcher := newFetchers(&config.StoredRequests{
		HTTP: &config.HTTPFetcherConfig{
			Endpoint:    "stored-requests.prebid.com",
			AmpEndpoint: "stored-requests.prebid.com?type=amp",
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

func TestNewEmptyCache(t *testing.T) {
	cache := newCache(&config.StoredRequests{})
	cache.Save(context.Background(), map[string]json.RawMessage{"foo": json.RawMessage("true")}, nil)
	reqs, _ := cache.Get(context.Background(), []string{"foo"}, nil)
	if len(reqs) != 0 {
		t.Errorf("The newCache method should return an empty cache if the config asks for it.")
	}
}

func TestNewInMemoryCache(t *testing.T) {
	cache := newCache(&config.StoredRequests{
		InMemoryCache: &config.InMemoryCache{
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
		Postgres: &config.PostgresConfig{
			CacheInitialization: &config.PostgresCacheInitializer{
				Timeout:  50,
				Query:    "SELECT id, requestData, type FROM stored_data",
				AmpQuery: "SELECT id, requestData, type FROM stored_amp_data",
			},
			PollUpdates: &config.PostgresUpdatePolling{
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
