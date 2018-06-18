package config

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
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
	AuctionTimeouts      AuctionTimeouts    `mapstructure:"auction_timeouts_ms"`
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

type configErrors []error

func (c configErrors) Error() string {
	if len(c) == 0 {
		return ""
	}
	buf := bytes.Buffer{}
	buf.WriteString("validation errors are:\n\n")
	for _, err := range c {
		buf.WriteString("  ")
		buf.WriteString(err.Error())
		buf.WriteString("\n")
	}
	buf.WriteString("\n")
	return buf.String()
}

func (cfg *Configuration) validate() configErrors {
	var errs configErrors
	errs = cfg.AuctionTimeouts.validate(errs)
	errs = cfg.StoredRequests.validate(errs)
	if cfg.MaxRequestSize < 0 {
		errs = append(errs, fmt.Errorf("cfg.max_request_size must be >= 0. Got %d", cfg.MaxRequestSize))
	}
	errs = cfg.GDPR.validate(errs)
	return errs
}

type AuctionTimeouts struct {
	// The default timeout is used if the user's request didn't define one. Use 0 if there's no default.
	Default uint64 `mapstructure:"default"`
	// The max timeout is used as an absolute cap, to prevent excessively long ones. Use 0 for no cap
	Max uint64 `mapstructure:"max"`
}

func (cfg *AuctionTimeouts) validate(errs configErrors) configErrors {
	if cfg.Max < cfg.Default {
		errs = append(errs, fmt.Errorf("auction_timeouts_ms.max cannot be less than auction_timeouts_ms.default. max=%d, default=%d", cfg.Max, cfg.Default))
	}
	return errs
}

// LimitAuctionTimeout returns the min of requested or cfg.MaxAuctionTimeout.
// Both values treat "0" as "infinite".
func (cfg *AuctionTimeouts) LimitAuctionTimeout(requested time.Duration) time.Duration {
	if requested == 0 && cfg.Default != 0 {
		return time.Duration(cfg.Default) * time.Millisecond
	}
	if cfg.Max > 0 {
		maxTimeout := time.Duration(cfg.Max) * time.Millisecond
		if requested == 0 || requested > maxTimeout {
			return maxTimeout
		}
	}
	return requested
}

type GDPR struct {
	HostVendorID        int          `mapstructure:"host_vendor_id"`
	UsersyncIfAmbiguous bool         `mapstructure:"usersync_if_ambiguous"`
	Timeouts            GDPRTimeouts `mapstructure:"timeouts_ms"`
}

func (cfg *GDPR) validate(errs configErrors) configErrors {
	if cfg.HostVendorID < 0 || cfg.HostVendorID > 0xffff {
		errs = append(errs, fmt.Errorf("gdpr.host_vendor_id must be in the range [0, %d]. Got %d", 0xffff, cfg.HostVendorID))
	}
	return errs
}

type GDPRTimeouts struct {
	InitVendorlistFetch   int `mapstructure:"init_vendorlist_fetches"`
	ActiveVendorlistFetch int `mapstructure:"active_vendorlist_fetch"`
}

func (t *GDPRTimeouts) InitTimeout() time.Duration {
	return time.Duration(t.InitVendorlistFetch) * time.Millisecond
}

