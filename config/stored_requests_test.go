package config

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInMemoryCacheValidationStoredRequests(t *testing.T) {
	assertNoErrs(t, (&InMemoryCache{
		Type: "unbounded",
	}).validate(RequestDataType, nil))
	assertNoErrs(t, (&InMemoryCache{
		Type: "none",
	}).validate(RequestDataType, nil))
	assertNoErrs(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 1000,
		ImpCacheSize:     1000,
		RespCacheSize:    1000,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "unrecognized",
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:         "unbounded",
		ImpCacheSize: 1000,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "unbounded",
		RequestCacheSize: 1000,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:          "unbounded",
		RespCacheSize: 1000,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "unbounded",
		TTL:  500,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 0,
		ImpCacheSize:     1000,
		RespCacheSize:    1000,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 1000,
		ImpCacheSize:     0,
		RespCacheSize:    1000,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 1000,
		ImpCacheSize:     1000,
		RespCacheSize:    0,
	}).validate(RequestDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "lru",
		Size: 1000,
	}).validate(RequestDataType, nil))
}

func TestInMemoryCacheValidationSingleCache(t *testing.T) {
	assertNoErrs(t, (&InMemoryCache{
		Type: "unbounded",
	}).validate(AccountDataType, nil))
	assertNoErrs(t, (&InMemoryCache{
		Type: "none",
	}).validate(AccountDataType, nil))
	assertNoErrs(t, (&InMemoryCache{
		Type: "lru",
		Size: 1000,
	}).validate(AccountDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "unrecognized",
	}).validate(AccountDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "unbounded",
		Size: 1000,
	}).validate(AccountDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "unbounded",
		TTL:  500,
	}).validate(AccountDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "lru",
		Size: 0,
	}).validate(AccountDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 1000,
	}).validate(AccountDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:         "lru",
		ImpCacheSize: 1000,
	}).validate(AccountDataType, nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:          "lru",
		RespCacheSize: 1000,
	}).validate(AccountDataType, nil))
}

func TestDatabaseConfigValidation(t *testing.T) {
	tests := []struct {
		description            string
		connectionStr          string
		cacheInitQuery         string
		cacheInitTimeout       int
		cacheUpdateQuery       string
		cacheUpdateRefreshRate int
		cacheUpdateTimeout     int
		existingErrors         []error
		wantErrorCount         int
	}{
		{
			description:   "No connection string",
			connectionStr: "",
		},
		{
			description:   "Connection string but no queries",
			connectionStr: "some-connection-string",
		},
		{
			description:      "Valid cache init query with non-zero timeout",
			connectionStr:    "some-connection-string",
			cacheInitQuery:   "SELECT * FROM table;",
			cacheInitTimeout: 1,
		},
		{
			description:      "Valid cache init query with zero timeout",
			connectionStr:    "some-connection-string",
			cacheInitQuery:   "SELECT * FROM table;",
			cacheInitTimeout: 0,
			wantErrorCount:   1,
		},
		{
			description:      "Invalid cache init query contains wildcard",
			connectionStr:    "some-connection-string",
			cacheInitQuery:   "SELECT * FROM table WHERE $LAST_UPDATED",
			cacheInitTimeout: 1,
			wantErrorCount:   1,
		},
		{
			description:            "Valid cache update query with non-zero timeout and refresh rate",
			connectionStr:          "some-connection-string",
			cacheUpdateQuery:       "SELECT * FROM table WHERE $LAST_UPDATED",
			cacheUpdateRefreshRate: 1,
			cacheUpdateTimeout:     1,
		},
		{
			description:            "Valid cache update query with zero timeout and non-zero refresh rate",
			connectionStr:          "some-connection-string",
			cacheUpdateQuery:       "SELECT * FROM table WHERE $LAST_UPDATED",
			cacheUpdateRefreshRate: 1,
			cacheUpdateTimeout:     0,
			wantErrorCount:         1,
		},
		{
			description:            "Valid cache update query with non-zero timeout and zero refresh rate",
			connectionStr:          "some-connection-string",
			cacheUpdateQuery:       "SELECT * FROM table WHERE $LAST_UPDATED",
			cacheUpdateRefreshRate: 0,
			cacheUpdateTimeout:     1,
			wantErrorCount:         1,
		},
		{
			description:            "Invalid cache update query missing wildcard",
			connectionStr:          "some-connection-string",
			cacheUpdateQuery:       "SELECT * FROM table",
			cacheUpdateRefreshRate: 1,
			cacheUpdateTimeout:     1,
			wantErrorCount:         1,
		},
		{
			description:      "Multiple errors: valid queries missing timeouts and refresh rates plus existing error",
			connectionStr:    "some-connection-string",
			cacheInitQuery:   "SELECT * FROM table;",
			cacheUpdateQuery: "SELECT * FROM table WHERE $LAST_UPDATED",
			existingErrors:   []error{errors.New("existing error before calling validate")},
			wantErrorCount:   4,
		},
	}

	for _, tt := range tests {
		dbConfig := &DatabaseConfig{
			ConnectionInfo: DatabaseConnection{
				Database: tt.connectionStr,
			},
			CacheInitialization: DatabaseCacheInitializer{
				Query:   tt.cacheInitQuery,
				Timeout: tt.cacheInitTimeout,
			},
			PollUpdates: DatabaseUpdatePolling{
				Query:       tt.cacheUpdateQuery,
				RefreshRate: tt.cacheUpdateRefreshRate,
				Timeout:     tt.cacheUpdateTimeout,
			},
		}

		errs := dbConfig.validate(RequestDataType, tt.existingErrors)
		assert.Equal(t, tt.wantErrorCount, len(errs), tt.description)
	}
}

