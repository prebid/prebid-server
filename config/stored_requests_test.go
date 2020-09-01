package config

import (
	"strconv"
	"strings"
	"testing"
)

const sampleQueryTemplate = "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in %REQUEST_ID_LIST% UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in %IMP_ID_LIST%"

func TestNormalQueryMaker(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 1, 3)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in ($1) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in ($2, $3, $4)")
}
func TestQueryMakerManyImps(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 1, 11)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in ($1) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in ($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)")
}

func TestQueryMakerNoRequests(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 0, 3)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in (NULL) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in ($1, $2, $3)")
}

func TestQueryMakerNoImps(t *testing.T) {
	madeQuery := buildQuery(sampleQueryTemplate, 1, 0)
	assertStringsEqual(t, madeQuery, "SELECT id, requestData, 'request' as type FROM stored_requests WHERE id in ($1) UNION ALL SELECT id, impData, 'imp' as type FROM stored_requests WHERE id in (NULL)")
}

func TestQueryMakerMultilists(t *testing.T) {
	madeQuery := buildQuery("SELECT id, config FROM table WHERE id in %IMP_ID_LIST% UNION ALL SELECT id, config FROM other_table WHERE id in %IMP_ID_LIST%", 0, 3)
	assertStringsEqual(t, madeQuery, "SELECT id, config FROM table WHERE id in ($1, $2, $3) UNION ALL SELECT id, config FROM other_table WHERE id in ($1, $2, $3)")
}

func TestQueryMakerNegative(t *testing.T) {
	query := buildQuery(sampleQueryTemplate, -1, -2)
	expected := buildQuery(sampleQueryTemplate, 0, 0)
	assertStringsEqual(t, query, expected)
}

func TestPostgressConnString(t *testing.T) {
	db := "TestDB"
	host := "somehost.com"
	port := 20
	username := "someuser"
	password := "somepassword"

	cfg := PostgresConnection{
		Database: db,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
	}

	dataSourceName := cfg.ConnString()
	paramList := strings.Split(dataSourceName, " ")
	params := make(map[string]string, len(paramList))
	for _, param := range paramList {
		keyVals := strings.Split(param, "=")
		if len(keyVals) != 2 {
			t.Fatalf(`param "%s" must only have one equals sign`, param)
		}
		if _, ok := params[keyVals[0]]; ok {
			t.Fatalf("found duplicate param at key %s", keyVals[0])
		}
		params[keyVals[0]] = keyVals[1]
	}

	assertHasValue(t, params, "dbname", db)
	assertHasValue(t, params, "host", host)
	assertHasValue(t, params, "port", strconv.Itoa(port))
	assertHasValue(t, params, "user", username)
	assertHasValue(t, params, "password", password)
	assertHasValue(t, params, "sslmode", "disable")
}

func TestInMemoryCacheValidation(t *testing.T) {
	assertNoErrs(t, (&InMemoryCache{
		Type: "unbounded",
	}).validate("Test", nil))
	assertNoErrs(t, (&InMemoryCache{
		Type: "none",
	}).validate("Test", nil))
	assertNoErrs(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 1000,
		ImpCacheSize:     1000,
	}).validate("Test", nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "unrecognized",
	}).validate("Test", nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:         "unbounded",
		ImpCacheSize: 1000,
	}).validate("Test", nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "unbounded",
		RequestCacheSize: 1000,
	}).validate("Test", nil))
	assertErrsExist(t, (&InMemoryCache{
		Type: "unbounded",
		TTL:  500,
	}).validate("Test", nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 0,
		ImpCacheSize:     1000,
	}).validate("Test", nil))
	assertErrsExist(t, (&InMemoryCache{
		Type:             "lru",
		RequestCacheSize: 1000,
		ImpCacheSize:     0,
	}).validate("Test", nil))
}

func assertErrsExist(t *testing.T, err configErrors) {
	t.Helper()
	if len(err) == 0 {
		t.Error("Expected error was not not found.")
	}
}

func assertNoErrs(t *testing.T, err configErrors) {
	t.Helper()
	if len(err) > 0 {
		t.Errorf("Got unexpected error(s): %v", err)
	}
}

func assertHasValue(t *testing.T, m map[string]string, key string, val string) {
	t.Helper()
	realVal, ok := m[key]
	if !ok {
		t.Errorf("Map missing required key: %s", key)
	}
	if val != realVal {
		t.Errorf("Unexpected value at key %s. Expected %s, Got %s", key, val, realVal)
	}
}

func buildQuery(template string, numReqs int, numImps int) string {
	cfg := PostgresFetcherQueries{}
	cfg.QueryTemplate = template

	return cfg.MakeQuery(numReqs, numImps)
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
			Postgres: PostgresConfig{
				ConnectionInfo: PostgresConnection{
					Database: "db",
					Host:     "pghost",
					Port:     5,
					Username: "user",
					Password: "pass",
				},
				FetcherQueries: PostgresFetcherQueries{
					AmpQueryTemplate: "amp-fetcher-query",
				},
				CacheInitialization: PostgresCacheInitializer{
					AmpQuery: "amp-cache-init-query",
				},
				PollUpdates: PostgresUpdatePolling{
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

	cfg.StoredRequests.Postgres.FetcherQueries.QueryTemplate = "auc-fetcher-query"
	cfg.StoredRequests.Postgres.CacheInitialization.Query = "auc-cache-init-query"
	cfg.StoredRequests.Postgres.PollUpdates.Query = "auc-poll-query"
	cfg.StoredRequests.HTTP.Endpoint = "auc-http-fetcher-endpoint"
	cfg.StoredRequests.HTTPEvents.Endpoint = "auc-http-events-endpoint"

	resolvedStoredRequestsConfig(cfg)
	auc := &cfg.StoredRequests
	amp := &cfg.StoredRequestsAMP

	// Auction should have the non-amp values in it
	assertStringsEqual(t, auc.CacheEvents.Endpoint, "/storedrequests/openrtb2")

	// Amp should have the amp values in it
	assertStringsEqual(t, amp.Postgres.FetcherQueries.QueryTemplate, cfg.StoredRequests.Postgres.FetcherQueries.AmpQueryTemplate)
	assertStringsEqual(t, amp.Postgres.CacheInitialization.Query, cfg.StoredRequests.Postgres.CacheInitialization.AmpQuery)
	assertStringsEqual(t, amp.Postgres.PollUpdates.Query, cfg.StoredRequests.Postgres.PollUpdates.AmpQuery)
	assertStringsEqual(t, amp.HTTP.Endpoint, cfg.StoredRequests.HTTP.AmpEndpoint)
	assertStringsEqual(t, amp.HTTPEvents.Endpoint, cfg.StoredRequests.HTTPEvents.AmpEndpoint)
	assertStringsEqual(t, amp.CacheEvents.Endpoint, "/storedrequests/amp")
}