func (t *GDPRTimeouts) ActiveTimeout() time.Duration {
	return time.Duration(t.ActiveVendorlistFetch) * time.Millisecond
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

// New uses viper to get our server configurations.
func New(v *viper.Viper) (*Configuration, error) {
	var c Configuration
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("viper failed to unmarshal app config: %v", err)
	}
	glog.Info("Logging the resolved configuration:")
	logGeneral(reflect.ValueOf(c), "  \t")
	if errs := c.validate(); len(errs) > 0 {
		return &c, errs
	}
	return &c, nil
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

// Set the default config values for the viper object we are using.
func SetupViper(v *viper.Viper) {
	v.SetConfigName("pbs")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/config")

	// Fixes #475: Some defaults will be set just so they are accessable via environment variables
	// (basically so viper knows they exist)
	v.SetDefault("external_url", "http://localhost:8000")
	v.SetDefault("host", "")
	v.SetDefault("port", 8000)
	v.SetDefault("admin_port", 6060)
	v.SetDefault("status_response", "")
	v.SetDefault("auction_timeouts_ms.default", 0)
	v.SetDefault("auction_timeouts_ms.max", 0)
	v.SetDefault("cache.scheme", "")
	v.SetDefault("cache.host", "")
	v.SetDefault("cache.query", "")
	v.SetDefault("cache.expected_millis", 10)
	v.SetDefault("recaptcha_secret", "")
	v.SetDefault("host_cookie.domain", "")
	v.SetDefault("host_cookie.family", "")
	v.SetDefault("host_cookie.cookie_name", "")
	v.SetDefault("host_cookie.opt_out_url", "")
	v.SetDefault("host_cookie.opt_in_url", "")
	v.SetDefault("host_cookie.optout_cookie.name", "")
	v.SetDefault("host_cookie.value", "")
	v.SetDefault("host_cookie.ttl_days", 90)
	// no metrics configured by default (metrics{host|database|username|password})
	v.SetDefault("metrics.influxdb.host", "")
	v.SetDefault("metrics.influxdb.database", "")
	v.SetDefault("metrics.influxdb.username", "")
	v.SetDefault("metrics.influxdb.password", "")
	v.SetDefault("datacache.type", "dummy")
	v.SetDefault("datacache.filename", "")
	v.SetDefault("datacache.cache_size", 0)
	v.SetDefault("datacache.ttl_seconds", 0)
	v.SetDefault("stored_requests.filesystem", false)
	v.SetDefault("stored_requests.postgres.connection.dbname", "")
	v.SetDefault("stored_requests.postgres.connection.host", "")
	v.SetDefault("stored_requests.postgres.connection.port", 0)
	v.SetDefault("stored_requests.postgres.connection.user", "")
	v.SetDefault("stored_requests.postgres.connection.password", "")
	v.SetDefault("stored_requests.postgres.fetcher.query", "")
	v.SetDefault("stored_requests.postgres.fetcher.amp_query", "")
	v.SetDefault("stored_requests.postgres.initialize_caches.timeout_ms", 0)
	v.SetDefault("stored_requests.postgres.initialize_caches.query", "")
	v.SetDefault("stored_requests.postgres.initialize_caches.amp_query", "")
	v.SetDefault("stored_requests.http.endpoint", "")
	v.SetDefault("stored_requests.http.amp_endpoint", "")
	v.SetDefault("stored_requests.in_memory_cache.type", "none")
	v.SetDefault("stored_requests.in_memory_cache.ttl_seconds", 0)
	v.SetDefault("stored_requests.in_memory_cache.request_cache_size_bytes", 0)
	v.SetDefault("stored_requests.in_memory_cache.imp_cache_size_bytes", 0)
	v.SetDefault("stored_requests.cache_events_api", false)
	v.SetDefault("stored_requests.http_events.endpoint", "")
	v.SetDefault("stored_requests.http_events.amp_endpoint", "")
	v.SetDefault("stored_requests.http_events.refresh_rate_seconds", 0)
	v.SetDefault("stored_requests.http_events.timeout_ms", 0)

	// This Appnexus endpoint works for most purposes. Docs can be found at https://wiki.appnexus.com/display/supply/Incoming+Bid+Request+from+SSPs
	v.SetDefault("adapters.appnexus.endpoint", "http://ib.adnxs.com/openrtb2")

	v.SetDefault("adapters.pubmatic.endpoint", "http://hbopenbid.pubmatic.com/translator?source=prebid-server")
	v.SetDefault("adapters.rubicon.endpoint", "http://exapi-us-east.rubiconproject.com/a/api/exchange.json")
	v.SetDefault("adapters.rubicon.usersync_url", "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid&gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}")
	v.SetDefault("adapters.eplanning.endpoint", "http://ads.us.e-planning.net/dsp/obr/1")
	v.SetDefault("adapters.eplanning.usersync_url", "http://sync.e-planning.net/um?uid")
	v.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	v.SetDefault("adapters.index.usersync_url", "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=https%3A%2F%2Fprebid.adnxs.com%2Fpbs%2Fv1%2Fsetuid%3Fbidder%3DindexExchange%26gdpr%3D{{gdpr}}%26gdpr_consent%3D{{gdpr_consent}}%26uid%3D")
	v.SetDefault("adapters.sovrn.endpoint", "http://ap.lijit.com/rtb/bid?src=prebid_server")
	v.SetDefault("adapters.sovrn.usersync_url", "//ap.lijit.com/pixel?")
	v.SetDefault("adapters.adform.endpoint", "http://adx.adform.net/adx")
	v.SetDefault("adapters.adform.usersync_url", "//cm.adform.net/cookie?redirect_url=")
	v.SetDefault("adapters.conversant.endpoint", "http://api.hb.ad.cpe.dotomi.com/s2s/header/24")
	v.SetDefault("adapters.conversant.usersync_url", "http://prebid-match.dotomi.com/prebid/match?rurl=")
	v.SetDefault("adapters.brightroll.endpoint", "http://east-bid.ybp.yahoo.com/bid/appnexuspbs")
	v.SetDefault("adapters.brightroll.usersync_url", "http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr={{gdpr}}&euconsent={{gdpr_consent}}&url=")

	v.SetDefault("max_request_size", 1024*256)
	v.SetDefault("analytics.file.filename", "")
	v.SetDefault("amp_timeout_adjustment_ms", 0)
	v.SetDefault("gdpr.host_vendor_id", 0)
	v.SetDefault("gdpr.usersync_if_ambiguous", false)
	v.SetDefault("gdpr.timeouts_ms.init_vendorlist_fetches", 0)
	v.SetDefault("gdpr.timeouts_ms.active_vendorlist_fetch", 0)

	// Set environment variable support:
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("PBS")
	v.AutomaticEnv()
	v.ReadInConfig()
}
