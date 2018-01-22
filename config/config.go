package config

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/spf13/viper"
	"strconv"
	"strings"
)

// Configuration
type Configuration struct {
	ExternalURL     string             `mapstructure:"external_url"`
	Host            string             `mapstructure:"host"`
	Port            int                `mapstructure:"port"`
	AdminPort       int                `mapstructure:"admin_port"`
	DefaultTimeout  uint64             `mapstructure:"default_timeout_ms"`
	CacheURL        Cache              `mapstructure:"cache"`
	RecaptchaSecret string             `mapstructure:"recaptcha_secret"`
	HostCookie      HostCookie         `mapstructure:"host_cookie"`
	Metrics         Metrics            `mapstructure:"metrics"`
	DataCache       DataCache          `mapstructure:"datacache"`
	StoredRequests  StoredRequests     `mapstructure:"stored_requests"`
	Adapters        map[string]Adapter `mapstructure:"adapters"`
	MaxRequestSize  int64              `mapstructure:"max_request_size"`
}

func (cfg *Configuration) validate() error {
	return cfg.StoredRequests.validate()
}

type HostCookie struct {
	Domain       string `mapstructure:"domain"`
	Family       string `mapstructure:"family"`
	CookieName   string `mapstructure:"cookie_name"`
	OptOutURL    string `mapstructure:"opt_out_url"`
	OptInURL     string `mapstructure:"opt_in_url"`
	OptOutCookie Cookie `mapstructure:"optout_cookie"`
}

type Adapter struct {
	Endpoint    string `mapstructure:"endpoint"` // Required
	UserSyncURL string `mapstructure:"usersync_url"`
	PlatformID  string `mapstructure:"platform_id"` // needed for Facebook
	XAPI        struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Tracker  string `mapstructure:"tracker"`
	} `mapstructure:"xapi"` // needed for Rubicon
}

type Metrics struct {
	Host     string `mapstructure:"host"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type DataCache struct {
	Type       string `mapstructure:"type"`
	Filename   string `mapstructure:"filename"`
	Database   string `mapstructure:"dbname"`
	Host       string `mapstructure:"host"`
	Username   string `mapstructure:"user"`
	Password   string `mapstructure:"password"`
	CacheSize  int    `mapstructure:"cache_size"`
	TTLSeconds int    `mapstructure:"ttl_seconds"`
}

// StoredRequests configures the backend used to store requests on the server.
type StoredRequests struct {
	// Files should be true if Stored Requests should be loaded from the filesystem.
	Files bool `mapstructure:"filesystem"`
	// Postgres should be non-nil if Stored Requests should be loaded from a Postgres database.
	Postgres *PostgresConfig `mapstructure:"postgres"`
}

func (cfg *StoredRequests) validate() error {
	if cfg.Files && cfg.Postgres != nil {
		return errors.New("Only one backend from {filesystem, postgres} can be used at the same time.")
	}

	return nil
}

// PostgresConfig configures the Postgres connection for Stored Requests
type PostgresConfig struct {
	Database string `mapstructure:"dbname"`
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"user"`
	Password string `mapstructure:"password"`

	// QueryTemplate is the Postgres Query which can be used to fetch configs from the database.
	// It is a Template, rather than a full Query, because a single HTTP request may reference multiple Stored Requests.
	//
	// In the simplest case, this could be something like:
	//   SELECT id, requestData FROM stored_requests WHERE id in %ID_LIST%
	//
	// The MakeQuery function will transform this query into:
	//   SELECT id, requestData FROM stored_requests WHERE id in ($1, $2, $3, ...)
	//
	// ... where the number of "$x" args depends on how many IDs are nested within the HTTP request.
	QueryTemplate string `mapstructure:"query"`
	// Hacky way to add Amp queries that won't fit in the global query above. A better solution would
	// rename all the queries to be specific for what they are for. But that can wait for a general
	// stored requests overhaul.
	AmpQueryTemplate string `mapstructure:"amp_query"`
}

// MakeQuery gets a stored-request-fetching query which can be used to fetch numRequests requests at once.
func (cfg *PostgresConfig) MakeQuery(numRequests int) (string, error) {
	if numRequests < 1 {
		return "", fmt.Errorf("can't generate query to fetch %d stored requests", numRequests)
	}
	final := bytes.NewBuffer(make([]byte, 0, 2+4*numRequests))
	final.WriteString("(")
	for i := 1; i < numRequests; i++ {
		final.WriteString("$")
		final.WriteString(strconv.Itoa(i))
		final.WriteString(", ")
	}
	final.WriteString("$")
	final.WriteString(strconv.Itoa(numRequests))
	final.WriteString(")")
	return strings.Replace(cfg.QueryTemplate, "%ID_LIST%", final.String(), -1), nil
}

// MakeAmpQuery gets a stored-request-fetching query which can be used to fetch numRequests requests at once.
// A hacky solution for AMP, but a good solution would require an overhaul of the stored request system to
// pass around the ID type. This should probably be its own PR.
func (cfg *PostgresConfig) MakeAmpQuery(numRequests int) (string, error) {
	if numRequests < 1 {
		return "", fmt.Errorf("can't generate query to fetch %d stored requests", numRequests)
	}
	final := bytes.NewBuffer(make([]byte, 0, 2+4*numRequests))
	final.WriteString("(")
	for i := 1; i < numRequests; i++ {
		final.WriteString("$")
		final.WriteString(strconv.Itoa(i))
		final.WriteString(", ")
	}
	final.WriteString("$")
	final.WriteString(strconv.Itoa(numRequests))
	final.WriteString(")")
	return strings.Replace(cfg.AmpQueryTemplate, "%ID_LIST%", final.String(), -1), nil
}

type Cache struct {
	Scheme string `mapstructure:"scheme"`
	Host   string `mapstructure:"host"`
	Query  string `mapstructure:"query"`

	// A static timeout here is not ideal. This is a hack because we have some aggressive timelines for OpenRTB support.
	// This value specifies how much time the prebid server host expects a call to prebid cache to take.
	//
	// OpenRTB allows the caller to specify the auction timeout. Prebid Server will subtract _this_ amount of time
	// from the timeout it gives demand sources to respond.
	//
	// In reality, the cache response time will probably fluctuate with the traffic over time. Someday,
	// this should be replaced by code which tracks the response time of recent cache calls and
	// adjusts the time dynamically.
	ExpectedTimeMillis int `mapstructure:"expected_millis"`
}

type Cookie struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

// New uses viper to get our server configurations
func New() (*Configuration, error) {
	var c Configuration
	if err := viper.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, c.validate()
}

//Allows for protocol relative URL if scheme is empty
func (cfg *Cache) GetBaseURL() string {
	cfg.Scheme = strings.ToLower(cfg.Scheme)
	if strings.Contains(cfg.Scheme, "https") {
		return fmt.Sprintf("https://%s", cfg.Host)
	}
	if strings.Contains(cfg.Scheme, "http") {
		return fmt.Sprintf("http://%s", cfg.Host)
	}
	return fmt.Sprintf("//%s", cfg.Host)
}

func (cfg *Configuration) GetCachedAssetURL(uuid string) string {
	return fmt.Sprintf("%s/cache?%s", cfg.CacheURL.GetBaseURL(), strings.Replace(cfg.CacheURL.Query, "%PBS_CACHE_UUID%", uuid, 1))
}
