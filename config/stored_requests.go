package config

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
)

// StoredRequests configures the backend used to store requests on the server.
type StoredRequests struct {
	// Files should be true if Stored Requests should be loaded from the filesystem.
	Files bool `mapstructure:"filesystem"`
	//If data should be loaded from file system, path should be specified in configuration
	Path string `mapstructure:"directorypath"`
	// Postgres configures Fetchers and EventProducers which read from a Postgres DB.
	// Fetchers are in stored_requests/backends/db_fetcher/postgres.go
	// EventProducers are in stored_requests/events/postgres
	Postgres PostgresConfig `mapstructure:"postgres"`
	// HTTP configures an instance of stored_requests/backends/http/http_fetcher.go.
	// If non-nil, Stored Requests will be fetched from the endpoint described there.
	HTTP HTTPFetcherConfig `mapstructure:"http"`
	// InMemoryCache configures an instance of stored_requests/caches/memory/cache.go.
	// If non-nil, Stored Requests will be saved in an in-memory cache.
	InMemoryCache InMemoryCache `mapstructure:"in_memory_cache"`
	// CacheEventsAPI configures an instance of stored_requests/events/api/api.go.
	// If non-nil, Stored Request Caches can be updated or invalidated through API endpoints.
	// This is intended to be a useful development tool and not recommended for a production environment.
	// It should not be exposed to public networks without authentication.
	CacheEventsAPI bool `mapstructure:"cache_events_api"`
	// HTTPEvents configures an instance of stored_requests/events/http/http.go.
	// If non-nil, the server will use those endpoints to populate and update the cache.
	HTTPEvents HTTPEventsConfig `mapstructure:"http_events"`
}

// StoredRequestsSlim struct defines options for stored requests from a single endpoint
type StoredRequestsSlim struct {
	// Files should be used if Stored Requests should be loaded from the filesystem.
	// Fetchers are in stored_requests/backends/file_system/fetcher.go
	Files FileFetcherConfig `mapstructure:"filesystem"`
	// Postgres configures Fetchers and EventProducers which read from a Postgres DB.
	// Fetchers are in stored_requests/backends/db_fetcher/postgres.go
	// EventProducers are in stored_requests/events/postgres
	Postgres PostgresConfigSlim `mapstructure:"postgres"`
	// HTTP configures an instance of stored_requests/backends/http/http_fetcher.go.
	// If non-nil, Stored Requests will be fetched from the endpoint described there.
	HTTP HTTPFetcherConfigSlim `mapstructure:"http"`
	// InMemoryCache configures an instance of stored_requests/caches/memory/cache.go.
	// If non-nil, Stored Requests will be saved in an in-memory cache.
	InMemoryCache InMemoryCache `mapstructure:"in_memory_cache"`
	// CacheEvents configures an instance of stored_requests/events/api/api.go.
	// This is a sub-object containing the endpoint name to use for this API endpoint.
	CacheEvents CacheEventsConfig `mapstructure:"cache_events"`
	// HTTPEvents configures an instance of stored_requests/events/http/http.go.
	// If non-nil, the server will use those endpoints to populate and update the cache.
	HTTPEvents HTTPEventsConfigSlim `mapstructure:"http_events"`
}

// HTTPEventsConfigSlim configures stored_requests/events/http/http.go
type HTTPEventsConfigSlim struct {
	Endpoint    string `mapstructure:"endpoint"`
	RefreshRate int64  `mapstructure:"refresh_rate_seconds"`
	Timeout     int    `mapstructure:"timeout_ms"`
}

// HTTPEventsConfig configures stored_requests/events/http/http.go
type HTTPEventsConfig struct {
	HTTPEventsConfigSlim
	AmpEndpoint string `mapstructure:"amp_endpoint"`
}

func (cfg HTTPEventsConfigSlim) TimeoutDuration() time.Duration {
	return time.Duration(cfg.Timeout) * time.Millisecond
}

func (cfg HTTPEventsConfigSlim) RefreshRateDuration() time.Duration {
	return time.Duration(cfg.RefreshRate) * time.Second
}

// CacheEventsConfig configured stored_requests/events/api/api.go
type CacheEventsConfig struct {
	// Enabled should be true to enable the events api endpoint
	Enabled bool `mapstructure:"enabled"`
	// Endpoint is the url path exposed for this stored requests events api
	Endpoint string `mapstructure:"endpoint"`
}

// FileFetcherConfig configures a stored_requests/backends/file_fetcher/fetcher.go
type FileFetcherConfig struct {
	// Enabled should be true if Stored Requests should be loaded from the filesystem.
	Enabled bool `mapstructure:"enabled"`
	// Path to the directory this file fetcher gets data from.
	Path string `mapstructure:"directorypath"`
}

