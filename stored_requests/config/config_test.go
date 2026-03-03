package config

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/julienschmidt/httprouter"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/stored_requests"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/db_provider"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/empty_fetcher"
	"github.com/prebid/prebid-server/v3/stored_requests/backends/http_fetcher"
	"github.com/prebid/prebid-server/v3/stored_requests/events"
	httpEvents "github.com/prebid/prebid-server/v3/stored_requests/events/http"
	"github.com/stretchr/testify/mock"
)

func typedConfig(dataType config.DataType, sr *config.StoredRequests) *config.StoredRequests {
	sr.SetDataType(dataType)
	return sr
}

func isEmptyCacheType(cache stored_requests.CacheJSON) bool {
	cache.Save(context.Background(), map[string]json.RawMessage{"foo": json.RawMessage("true")})
	objs := cache.Get(context.Background(), []string{"foo"})
	return len(objs) == 0
}

func isMemoryCacheType(cache stored_requests.CacheJSON) bool {
	cache.Save(context.Background(), map[string]json.RawMessage{"foo": json.RawMessage("true")})
	objs := cache.Get(context.Background(), []string{"foo"})
	return len(objs) == 1
}

func TestNewEmptyFetcher(t *testing.T) {

	type testCase struct {
		config       *config.StoredRequests
		emptyFetcher bool
		description  string
	}
	testCases := []testCase{
		{
			config:       &config.StoredRequests{},
			emptyFetcher: true,
			description:  "If the config is empty, an EmptyFetcher should be returned",
		},
		{
			config: &config.StoredRequests{
				Database: config.DatabaseConfig{
					ConnectionInfo: config.DatabaseConnection{
						Driver: "postgres",
					},
					CacheInitialization: config.DatabaseCacheInitializer{
						Query: "test query",
					},
					PollUpdates: config.DatabaseUpdatePolling{
						Query: "test poll query",
					},
					FetcherQueries: config.DatabaseFetcherQueries{
						QueryTemplate: "",
					},
				},
			},
			emptyFetcher: true,
			description:  "If Database fetcher query is not defined, but Database Cache init query and Database update polling query are defined EmptyFetcher should be returned",
		},
		{
			config: &config.StoredRequests{
				Database: config.DatabaseConfig{
					ConnectionInfo: config.DatabaseConnection{
						Driver: "postgres",
					},
					CacheInitialization: config.DatabaseCacheInitializer{
						Query: "",
					},
					PollUpdates: config.DatabaseUpdatePolling{
						Query: "",
					},
					FetcherQueries: config.DatabaseFetcherQueries{
						QueryTemplate: "test fetcher query",
					},
				},
			},
			emptyFetcher: false,
			description:  "If Database fetcher query is defined, but Database Cache init query and Database update polling query are not defined not EmptyFetcher (DBFetcher) should be returned",
		},
		{
			config: &config.StoredRequests{
				Database: config.DatabaseConfig{
					ConnectionInfo: config.DatabaseConnection{
						Driver: "postgres",
					},
					CacheInitialization: config.DatabaseCacheInitializer{
						Query: "test cache query",
					},
					PollUpdates: config.DatabaseUpdatePolling{
						Query: "test poll query",
					},
					FetcherQueries: config.DatabaseFetcherQueries{
						QueryTemplate: "test fetcher query",
					},
				},
			},
			emptyFetcher: false,
			description:  "If Database fetcher query is defined and Database Cache init query and Database update polling query are defined not EmptyFetcher (DBFetcher) should be returned",
		},
	}

	for _, test := range testCases {
		fetcher := newFetcher(test.config, nil, db_provider.DbProviderMock{})
		assert.NotNil(t, fetcher, "The fetcher should be non-nil.")
		if test.emptyFetcher {
			assert.Equal(t, empty_fetcher.EmptyFetcher{}, fetcher, "Empty fetcher should be returned")
		} else {
			assert.NotEqual(t, empty_fetcher.EmptyFetcher{}, fetcher)
		}
	}
}

func TestNewHTTPFetcher(t *testing.T) {
	fetcher := newFetcher(&config.StoredRequests{
		HTTP: config.HTTPFetcherConfig{
			Endpoint: "stored-requests.prebid.com",
		},
	}, nil, nil)
	if httpFetcher, ok := fetcher.(*http_fetcher.HttpFetcher); ok {
		if httpFetcher.EndpointURL.String() != "stored-requests.prebid.com" {
			t.Errorf("The HTTP fetcher is using the wrong endpoint. Expected %s, got %s", "stored-requests.prebid.com", httpFetcher.EndpointURL)
		}
	} else {
		t.Errorf("An HTTP Fetching config should return an HTTPFetcher. Got %v", fetcher)
	}
}

