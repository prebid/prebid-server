package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
)

// DataType constants
type DataType string

const (
	RequestDataType    DataType = "Request"
	CategoryDataType   DataType = "Category"
	VideoDataType      DataType = "Video"
	AMPRequestDataType DataType = "AMP Request"
	AccountDataType    DataType = "Account"
	ResponseDataType   DataType = "Response"
)

// Section returns the config section this type is defined in
func (dataType DataType) Section() string {
	return map[DataType]string{
		RequestDataType:    "stored_requests",
		CategoryDataType:   "categories",
		VideoDataType:      "stored_video_req",
		AMPRequestDataType: "stored_amp_req",
		AccountDataType:    "accounts",
		ResponseDataType:   "stored_responses",
	}[dataType]
}

// Section returns the config section
func (sr *StoredRequests) Section() string {
	return sr.dataType.Section()
}

// DataType returns the DataType associated with this config
func (sr *StoredRequests) DataType() DataType {
	return sr.dataType
}

// SetDataType sets the DataType on this config. Needed for tests.
func (sr *StoredRequests) SetDataType(dataType DataType) {
	sr.dataType = dataType
}

// StoredRequests struct defines options for stored requests for each data type
// including some amp stored_requests options
type StoredRequests struct {
	// dataType is a tag pushed from upstream indicating the type of object fetched here
	dataType DataType
	// Files should be used if Stored Requests should be loaded from the filesystem.
	// Fetchers are in stored_requests/backends/file_system/fetcher.go
	Files FileFetcherConfig `mapstructure:"filesystem"`
	// Database configures Fetchers and EventProducers which read from a Database.
	// Fetchers are in stored_requests/backends/db_fetcher/fetcher.go
	// EventProducers are in stored_requests/events/database
	Database DatabaseConfig `mapstructure:"database"`
	// HTTP configures an instance of stored_requests/backends/http/http_fetcher.go.
	// If non-nil, Stored Requests will be fetched from the endpoint described there.
	HTTP HTTPFetcherConfig `mapstructure:"http"`
	// InMemoryCache configures an instance of stored_requests/caches/memory/cache.go.
	// If non-nil, Stored Requests will be saved in an in-memory cache.
	InMemoryCache InMemoryCache `mapstructure:"in_memory_cache"`
	// CacheEvents configures an instance of stored_requests/events/api/api.go.
	// This is a sub-object containing the endpoint name to use for this API endpoint.
	CacheEvents CacheEventsConfig `mapstructure:"cache_events"`
	// HTTPEvents configures an instance of stored_requests/events/http/http.go.
	// If non-nil, the server will use those endpoints to populate and update the cache.
	HTTPEvents HTTPEventsConfig `mapstructure:"http_events"`
}

// HTTPEventsConfig configures stored_requests/events/http/http.go
type HTTPEventsConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	RefreshRate int64  `mapstructure:"refresh_rate_seconds"`
	Timeout     int    `mapstructure:"timeout_ms"`
	AmpEndpoint string `mapstructure:"amp_endpoint"`
}

func (cfg HTTPEventsConfig) TimeoutDuration() time.Duration {
	return time.Duration(cfg.Timeout) * time.Millisecond
}