// HTTPFetcherConfigSlim configures a stored_requests/backends/http_fetcher/fetcher.go
type HTTPFetcherConfigSlim struct {
	Endpoint string `mapstructure:"endpoint"`
}

// HTTPFetcherConfig configures a stored_requests/backends/http_fetcher/fetcher.go
type HTTPFetcherConfig struct {
	HTTPFetcherConfigSlim
	AmpEndpoint string `mapstructure:"amp_endpoint"`
}

func (cfg *StoredRequests) validate(errs configErrors) configErrors {
	if cfg.InMemoryCache.Type == "none" {
		if cfg.CacheEventsAPI {
			errs = append(errs, errors.New("stored_requests.cache_events_api must be false if stored_requests.in_memory_cache=none"))
		}

		if cfg.HTTPEvents.RefreshRate != 0 {
			errs = append(errs, errors.New("stored_requests.http_events.refresh_rate_seconds must be 0 if stored_requests.in_memory_cache=none"))
		}

		if cfg.Postgres.PollUpdates.Query != "" {
			errs = append(errs, errors.New("stored_requests.postgres.poll_for_updates.query must be empty if stored_requests.in_memory_cache=none"))
		}
		if cfg.Postgres.CacheInitialization.Query != "" {
			errs = append(errs, errors.New("stored_requests.postgres.initialize_caches.query must be empty if stored_requests.in_memory_cache=none"))
		}
	}
	errs = cfg.InMemoryCache.validate(errs)
	errs = cfg.Postgres.validate(errs)
	return errs
}

// PostgresConfigSlim configures the Stored Request ecosystem to use Postgres. This must include a Fetcher,
// and may optionally include some EventProducers to populate and refresh the caches.
type PostgresConfigSlim struct {
	ConnectionInfo      PostgresConnection           `mapstructure:"connection"`
	FetcherQueries      PostgresFetcherQueriesSlim   `mapstructure:"fetcher"`
	CacheInitialization PostgresCacheInitializerSlim `mapstructure:"initialize_caches"`
	PollUpdates         PostgresUpdatePollingSlim    `mapstructure:"poll_for_updates"`
}

// PostgresConfig configures the Stored Request ecosystem to use Postgres. This must include a Fetcher,
// and may optionally include some EventProducers to populate and refresh the caches.
type PostgresConfig struct {
	ConnectionInfo      PostgresConnection       `mapstructure:"connection"`
	FetcherQueries      PostgresFetcherQueries   `mapstructure:"fetcher"`
	CacheInitialization PostgresCacheInitializer `mapstructure:"initialize_caches"`
	PollUpdates         PostgresUpdatePolling    `mapstructure:"poll_for_updates"`
}

func (cfg *PostgresConfig) validate(errs configErrors) configErrors {
	if cfg.ConnectionInfo.Database == "" {
		return errs
	}

	return cfg.PollUpdates.validate(errs)
}

// PostgresConnection has options which put types to the Postgres Connection string. See:
// https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters
type PostgresConnection struct {
	Database string `mapstructure:"dbname"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"user"`
	Password string `mapstructure:"password"`
}

func (cfg *PostgresConnection) ConnString() string {
	buffer := bytes.NewBuffer(nil)

	if cfg.Host != "" {
		buffer.WriteString("host=")
		buffer.WriteString(cfg.Host)
		buffer.WriteString(" ")
	}

	if cfg.Port > 0 {
		buffer.WriteString("port=")
		buffer.WriteString(strconv.Itoa(cfg.Port))
		buffer.WriteString(" ")
	}

	if cfg.Username != "" {
		buffer.WriteString("user=")
		buffer.WriteString(cfg.Username)
		buffer.WriteString(" ")
	}

	if cfg.Password != "" {
		buffer.WriteString("password=")
		buffer.WriteString(cfg.Password)
		buffer.WriteString(" ")
	}

	if cfg.Database != "" {
		buffer.WriteString("dbname=")
		buffer.WriteString(cfg.Database)
		buffer.WriteString(" ")
	}

	buffer.WriteString("sslmode=disable")
	return buffer.String()
}

type PostgresFetcherQueries struct {
	PostgresFetcherQueriesSlim

	// AmpQueryTemplate is the same as QueryTemplate, but used in the `/openrtb2/amp` endpoint.
	AmpQueryTemplate string `mapstructure:"amp_query"`
}

