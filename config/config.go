package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Configuration
type Configuration struct {
	ExternalURL string `mapstructure:"external_url"`
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	AdminPort   int    `mapstructure:"admin_port"`
	// StatusResponse is the string which will be returned by the /status endpoint when things are OK.
	// If empty, it will return a 204 with no content.
	StatusResponse       string             `mapstructure:"status_response"`
	DefaultTimeout       uint64             `mapstructure:"default_timeout_ms"`
	CacheURL             Cache              `mapstructure:"cache"`
	RecaptchaSecret      string             `mapstructure:"recaptcha_secret"`
	HostCookie           HostCookie         `mapstructure:"host_cookie"`
	Metrics              Metrics            `mapstructure:"metrics"`
	DataCache            DataCache          `mapstructure:"datacache"`
	StoredRequests       StoredRequests     `mapstructure:"stored_requests"`
	Adapters             map[string]Adapter `mapstructure:"adapters"`
	MaxRequestSize       int64              `mapstructure:"max_request_size"`
	Analytics            Analytics          `mapstructure:"analytics"`
	AMPTimeoutAdjustment int64              `mapstructure:"amp_timeout_adjustment_ms"`
	GDPR                 GDPR               `mapstructure:"gdpr"`
}

func (cfg *Configuration) validate() error {
	if cfg.MaxRequestSize < 0 {
		return fmt.Errorf("cfg.max_request_size must be a positive number. Got  %d", cfg.MaxRequestSize)
	}

	if err := cfg.StoredRequests.validate(); err != nil {
		return err
	}

	return cfg.GDPR.validate()
}

type GDPR struct {
	HostVendorID        int  `mapstructure:"host_vendor_id"`
	UsersyncIfAmbiguous bool `mapstructure:"usersync_if_ambiguous"`
}

func (cfg *GDPR) validate() error {
	if cfg.HostVendorID < 0 || cfg.HostVendorID > 0xffff {
		return fmt.Errorf("host_vendor_id must be in the range [0, %d]. Got %d", 0xffff, cfg.HostVendorID)
	}

	return nil
}

type Analytics struct {
	File FileLogs `mapstructure:"file"`
}

//Corresponding config for FileLogger as a PBS Analytics Module
type FileLogs struct {
	Filename string `mapstructure:"filename"`
}

type HostCookie struct {
	Domain       string `mapstructure:"domain"`
	Family       string `mapstructure:"family"`
	CookieName   string `mapstructure:"cookie_name"`
	OptOutURL    string `mapstructure:"opt_out_url"`
	OptInURL     string `mapstructure:"opt_in_url"`
	OptOutCookie Cookie `mapstructure:"optout_cookie"`
	// Cookie timeout in days
	TTL int64 `mapstructure:"ttl_days"`
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
	Influxdb InfluxMetrics `mapstructure:"influxdb"`
}

type InfluxMetrics struct {
	Host     string `mapstructure:"host"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

type DataCache struct {
	Type       string `mapstructure:"type"`
	Filename   string `mapstructure:"filename"`
	CacheSize  int    `mapstructure:"cache_size"`
	TTLSeconds int    `mapstructure:"ttl_seconds"`
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
func New(v *viper.Viper) (*Configuration, error) {
	var c Configuration
	if err := v.Unmarshal(&c); err != nil {
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