func (cfg HTTPEventsConfig) RefreshRateDuration() time.Duration {
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

// HTTPFetcherConfig configures a stored_requests/backends/http_fetcher/fetcher.go
type HTTPFetcherConfig struct {
	Endpoint               string `mapstructure:"endpoint"`
	AmpEndpoint            string `mapstructure:"amp_endpoint"`
	UseRfcCompliantBuilder bool   `mapstructure:"use_rfc3986_compliant_request_builder"`
}

// Migrate combined stored_requests+amp configuration to separate simple config sections
func resolvedStoredRequestsConfig(cfg *Configuration) {
	sr := &cfg.StoredRequests
	amp := &cfg.StoredRequestsAMP

	sr.CacheEvents.Endpoint = "/storedrequests/openrtb2" // why is this here and not SetDefault ?

	// Amp uses the same config but some fields get replaced by Amp* version of similar fields
	cfg.StoredRequestsAMP = cfg.StoredRequests
	amp.Database.FetcherQueries.QueryTemplate = sr.Database.FetcherQueries.AmpQueryTemplate
	amp.Database.CacheInitialization.Query = sr.Database.CacheInitialization.AmpQuery
	amp.Database.PollUpdates.Query = sr.Database.PollUpdates.AmpQuery
	amp.HTTP.Endpoint = sr.HTTP.AmpEndpoint
	amp.CacheEvents.Endpoint = "/storedrequests/amp"
	amp.HTTPEvents.Endpoint = sr.HTTPEvents.AmpEndpoint

	// Set data types for each section
	cfg.StoredRequests.dataType = RequestDataType
	cfg.StoredRequestsAMP.dataType = AMPRequestDataType
	cfg.StoredVideo.dataType = VideoDataType
	cfg.CategoryMapping.dataType = CategoryDataType
	cfg.Accounts.dataType = AccountDataType
	cfg.StoredResponses.dataType = ResponseDataType
}

func (cfg *StoredRequests) validate(errs []error) []error {
	if cfg.DataType() == AccountDataType && cfg.Database.ConnectionInfo.Database != "" {
		errs = append(errs, fmt.Errorf("%s.database: retrieving accounts via database not available, use accounts.files", cfg.Section()))
	} else {
		errs = cfg.Database.validate(cfg.DataType(), errs)
	}

	// Categories do not use cache so none of the following checks apply
	if cfg.DataType() == CategoryDataType {
		return errs
	}

	if cfg.InMemoryCache.Type == "none" {
		if cfg.CacheEvents.Enabled {
			errs = append(errs, fmt.Errorf("%s: cache_events must be disabled if in_memory_cache=none", cfg.Section()))
		}

		if cfg.HTTPEvents.RefreshRate != 0 {
			errs = append(errs, fmt.Errorf("%s: http_events.refresh_rate_seconds must be 0 if in_memory_cache=none", cfg.Section()))
		}

		if cfg.Database.PollUpdates.Query != "" {
			errs = append(errs, fmt.Errorf("%s: database.poll_for_updates.query must be empty if in_memory_cache=none", cfg.Section()))
		}
		if cfg.Database.CacheInitialization.Query != "" {
			errs = append(errs, fmt.Errorf("%s: database.initialize_caches.query must be empty if in_memory_cache=none", cfg.Section()))
		}
	}
	errs = cfg.InMemoryCache.validate(cfg.DataType(), errs)
	return errs
}

// DatabaseConfig configures the Stored Request ecosystem to use Database. This must include a Fetcher,
// and may optionally include some EventProducers to populate and refresh the caches.
type DatabaseConfig struct {
	ConnectionInfo      DatabaseConnection       `mapstructure:"connection"`
	FetcherQueries      DatabaseFetcherQueries   `mapstructure:"fetcher"`
	CacheInitialization DatabaseCacheInitializer `mapstructure:"initialize_caches"`
	PollUpdates         DatabaseUpdatePolling    `mapstructure:"poll_for_updates"`
}

func (cfg *DatabaseConfig) validate(dataType DataType, errs []error) []error {
	if cfg.ConnectionInfo.Database == "" {
		return errs
	}

	errs = cfg.CacheInitialization.validate(dataType, errs)
	errs = cfg.PollUpdates.validate(dataType, errs)
	return errs
}

// DatabaseConnection has options which put types to the Database Connection string. See:
// https://godoc.org/github.com/lib/pq#hdr-Connection_String_Parameters
type DatabaseConnection struct {
	Driver      string `mapstructure:"driver"`
	Database    string `mapstructure:"dbname"`
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	Username    string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	QueryString string `mapstructure:"query_string"`
	TLS         TLS    `mapstructure:"tls"`
}

type TLS struct {
	RootCert   string `mapstructure:"root_cert"`
	ClientCert string `mapstructure:"client_cert"`
	ClientKey  string `mapstructure:"client_key"`
}

type DatabaseFetcherQueries struct {
	// QueryTemplate is the Database Query which can be used to fetch configs from the database.
	// It is a Template, rather than a full Query, because a single HTTP request may reference multiple Stored Requests.
	//
	// In the simplest case, this could be something like:
	//   SELECT id, requestData, 'request' as type
	//     FROM stored_requests
	//     WHERE id in $REQUEST_ID_LIST
	//     UNION ALL
	//   SELECT id, impData, 'imp' as type
	//     FROM stored_imps
	//     WHERE id in $IMP_ID_LIST
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

	// AmpQueryTemplate is the same as QueryTemplate, but used in the `/openrtb2/amp` endpoint.
	AmpQueryTemplate string `mapstructure:"amp_query"`
}

type DatabaseCacheInitializer struct {
	Timeout int `mapstructure:"timeout_ms"`
	// Query should be something like:
	//
	// SELECT id, requestData, 'request' AS type FROM stored_requests
	// UNION ALL
	// SELECT id, impData, 'imp' AS type FROM stored_imps
	//
	// This query will be run once on startup to fetch _all_ known Stored Request data from the database.
	//
	// For more details on the expected format of requestData and impData, see stored_requests/events/database/database.go
	Query string `mapstructure:"query"`
	// AmpQuery is just like Query, but for AMP Stored Requests
	AmpQuery string `mapstructure:"amp_query"`
}

func (cfg *DatabaseCacheInitializer) validate(dataType DataType, errs []error) []error {
	section := dataType.Section()
	if cfg.Query == "" {
		return errs
	}
	if cfg.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("%s: database.initialize_caches.timeout_ms must be positive", section))
	}
	if strings.Contains(cfg.Query, "$") {
		errs = append(errs, fmt.Errorf("%s: database.initialize_caches.query should not contain any wildcards denoted by $ (e.g. $LAST_UPDATED)", section))
	}
	return errs
}