type PostgresFetcherQueriesSlim struct {
	// QueryTemplate is the Postgres Query which can be used to fetch configs from the database.
	// It is a Template, rather than a full Query, because a single HTTP request may reference multiple Stored Requests.
	//
	// In the simplest case, this could be something like:
	//   SELECT id, requestData, 'request' as type
	//     FROM stored_requests
	//     WHERE id in %REQUEST_ID_LIST%
	//     UNION ALL
	//   SELECT id, impData, 'imp' as type
	//     FROM stored_imps
	//     WHERE id in %IMP_ID_LIST%
	//
	// The MakeQuery function will transform this query into:
	//   SELECT id, requestData, 'request' as type
	//     FROM stored_requests
	//     WHERE id in ($1)
	//     UNION ALL
	//   SELECT id, impData, 'imp' as type
	//     FROM stored_imps
	//     WHERE id in ($2, $3, $4, ...)
	//
	// ... where the number of "$x" args depends on how many IDs are nested within the HTTP request.
	QueryTemplate string `mapstructure:"query"`
}

type PostgresCacheInitializer struct {
	PostgresCacheInitializerSlim
	// AmpQuery is just like Query, but for AMP Stored Requests
	AmpQuery string `mapstructure:"amp_query"`
}

type PostgresCacheInitializerSlim struct {
	Timeout int `mapstructure:"timeout_ms"`
	// Query should be something like:
	//
	// SELECT id, requestData, 'request' AS type FROM stored_requests
	// UNION ALL
	// SELECT id, impData, 'imp' AS type FROM stored_imps
	//
	// This query will be run once on startup to fetch _all_ known Stored Request data from the database.
	//
	// For more details on the expected format of requestData and impData, see stored_requests/events/postgres/polling.go
	Query string `mapstructure:"query"`
}

func (cfg *PostgresCacheInitializerSlim) validate(errs configErrors) configErrors {
	if cfg.Query == "" {
		return errs
	}
	if cfg.Timeout <= 0 {
		errs = append(errs, errors.New("stored_requests.postgres.initialize_caches.timeout_ms must be positive"))
	}
	if strings.Contains(cfg.Query, "$") {
		errs = append(errs, errors.New("stored_requests.postgres.initialize_caches.query should not contain any wildcards (e.g. $1)"))
	}
	return errs
}

func (cfg *PostgresCacheInitializer) validate(errs configErrors) configErrors {
	if cfg.Query != "" {
		errs = (&cfg.PostgresCacheInitializerSlim).validate(errs)
	} else if cfg.AmpQuery != "" {
		cfg.Query = cfg.AmpQuery
		errs = (&cfg.PostgresCacheInitializerSlim).validate(errs)
		cfg.Query = ""
	}

	return errs
}

type PostgresUpdatePollingSlim struct {
	// RefreshRate determines how frequently the Query and AmpQuery are run.
	RefreshRate int `mapstructure:"refresh_rate_seconds"`

	// Timeout is the amount of time before a call to the database is aborted.
	Timeout int `mapstructure:"timeout_ms"`

	// An example UpdateQuery is:
	//
	// SELECT id, requestData, 'request' AS type
	//   FROM stored_requests
	//   WHERE last_updated > $1
	// UNION ALL
	// SELECT id, requestData, 'imp' AS type
	//   FROM stored_imps
	//   WHERE last_updated > $1
	//
	// The code will be run periodically to fetch updates from the database.
	Query string `mapstructure:"query"`
}

type PostgresUpdatePolling struct {
	PostgresUpdatePollingSlim
	// AmpQuery is the same as Query, but used for the `/openrtb2/amp` endpoint.
	AmpQuery string `mapstructure:"amp_query"`
}

func (cfg *PostgresUpdatePollingSlim) validate(errs configErrors) configErrors {
	if cfg.Query == "" {
		return errs
	}

	if cfg.RefreshRate <= 0 {
		errs = append(errs, errors.New("stored_requests.postgres.poll_for_updates.refresh_rate_seconds must be > 0"))
	}

	if cfg.Timeout <= 0 {
		errs = append(errs, errors.New("stored_requests.postgres.poll_for_updates.timeout_ms must be > 0"))
	}

	if !strings.Contains(cfg.Query, "$1") || strings.Contains(cfg.Query, "$2") {
		errs = append(errs, errors.New("stored_requests.postgres.poll_for_updates.query must contain exactly one wildcard"))
	}
	return errs
}

func (cfg *PostgresUpdatePolling) validate(errs configErrors) configErrors {
	if cfg.Query != "" {
		errs = (&cfg.PostgresUpdatePollingSlim).validate(errs)
	} else if cfg.AmpQuery != "" {
		cfg.Query = cfg.AmpQuery
		errs = (&cfg.PostgresUpdatePollingSlim).validate(errs)
		cfg.Query = ""
	}
	return errs
}