func TestNewHTTPEvents(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}
	server1 := httptest.NewServer(http.HandlerFunc(handler))

	cfg := &config.StoredRequests{
		HTTPEvents: config.HTTPEventsConfig{
			Endpoint:    server1.URL,
			RefreshRate: 100,
			Timeout:     1000,
		},
	}

	metricsMock := &metrics.MetricsEngineMock{}

	evProducers := newEventProducers(cfg, server1.Client(), nil, metricsMock, nil)
	assertSliceLength(t, evProducers, 1)
	assertHttpWithURL(t, evProducers[0], server1.URL)
}

func TestNewEmptyCache(t *testing.T) {
	cache := newCache(&config.StoredRequests{InMemoryCache: config.InMemoryCache{Type: "none"}})
	assert.True(t, isEmptyCacheType(cache.Requests), "The newCache method should return an empty Request cache")
	assert.True(t, isEmptyCacheType(cache.Imps), "The newCache method should return an empty Imp cache")
	assert.True(t, isEmptyCacheType(cache.Responses), "The newCache method should return an empty Responses cache")
	assert.True(t, isEmptyCacheType(cache.Accounts), "The newCache method should return an empty Account cache")
}

func TestNewInMemoryCache(t *testing.T) {
	cache := newCache(&config.StoredRequests{
		InMemoryCache: config.InMemoryCache{
			TTL:              60,
			RequestCacheSize: 100,
			ImpCacheSize:     100,
			RespCacheSize:    100,
		},
	})
	assert.True(t, isMemoryCacheType(cache.Requests), "The newCache method should return an in-memory Request cache for StoredRequests config")
	assert.True(t, isMemoryCacheType(cache.Imps), "The newCache method should return an in-memory Imp cache for StoredRequests config")
	assert.True(t, isMemoryCacheType(cache.Responses), "The newCache method should return an in-memory Responses cache for StoredResponses config")
	assert.True(t, isEmptyCacheType(cache.Accounts), "The newCache method should return an empty Account cache for StoredRequests config")
}

func TestNewInMemoryAccountCache(t *testing.T) {
	cache := newCache(typedConfig(config.AccountDataType, &config.StoredRequests{
		InMemoryCache: config.InMemoryCache{
			TTL:  60,
			Size: 100,
		},
	}))
	assert.True(t, isMemoryCacheType(cache.Accounts), "The newCache method should return an in-memory Account cache for Accounts config")
	assert.True(t, isEmptyCacheType(cache.Requests), "The newCache method should return an empty Request cache for Accounts config")
	assert.True(t, isEmptyCacheType(cache.Imps), "The newCache method should return an empty Imp cache for Accounts config")
	assert.True(t, isEmptyCacheType(cache.Responses), "The newCache method should return an empty Responses cache for Accounts config")
}

func TestNewDatabaseEventProducers(t *testing.T) {
	metricsMock := &metrics.MetricsEngineMock{}
	metricsMock.Mock.On("RecordStoredDataFetchTime", mock.Anything, mock.Anything).Return()
	metricsMock.Mock.On("RecordStoredDataError", mock.Anything).Return()

	cfg := &config.StoredRequests{
		Database: config.DatabaseConfig{
			CacheInitialization: config.DatabaseCacheInitializer{
				Timeout: 50,
				Query:   "SELECT id, requestData, type FROM stored_data",
			},
			PollUpdates: config.DatabaseUpdatePolling{
				RefreshRate: 20,
				Timeout:     50,
				Query:       "SELECT id, requestData, type FROM stored_data WHERE last_updated > $1",
			},
		},
	}
	client := &http.Client{}
	provider, mock, err := db_provider.NewDbProviderMock()
	if err != nil {
		t.Fatalf("Failed to create mock: %v", err)
	}
	mock.ExpectQuery("^" + regexp.QuoteMeta(cfg.Database.CacheInitialization.Query) + "$").WillReturnError(errors.New("Query failed"))

	evProducers := newEventProducers(cfg, client, provider, metricsMock, nil)
	assertProducerLength(t, evProducers, 1)

	assertExpectationsMet(t, mock)
	metricsMock.AssertExpectations(t)
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
