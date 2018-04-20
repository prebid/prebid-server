package config

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/glog"
)

// StoredRequests configures the backend used to store requests on the server.
type StoredRequests struct {
	// Files should be true if Stored Requests should be loaded from the filesystem.
	Files bool `mapstructure:"filesystem"`
	// Postgres configures an instance of stored_requests/backends/db_fetcher/postgres.go.
	// If non-nil, Stored Requests will be fetched from a postgres DB.
	Postgres *PostgresFetcherConfig `mapstructure:"postgres"`
	// HTTP configures an instance of stored_requests/backends/http/http_fetcher.go.
	// If non-nil, Stored Requests will be fetched from the endpoint described there.
	HTTP *HTTPFetcherConfig `mapstructure:"http"`
	// InMemoryCache configures an instance of stored_requests/caches/memory/cache.go.
	// If non-nil, Stored Requests will be saved in an in-memory cache.
	InMemoryCache *InMemoryCache `mapstructure:"in_memory_cache"`
	// PostgresEvents configures an instance of stored_requests/events/postgres/postgres.go.
	// If non-nil, Caches will be updated or invalidated based on changes to the Postgres data.
	PostgresEvents *PostgresEventsConfig `mapstructure:"postgres_events"`
	// CacheEventsAPI configures an instance of stored_requests/events/api/api.go.
	// If non-nil, Stored Request Caches can be updated or invalidated through API endpoints.
	// This is intended to be a useful development tool and not recommended for a production environment.
	// It should not be exposed to public networks without authentication.
	CacheEventsAPI bool `mapstructure:"cache_events_api"`
	// HTTPEvents configures an instance of stored_requests/events/http/http.go.
	// If non-nil, the server will use those endpoints to populate and update the cache.
	HTTPEvents *HTTPEventsConfig `mapstructure:"http_events"`
}

// HTTPEventsConfig configures stored_requests/events/http/http.go
type HTTPEventsConfig struct {
	AmpEndpoint string `mapstructure:"amp_endpoint"`
	Endpoint    string `mapstructure:"endpoint"`
	RefreshRate int64  `mapstructure:"refresh_rate_seconds"`
	Timeout     int    `mapstructure:"timeout_ms"`
}

// HTTPFetcherConfig configures a stored_requests/backends/http_fetcher/fetcher.go
type HTTPFetcherConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	AmpEndpoint string `mapstructure:"amp_endpoint"`
}

func (cfg *StoredRequests) validate() error {
	if cfg.InMemoryCache == nil {
		if cfg.CacheEventsAPI {
			return errors.New("stored_requests.cache_events_api requires a configured in_memory_cache")
		}

		if cfg.HTTPEvents != nil {
			return errors.New("stored_requests.http_events requires a configured in_memory_cache")
		}

		if cfg.PostgresEvents != nil {
			return errors.New("stored_requests.postgres_events requires a configured in_memory_cache")
		}
	}

	if err := cfg.PostgresEvents.validate(); err != nil {
		return err
	}

	return cfg.InMemoryCache.validate()
}

type PostgresFetcherConfig struct {
	ConnectionInfo PostgresConnection `mapstructure:"connection"`
	Queries        PostgresQueries    `mapstructure:"queries"`
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

type PostgresQueries struct {
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
	QueryTemplate string `mapstructure:"openrtb2"`

	// AmpQueryTemplate is the same as QueryTemplate, but used in the `/openrtb2/amp` endpoint.
	AmpQueryTemplate string `mapstructure:"amp"`
}

type PostgresEventsConfig struct {
	ConnectionInfo       PostgresConnection `mapstructure:"connection"`
	MinReconnectInterval int                `mapstructure:"min_reconnect_interval_ms"`
	MaxReconnectInterval int                `mapstructure:"max_reconnect_interval_ms"`
	ORTBChannel          string             `mapstructure:"openrtb2_channel"`
	AMPChannel           string             `mapstructure:"amp_channel"`
}

func (cfg *PostgresEventsConfig) validate() error {
	if cfg == nil {
		return nil
	}

	if cfg.MinReconnectInterval <= 0 {
		return fmt.Errorf("stored_requests.postgres_events.min_reconnect_interval_ms must be positive. Got %d", cfg.MinReconnectInterval)
	}

	if cfg.MaxReconnectInterval <= 0 {
		return fmt.Errorf("stored_requests.postgres_events.max_reconnect_interval_ms must be positive. Got %d", cfg.MaxReconnectInterval)
	}

	if cfg.MaxReconnectInterval < cfg.MinReconnectInterval {
		return fmt.Errorf("stored_requests.postgres_events.max_reconnect_interval_ms must be greater than min_reconnect_interval_ms, but %d > %d", cfg.MinReconnectInterval, cfg.MaxReconnectInterval)
	}

	// Per the docs, a channel can be "any identifier". An identifier is:
	//
	// - must begin with a letter (a-z, but also letters with diacritical marks and non-Latin letters) or an underscore (_).
	// - can contain letters, underscores, digits (0-9), or dollar signs ($).
	//
	if cfg.ORTBChannel == "" {
		return errors.New("stored_requests.postgres_events.openrtb2_channel must not be an empty string.")
	}

	if cfg.AMPChannel == "" {
		return errors.New("stored_requests.postgres_events.amp_channel must not be an empty string.")
	}

	return nil
}

// MakeQuery builds a query which can fetch numReqs Stored Requetss and numImps Stored Imps.
// See the docs on PostgresConfig.QueryTemplate for a description of how it works.
func (cfg *PostgresQueries) MakeQuery(numReqs int, numImps int) (query string) {
	return resolve(cfg.QueryTemplate, numReqs, numImps)
}

// MakeAmpQuery is the equivalent of MakeQuery() for AMP.
func (cfg *PostgresQueries) MakeAmpQuery(numReqs int, numImps int) string {
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
	// TTL is the maximum number of seconds that an unused value will stay in the cache.
	// TTL <= 0 can be used for "no ttl". Elements will still be evicted based on the Size.
	TTL int `mapstructure:"ttl_seconds"`
	// RequestCacheSize is the max number of bytes allowed in the cache for Stored Requests. Values <= 0 will have no limit
	RequestCacheSize int `mapstructure:"request_cache_size_bytes"`
	// ImpCacheSize is the max number of bytes allowed in the cache for Stored Imps. Values <= 0 will have no limit
	ImpCacheSize int `mapstructure:"imp_cache_size_bytes"`
}

func (inMemCache *InMemoryCache) validate() error {
	if inMemCache == nil {
		return nil
	}

	if inMemCache.TTL > 0 && (inMemCache.RequestCacheSize <= 0 || inMemCache.ImpCacheSize <= 0) {
		return fmt.Errorf("Stored Request In-Memory caches don't yet support TTLs with no max size. PRs for this are welcome. Given: TTL=%d, request-cache-size=%d, imp-cache-size=%d.", inMemCache.TTL, inMemCache.RequestCacheSize, inMemCache.ImpCacheSize)
	}
	return nil
}