type DatabaseUpdatePolling struct {
	// RefreshRate determines how frequently the Query and AmpQuery are run.
	RefreshRate int `mapstructure:"refresh_rate_seconds"`

	// Timeout is the amount of time before a call to the database is aborted.
	Timeout int `mapstructure:"timeout_ms"`

	// An example UpdateQuery is:
	//
	// SELECT id, requestData, 'request' AS type
	//   FROM stored_requests
	//   WHERE last_updated > $LAST_UPDATED
	// UNION ALL
	// SELECT id, impData, 'imp' AS type
	//   FROM stored_imps
	//   WHERE last_updated > $LAST_UPDATED
	//
	// The code will be run periodically to fetch updates from the database.
	Query string `mapstructure:"query"`
	// AmpQuery is the same as Query, but used for the `/openrtb2/amp` endpoint.
	AmpQuery string `mapstructure:"amp_query"`
}

func (cfg *DatabaseUpdatePolling) validate(dataType DataType, errs []error) []error {
	section := dataType.Section()
	if cfg.Query == "" {
		return errs
	}

	if cfg.RefreshRate <= 0 {
		errs = append(errs, fmt.Errorf("%s: database.poll_for_updates.refresh_rate_seconds must be > 0", section))
	}

	if cfg.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("%s: database.poll_for_updates.timeout_ms must be > 0", section))
	}
	if !strings.Contains(cfg.Query, "$LAST_UPDATED") {
		errs = append(errs, fmt.Errorf("%s: database.poll_for_updates.query must contain $LAST_UPDATED parameter", section))
	}

	return errs
}

type InMemoryCache struct {
	// Identify the type of memory cache. "none", "unbounded", "lru"
	Type string `mapstructure:"type"`
	// TTL is the maximum number of seconds that an unused value will stay in the cache.
	// TTL <= 0 can be used for "no ttl". Elements will still be evicted based on the Size.
	TTL int `mapstructure:"ttl_seconds"`
	// Size is the max total cache size allowed for single caches
	Size int `mapstructure:"size_bytes"`
	// RequestCacheSize is the max number of bytes allowed in the cache for Stored Requests. Values <= 0 will have no limit
	RequestCacheSize int `mapstructure:"request_cache_size_bytes"`
	// ImpCacheSize is the max number of bytes allowed in the cache for Stored Imps. Values <= 0 will have no limit
	ImpCacheSize int `mapstructure:"imp_cache_size_bytes"`
	// ResponsesCacheSize is the max number of bytes allowed in the cache for Stored Responses. Values <= 0 will have no limit
	RespCacheSize int `mapstructure:"resp_cache_size_bytes"`
}

func (cfg *InMemoryCache) validate(dataType DataType, errs []error) []error {
	section := dataType.Section()
	switch cfg.Type {
	case "none":
		// No errors for no config options
	case "unbounded":
		if cfg.TTL != 0 {
			errs = append(errs, fmt.Errorf("%s: in_memory_cache.ttl_seconds is not supported for unbounded caches. Got %d", section, cfg.TTL))
		}
		if dataType == AccountDataType {
			// single cache
			if cfg.Size != 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.size_bytes is not supported for unbounded caches. Got %d", section, cfg.Size))
			}
		} else {
			// dual (request and imp) caches
			if cfg.RequestCacheSize != 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.request_cache_size_bytes is not supported for unbounded caches. Got %d", section, cfg.RequestCacheSize))
			}
			if cfg.ImpCacheSize != 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.imp_cache_size_bytes is not supported for unbounded caches. Got %d", section, cfg.ImpCacheSize))
			}
			if cfg.RespCacheSize != 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.resp_cache_size_bytes is not supported for unbounded caches. Got %d", section, cfg.RespCacheSize))
			}
		}
	case "lru":
		if dataType == AccountDataType {
			// single cache
			if cfg.Size <= 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.size_bytes must be >= 0 when in_memory_cache.type=lru. Got %d", section, cfg.Size))
			}
			if cfg.RequestCacheSize > 0 || cfg.ImpCacheSize > 0 || cfg.RespCacheSize > 0 {
				glog.Warningf("%s: in_memory_cache.request_cache_size_bytes, imp_cache_size_bytes and resp_cache_size_bytes do not apply to this section and will be ignored", section)
			}
		} else {
			// dual (request and imp) caches
			if cfg.RequestCacheSize <= 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.request_cache_size_bytes must be >= 0 when in_memory_cache.type=lru. Got %d", section, cfg.RequestCacheSize))
			}
			if cfg.ImpCacheSize <= 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.imp_cache_size_bytes must be >= 0 when in_memory_cache.type=lru. Got %d", section, cfg.ImpCacheSize))
			}
			if cfg.RespCacheSize <= 0 {
				errs = append(errs, fmt.Errorf("%s: in_memory_cache.resp_cache_size_bytes must be >= 0 when in_memory_cache.type=lru. Got %d", section, cfg.RespCacheSize))
			}
			if cfg.Size > 0 {
				glog.Warningf("%s: in_memory_cache.size_bytes does not apply in this section and will be ignored", section)
			}
		}
	default:
		errs = append(errs, fmt.Errorf("%s: in_memory_cache.type %s is invalid", section, cfg.Type))
	}
	return errs
}