func assertErrsExist(t *testing.T, err []error) {
	t.Helper()
	if len(err) == 0 {
		t.Error("Expected error was not not found.")
	}
}

func assertNoErrs(t *testing.T, err []error) {
	t.Helper()
	if len(err) > 0 {
		t.Errorf("Got unexpected error(s): %v", err)
	}
}

func assertStringsEqual(t *testing.T, actual string, expected string) {
	if actual != expected {
		t.Errorf("Queries did not match.\n\"%s\" -- expected\n\"%s\" -- actual", expected, actual)

	}
}

func TestResolveConfig(t *testing.T) {
	cfg := &Configuration{
		StoredRequests: StoredRequests{
			Files: FileFetcherConfig{
				Enabled: true,
				Path:    "/test-path"},
			Database: DatabaseConfig{
				ConnectionInfo: DatabaseConnection{
					Driver:   "postgres",
					Database: "db",
					Host:     "pghost",
					Port:     5,
					Username: "user",
					Password: "pass",
				},
				FetcherQueries: DatabaseFetcherQueries{
					AmpQueryTemplate: "amp-fetcher-query",
				},
				CacheInitialization: DatabaseCacheInitializer{
					AmpQuery: "amp-cache-init-query",
				},
				PollUpdates: DatabaseUpdatePolling{
					AmpQuery: "amp-poll-query",
				},
			},
			HTTP: HTTPFetcherConfig{
				AmpEndpoint: "amp-http-fetcher-endpoint",
			},
			InMemoryCache: InMemoryCache{
				Type:             "none",
				TTL:              50,
				RequestCacheSize: 1,
				ImpCacheSize:     2,
			},
			CacheEvents: CacheEventsConfig{
				Enabled: true,
			},
			HTTPEvents: HTTPEventsConfig{
				AmpEndpoint: "amp-http-events-endpoint",
			},
		},
	}

	cfg.StoredRequests.Database.FetcherQueries.QueryTemplate = "auc-fetcher-query"
	cfg.StoredRequests.Database.CacheInitialization.Query = "auc-cache-init-query"
	cfg.StoredRequests.Database.PollUpdates.Query = "auc-poll-query"
	cfg.StoredRequests.HTTP.Endpoint = "auc-http-fetcher-endpoint"
	cfg.StoredRequests.HTTPEvents.Endpoint = "auc-http-events-endpoint"

	resolvedStoredRequestsConfig(cfg)
	auc := &cfg.StoredRequests
	amp := &cfg.StoredRequestsAMP

	// Auction should have the non-amp values in it
	assertStringsEqual(t, auc.CacheEvents.Endpoint, "/storedrequests/openrtb2")

	// Amp should have the amp values in it
	assertStringsEqual(t, amp.Database.FetcherQueries.QueryTemplate, cfg.StoredRequests.Database.FetcherQueries.AmpQueryTemplate)
	assertStringsEqual(t, amp.Database.CacheInitialization.Query, cfg.StoredRequests.Database.CacheInitialization.AmpQuery)
	assertStringsEqual(t, amp.Database.PollUpdates.Query, cfg.StoredRequests.Database.PollUpdates.AmpQuery)
	assertStringsEqual(t, amp.HTTP.Endpoint, cfg.StoredRequests.HTTP.AmpEndpoint)
	assertStringsEqual(t, amp.HTTPEvents.Endpoint, cfg.StoredRequests.HTTPEvents.AmpEndpoint)
	assertStringsEqual(t, amp.CacheEvents.Endpoint, "/storedrequests/amp")
}