// MakeQuery builds a query which can fetch numReqs Stored Requetss and numImps Stored Imps.
// See the docs on PostgresConfig.QueryTemplate for a description of how it works.
func (cfg *PostgresFetcherQueriesSlim) MakeQuery(numReqs int, numImps int) (query string) {
	return resolve(cfg.QueryTemplate, numReqs, numImps)
}

// MakeAmpQuery is the equivalent of MakeQuery() for AMP.
func (cfg *PostgresFetcherQueries) MakeAmpQuery(numReqs int, numImps int) string {
	return resolve(cfg.AmpQueryTemplate, numReqs, numImps)
}

func resolve(template string, numReqs int, numImps int) (query string) {
	numReqs = ensureNonNegative("Request", numReqs)
	numImps = ensureNonNegative("Imp", numImps)

	query = strings.Replace(template, "%REQUEST_ID_LIST%", makeIdList(0, numReqs), -1)
	query = strings.Replace(query, "%IMP_ID_LIST%", makeIdList(numReqs, numImps), -1)
	return
}

func ensureNonNegative(storedThing string, num int) int {
	if num < 0 {
		glog.Errorf("Can't build a SQL query for %d Stored %ss.", num, storedThing)
		return 0
	}
	return num
}

func makeIdList(numSoFar int, numArgs int) string {
	// Any empty list like "()" is illegal in Postgres. A (NULL) is the next best thing,
	// though, since `id IN (NULL)` is valid for all "id" column types, and evaluates to an empty set.
	//
	// The query plan also suggests that it's basically free:
	//
	// explain SELECT id, requestData FROM stored_requests WHERE id in %ID_LIST%;
	//
	// QUERY PLAN
	// -------------------------------------------
	// Result  (cost=0.00..0.00 rows=0 width=16)
	//	 One-Time Filter: false
	// (2 rows)
	if numArgs == 0 {
		return "(NULL)"
	}

	final := bytes.NewBuffer(make([]byte, 0, 2+4*numArgs))
	final.WriteString("(")
	for i := numSoFar + 1; i < numSoFar+numArgs; i++ {
		final.WriteString("$")
		final.WriteString(strconv.Itoa(i))
		final.WriteString(", ")
	}
	final.WriteString("$")
	final.WriteString(strconv.Itoa(numSoFar + numArgs))
	final.WriteString(")")

	return final.String()
}

type InMemoryCache struct {
	// Identify the type of memory cache. "none", "unbounded", "lru"
	Type string `mapstructure:"type"`
	// TTL is the maximum number of seconds that an unused value will stay in the cache.
	// TTL <= 0 can be used for "no ttl". Elements will still be evicted based on the Size.
	TTL int `mapstructure:"ttl_seconds"`
	// RequestCacheSize is the max number of bytes allowed in the cache for Stored Requests. Values <= 0 will have no limit
	RequestCacheSize int `mapstructure:"request_cache_size_bytes"`
	// ImpCacheSize is the max number of bytes allowed in the cache for Stored Imps. Values <= 0 will have no limit
	ImpCacheSize int `mapstructure:"imp_cache_size_bytes"`
}

func (cfg *InMemoryCache) validate(errs configErrors) configErrors {
	switch cfg.Type {
	case "none":
		// No errors for no config options
	case "unbounded":
		if cfg.TTL != 0 {
			errs = append(errs, fmt.Errorf("stored_requests.in_memory_cache must be 0 for unbounded caches. Got %d", cfg.TTL))
		}
		if cfg.RequestCacheSize != 0 {
			errs = append(errs, fmt.Errorf("stored_requests.in_memory_cache.request_cache_size_bytes must be 0 for unbounded caches. Got %d", cfg.RequestCacheSize))
		}
		if cfg.ImpCacheSize != 0 {
			errs = append(errs, fmt.Errorf("stored_requests.in_memory_cache.imp_cache_size_bytes must be 0 for unbounded caches. Got %d", cfg.ImpCacheSize))
		}
	case "lru":
		if cfg.RequestCacheSize <= 0 {
			errs = append(errs, fmt.Errorf("stored_requests.in_memory_cache.request_cache_size_bytes must be >= 0 when stored_requests.in_memory_cache.type=lru. Got %d", cfg.RequestCacheSize))
		}
		if cfg.ImpCacheSize <= 0 {
			errs = append(errs, fmt.Errorf("stored_requests.in_memory_cache.imp_cache_size_bytes must be >= 0 when stored_requests.in_memory_cache.type=lru. Got %d", cfg.ImpCacheSize))
		}
	default:
		errs = append(errs, fmt.Errorf("stored_requests.in_memory_cache.type %s is invalid", cfg.Type))
	}
	return errs
}
