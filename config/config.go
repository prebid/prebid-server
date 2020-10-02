package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"text/template"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/macros"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/spf13/viper"

	validator "github.com/asaskevich/govalidator"
)

// Configuration specifies the static application config.
type Configuration struct {
	ExternalURL string     `mapstructure:"external_url"`
	Host        string     `mapstructure:"host"`
	Port        int        `mapstructure:"port"`
	Client      HTTPClient `mapstructure:"http_client"`
	CacheClient HTTPClient `mapstructure:"http_client_cache"`
	AdminPort   int        `mapstructure:"admin_port"`
	EnableGzip  bool       `mapstructure:"enable_gzip"`
	// StatusResponse is the string which will be returned by the /status endpoint when things are OK.
	// If empty, it will return a 204 with no content.
	StatusResponse    string          `mapstructure:"status_response"`
	AuctionTimeouts   AuctionTimeouts `mapstructure:"auction_timeouts_ms"`
	CacheURL          Cache           `mapstructure:"cache"`
	ExtCacheURL       ExternalCache   `mapstructure:"external_cache"`
	RecaptchaSecret   string          `mapstructure:"recaptcha_secret"`
	HostCookie        HostCookie      `mapstructure:"host_cookie"`
	Metrics           Metrics         `mapstructure:"metrics"`
	DataCache         DataCache       `mapstructure:"datacache"`
	StoredRequests    StoredRequests  `mapstructure:"stored_requests"`
	StoredRequestsAMP StoredRequests  `mapstructure:"stored_amp_req"`
	CategoryMapping   StoredRequests  `mapstructure:"category_mapping"`
	VTrack            VTrack          `mapstructure:"vtrack"`
	Event             Event           `mapstructure:"event"`
	Accounts          StoredRequests  `mapstructure:"accounts"`
	// Note that StoredVideo refers to stored video requests, and has nothing to do with caching video creatives.
	StoredVideo StoredRequests `mapstructure:"stored_video_req"`

	// Adapters should have a key for every openrtb_ext.BidderName, converted to lower-case.
	// Se also: https://github.com/spf13/viper/issues/371#issuecomment-335388559
	Adapters             map[string]Adapter `mapstructure:"adapters"`
	MaxRequestSize       int64              `mapstructure:"max_request_size"`
	Analytics            Analytics          `mapstructure:"analytics"`
	AMPTimeoutAdjustment int64              `mapstructure:"amp_timeout_adjustment_ms"`
	GDPR                 GDPR               `mapstructure:"gdpr"`
	CCPA                 CCPA               `mapstructure:"ccpa"`
	LMT                  LMT                `mapstructure:"lmt"`
	CurrencyConverter    CurrencyConverter  `mapstructure:"currency_converter"`
	DefReqConfig         DefReqConfig       `mapstructure:"default_request"`

	VideoStoredRequestRequired bool `mapstructure:"video_stored_request_required"`

	// Array of blacklisted apps that is used to create the hash table BlacklistedAppMap so App.ID's can be instantly accessed.
	BlacklistedApps   []string `mapstructure:"blacklisted_apps,flow"`
	BlacklistedAppMap map[string]bool
	// Array of blacklisted accounts that is used to create the hash table BlacklistedAcctMap so Account.ID's can be instantly accessed.
	BlacklistedAccts   []string `mapstructure:"blacklisted_accts,flow"`
	BlacklistedAcctMap map[string]bool
	// Is publisher/account ID required to be submitted in the OpenRTB2 request
	AccountRequired bool `mapstructure:"account_required"`
	// AccountDefaults defines default settings for valid accounts that are partially defined
	// and provides a way to set global settings that can be overridden at account level.
	AccountDefaults Account `mapstructure:"account_defaults"`
	// accountDefaultsJSON is the internal serialized form of AccountDefaults used for json merge
	accountDefaultsJSON json.RawMessage
	// Local private file containing SSL certificates
	PemCertsFile string `mapstructure:"certificates_file"`
	// Custom headers to handle request timeouts from queueing infrastructure
	RequestTimeoutHeaders RequestTimeoutHeaders `mapstructure:"request_timeout_headers"`
	// Debug/logging flags go here
	Debug Debug `mapstructure:"debug"`
	// RequestValidation specifies the request validation options.
	RequestValidation RequestValidation `mapstructure:"request_validation"`
	// When true, PBS will assign a randomly generated UUID to req.Source.TID if it is empty
	AutoGenSourceTID bool `mapstructure:"auto_gen_source_tid"`
}

const MIN_COOKIE_SIZE_BYTES = 500

type HTTPClient struct {
	MaxConnsPerHost     int `mapstructure:"max_connections_per_host"`
	MaxIdleConns        int `mapstructure:"max_idle_connections"`
	MaxIdleConnsPerHost int `mapstructure:"max_idle_connections_per_host"`
	IdleConnTimeout     int `mapstructure:"idle_connection_timeout_seconds"`
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
	errs = cfg.StoredRequestsAMP.validate(errs)
	errs = cfg.Accounts.validate(errs)
	errs = cfg.CategoryMapping.validate(errs)
	errs = cfg.StoredVideo.validate(errs)
	errs = cfg.Metrics.validate(errs)
	if cfg.MaxRequestSize < 0 {
		errs = append(errs, fmt.Errorf("cfg.max_request_size must be >= 0. Got %d", cfg.MaxRequestSize))
	}
	errs = cfg.GDPR.validate(errs)
	errs = cfg.CurrencyConverter.validate(errs)
	errs = validateAdapters(cfg.Adapters, errs)
	errs = cfg.Debug.validate(errs)
	errs = cfg.ExtCacheURL.validate(errs)
	if cfg.AccountDefaults.Disabled {
		glog.Warning(`With account_defaults.disabled=true, host-defined accounts must exist and have "disabled":false. All other requests will be rejected.`)
	}
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

func (data *ExternalCache) validate(errs configErrors) configErrors {
	if data.Host == "" && data.Path == "" {
		// Both host and path can be blank. No further validation needed
		return errs
	}

	if data.Scheme != "" && data.Scheme != "http" && data.Scheme != "https" {
		return append(errs, errors.New("External cache Scheme must be http or https if specified"))
	}

	// Either host or path or both not empty, validate.
	if data.Host == "" && data.Path != "" || data.Host != "" && data.Path == "" {
		return append(errs, errors.New("External cache Host and Path must both be specified"))
	}
	if strings.HasSuffix(data.Host, "/") {
		return append(errs, errors.New(fmt.Sprintf("External cache Host '%s' must not end with a path separator", data.Host)))
	}
	if strings.ContainsAny(data.Host, "://") {
		return append(errs, errors.New(fmt.Sprintf("External cache Host must not specify a protocol. '%s'", data.Host)))
	}
	if !strings.HasPrefix(data.Path, "/") {
		return append(errs, errors.New(fmt.Sprintf("External cache Path '%s' must begin with a path separator", data.Path)))
	}

	urlObj, err := url.Parse("https://" + data.Host + data.Path)
	if err != nil {
		return append(errs, errors.New(fmt.Sprintf("External cache Path validation error: %s ", err.Error())))
	}
	if urlObj.Host != data.Host {
		return append(errs, errors.New(fmt.Sprintf("External cache Host '%s' is invalid", data.Host)))
	}
	if urlObj.Path != data.Path {
		return append(errs, errors.New("External cache Path is invalid"))
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

// Privacy is a grouping of privacy related configs to assist in dependency injection.
type Privacy struct {
	CCPA CCPA
	GDPR GDPR
	LMT  LMT
}

type GDPR struct {
	HostVendorID            int          `mapstructure:"host_vendor_id"`
	UsersyncIfAmbiguous     bool         `mapstructure:"usersync_if_ambiguous"`
	Timeouts                GDPRTimeouts `mapstructure:"timeouts_ms"`
	NonStandardPublishers   []string     `mapstructure:"non_standard_publishers,flow"`
	NonStandardPublisherMap map[string]struct{}
	TCF1                    TCF1 `mapstructure:"tcf1"`
	TCF2                    TCF2 `mapstructure:"tcf2"`
	AMPException            bool `mapstructure:"amp_exception"`
	// EEACountries (EEA = European Economic Area) are a list of countries where we should assume GDPR applies.
	// If the gdpr flag is unset in a request, but geo.country is set, we will assume GDPR applies if and only
	// if the country matches one on this list. If both the GDPR flag and country are not set, we default
	// to UsersyncIfAmbiguous
	EEACountries    []string `mapstructure:"eea_countries"`
	EEACountriesMap map[string]struct{}
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

// TCF1 defines the TCF1 specific configurations for GDPR
type TCF1 struct {
	FetchGVL        bool   `mapstructure:"fetch_gvl"`
	FallbackGVLPath string `mapstructure:"fallback_gvl_path"`
}

// TCF2 defines the TCF2 specific configurations for GDPR
type TCF2 struct {
	Enabled             bool                 `mapstructure:"enabled"`
	Purpose1            PurposeDetail        `mapstructure:"purpose1"`
	Purpose2            PurposeDetail        `mapstructure:"purpose2"`
	Purpose7            PurposeDetail        `mapstructure:"purpose7"`
	SpecialPurpose1     PurposeDetail        `mapstructure:"special_purpose1"`
	PurposeOneTreatment PurposeOneTreatement `mapstructure:"purpose_one_treatement"`
}

// Making a purpose struct so purpose specific details can be added later.
type PurposeDetail struct {
	Enabled bool `mapstructure:"enabled"`
}

type PurposeOneTreatement struct {
	Enabled       bool `mapstructure:"enabled"`
	AccessAllowed bool `mapstructure:"access_allowed"`
}

type CCPA struct {
	Enforce bool `mapstructure:"enforce"`
}

type LMT struct {
	Enforce bool `mapstructure:"enforce"`
}

type Analytics struct {
	File     FileLogs `mapstructure:"file"`
	Pubstack Pubstack `mapstructure:"pubstack"`
}

type CurrencyConverter struct {
	FetchURL             string `mapstructure:"fetch_url"`
	FetchIntervalSeconds int    `mapstructure:"fetch_interval_seconds"`
	StaleRatesSeconds    int    `mapstructure:"stale_rates_seconds"`
}

func (cfg *CurrencyConverter) validate(errs configErrors) configErrors {
	if cfg.FetchIntervalSeconds < 0 {
		errs = append(errs, fmt.Errorf("currency_converter.fetch_interval_seconds must be in the range [0, %d]. Got %d", 0xffff, cfg.FetchIntervalSeconds))
	}
	return errs
}

// FileLogs Corresponding config for FileLogger as a PBS Analytics Module
type FileLogs struct {
	Filename string `mapstructure:"filename"`
}

type Pubstack struct {
	Enabled     bool           `mapstructure:"enabled"`
	ScopeId     string         `mapstructure:"scopeid"`
	IntakeUrl   string         `mapstructure:"endpoint"`
	Buffers     PubstackBuffer `mapstructure:"buffers"`
	ConfRefresh string         `mapstructure:"configuration_refresh_delay"`
}

type PubstackBuffer struct {
	BufferSize string `mapstructure:"size"`
	EventCount int    `mapstructure:"count"`
	Timeout    string `mapstructure:"timeout"`
}

type VTrack struct {
	TimeoutMS          int64 `mapstructure:"timeout_ms"`
	AllowUnknownBidder bool  `mapstructure:"allow_unknown_bidder"`
	Enabled            bool  `mapstructure:"enabled"`
}

type Event struct {
	TimeoutMS int64 `mapstructure:"timeout_ms"`
}

type HostCookie struct {
	Domain             string `mapstructure:"domain"`
	Family             string `mapstructure:"family"`
	CookieName         string `mapstructure:"cookie_name"`
	OptOutURL          string `mapstructure:"opt_out_url"`
	OptInURL           string `mapstructure:"opt_in_url"`
	MaxCookieSizeBytes int    `mapstructure:"max_cookie_size_bytes"`
	OptOutCookie       Cookie `mapstructure:"optout_cookie"`
	// Cookie timeout in days
	TTL int64 `mapstructure:"ttl_days"`
}

func (cfg *HostCookie) TTLDuration() time.Duration {
	return time.Duration(cfg.TTL) * time.Hour * 24
}

type RequestTimeoutHeaders struct {
	RequestTimeInQueue    string `mapstructure:"request_time_in_queue"`
	RequestTimeoutInQueue string `mapstructure:"request_timeout_in_queue"`
}

const (
	dummyHost        string = "dummyhost.com"
	dummyPublisherID string = "12"
	dummyAccountID   string = "some_account"
	dummyGDPR        string = "0"
	dummyGDPRConsent string = "someGDPRConsentString"
	dummyCCPA        string = "1NYN"
)

type Adapter struct {
	Endpoint string `mapstructure:"endpoint"` // Required
	// UserSyncURL is the URL returned by /cookie_sync for this Bidder. It is _usually_ optional.
	// If not defined, sensible defaults will be derived based on the config.external_url.
	// Note that some Bidders don't have sensible defaults, because their APIs require an ID that will vary
	// from one PBS host to another.
	//
	// For these bidders, there will be a warning logged on startup that usersyncs will not work if you have not
	// defined one in the app config. Check your app logs for more info.
	//
	// This value will be interpreted as a Golang Template. At runtime, the following Template variables will be replaced.
	//
	//   {{.GDPR}}      -- This will be replaced with the "gdpr" property sent to /cookie_sync
	//   {{.Consent}}   -- This will be replaced with the "consent" property sent to /cookie_sync
	//   {{.USPrivacy}} -- This will be replaced with the "us_privacy" property sent to /cookie_sync
	//
	// For more info on templates, see: https://golang.org/pkg/text/template/
	UserSyncURL string `mapstructure:"usersync_url"`
	XAPI        struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Tracker  string `mapstructure:"tracker"`
	} `mapstructure:"xapi"` // needed for Rubicon
	Disabled         bool   `mapstructure:"disabled"`
	ExtraAdapterInfo string `mapstructure:"extra_info"`

	// needed for Facebook
	PlatformID string `mapstructure:"platform_id"`
	AppSecret  string `mapstructure:"app_secret"`
}

// validateAdapterEndpoint makes sure that an adapter has a valid endpoint
// associated with it
func validateAdapterEndpoint(endpoint string, adapterName string, errs configErrors) configErrors {
	if endpoint == "" {
		return append(errs, fmt.Errorf("There's no default endpoint available for %s. Calls to this bidder/exchange will fail. "+
			"Please set adapters.%s.endpoint in your app config", adapterName, adapterName))
	}

	// Create endpoint template
	endpointTemplate, err := template.New("endpointTemplate").Parse(endpoint)
	if err != nil {
		return append(errs, fmt.Errorf("Invalid endpoint template: %s for adapter: %s. %v", endpoint, adapterName, err))
	}
	// Resolve macros (if any) in the endpoint URL
	resolvedEndpoint, err := macros.ResolveMacros(*endpointTemplate, macros.EndpointTemplateParams{Host: dummyHost, PublisherID: dummyPublisherID, AccountID: dummyAccountID})
	if err != nil {
		return append(errs, fmt.Errorf("Unable to resolve endpoint: %s for adapter: %s. %v", endpoint, adapterName, err))
	}
	// Validate the resolved endpoint
	//
	// Validating using both IsURL and IsRequestURL because IsURL allows relative paths
	// whereas IsRequestURL requires absolute path but fails to check other valid URL
	// format constraints.
	//
	// For example: IsURL will allow "abcd.com" but IsRequestURL won't
	// IsRequestURL will allow "http://http://abcd.com" but IsURL won't
	if !validator.IsURL(resolvedEndpoint) || !validator.IsRequestURL(resolvedEndpoint) {
		errs = append(errs, fmt.Errorf("The endpoint: %s for %s is not a valid URL", resolvedEndpoint, adapterName))
	}
	return errs
}

// validateAdapterUserSyncURL validates an adapter's user sync URL if it is set
func validateAdapterUserSyncURL(userSyncURL string, adapterName string, errs configErrors) configErrors {
	if userSyncURL != "" {
		// Create user_sync URL template
		userSyncTemplate, err := template.New("userSyncTemplate").Parse(userSyncURL)
		if err != nil {
			return append(errs, fmt.Errorf("Invalid user sync URL template: %s for adapter: %s. %v", userSyncURL, adapterName, err))
		}
		// Resolve macros (if any) in the user_sync URL
		dummyMacroValues := macros.UserSyncTemplateParams{
			GDPR:        dummyGDPR,
			GDPRConsent: dummyGDPRConsent,
			USPrivacy:   dummyCCPA,
		}
		resolvedUserSyncURL, err := macros.ResolveMacros(*userSyncTemplate, dummyMacroValues)
		if err != nil {
			return append(errs, fmt.Errorf("Unable to resolve user sync URL: %s for adapter: %s. %v", userSyncURL, adapterName, err))
		}
		// Validate the resolved user sync URL
		//
		// Validating using both IsURL and IsRequestURL because IsURL allows relative paths
		// whereas IsRequestURL requires absolute path but fails to check other valid URL
		// format constraints.
		//
		// For example: IsURL will allow "abcd.com" but IsRequestURL won't
		// IsRequestURL will allow "http://http://abcd.com" but IsURL won't
		if !validator.IsURL(resolvedUserSyncURL) || !validator.IsRequestURL(resolvedUserSyncURL) {
			errs = append(errs, fmt.Errorf("The user_sync URL: %s for %s is invalid", resolvedUserSyncURL, adapterName))
		}
	}
	return errs
}

// validateAdapters validates adapter's endpoint and user sync URL
func validateAdapters(adapterMap map[string]Adapter, errs configErrors) configErrors {
	for adapterName, adapter := range adapterMap {
		if !adapter.Disabled {
			// Verify that every adapter has a valid endpoint associated with it
			errs = validateAdapterEndpoint(adapter.Endpoint, adapterName, errs)

			// Verify that valid user_sync URLs are specified in the config
			errs = validateAdapterUserSyncURL(adapter.UserSyncURL, adapterName, errs)
		}
	}
	return errs
}

type Metrics struct {
	Influxdb   InfluxMetrics     `mapstructure:"influxdb"`
	Prometheus PrometheusMetrics `mapstructure:"prometheus"`
	Disabled   DisabledMetrics   `mapstructure:"disabled_metrics"`
}

type DisabledMetrics struct {
	// True if we want to stop collecting account-to-adapter metrics
	AccountAdapterDetails bool `mapstructure:"account_adapter_details"`

	// True if we don't want to collect metrics about the connections prebid
	// server establishes with bidder servers such as the number of connections
	// that were created or reused.
	AdapterConnectionMetrics bool `mapstructure:"adapter_connections_metrics"`
}

func (cfg *Metrics) validate(errs configErrors) configErrors {
	return cfg.Prometheus.validate(errs)
}

type InfluxMetrics struct {
	Host               string `mapstructure:"host"`
	Database           string `mapstructure:"database"`
	Username           string `mapstructure:"username"`
	Password           string `mapstructure:"password"`
	MetricSendInterval int    `mapstructure:"metric_send_interval"`
}

type PrometheusMetrics struct {
	Port             int    `mapstructure:"port"`
	Namespace        string `mapstructure:"namespace"`
	Subsystem        string `mapstructure:"subsystem"`
	TimeoutMillisRaw int    `mapstructure:"timeout_ms"`
}

func (cfg *PrometheusMetrics) validate(errs configErrors) configErrors {
	if cfg.Port > 0 && cfg.TimeoutMillisRaw <= 0 {
		errs = append(errs, fmt.Errorf("metrics.prometheus.timeout_ms must be positive if metrics.prometheus.port is defined. Got timeout=%d and port=%d", cfg.TimeoutMillisRaw, cfg.Port))
	}
	return errs
}

func (m *PrometheusMetrics) Timeout() time.Duration {
	return time.Duration(m.TimeoutMillisRaw) * time.Millisecond
}

type DataCache struct {
	Type       string `mapstructure:"type"`
	Filename   string `mapstructure:"filename"`
	CacheSize  int    `mapstructure:"cache_size"`
	TTLSeconds int    `mapstructure:"ttl_seconds"`
}

// ExternalCache configures the externally accessible cache url.
type ExternalCache struct {
	Scheme string `mapstructure:"scheme"`
	Host   string `mapstructure:"host"`
	Path   string `mapstructure:"path"`
}

// Cache configures the url used internally by Prebid Server to communicate with Prebid Cache.
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

	DefaultTTLs DefaultTTLs `mapstructure:"default_ttl_seconds"`
}

// Default TTLs to use to cache bids for different types of imps.
type DefaultTTLs struct {
	Banner int `mapstructure:"banner"`
	Video  int `mapstructure:"video"`
	Native int `mapstructure:"native"`
	Audio  int `mapstructure:"audio"`
}

type Cookie struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

// AliasConfig will define the various source(s) or the default aliases
// Currently only filesystem is supported, but keeping the config structure
type DefReqConfig struct {
	Type       string      `mapstructure:"type"`
	FileSystem DefReqFiles `mapstructure:"file"`
	AliasInfo  bool        `mapstructure:"alias_info"`
}

type DefReqFiles struct {
	FileName string `mapstructure:"name"`
}

type Debug struct {
	TimeoutNotification TimeoutNotification `mapstructure:"timeout_notification"`
}

func (cfg *Debug) validate(errs configErrors) configErrors {
	return cfg.TimeoutNotification.validate(errs)
}

type TimeoutNotification struct {
	// Log timeout notifications in the application log
	Log bool `mapstructure:"log"`
	// Fraction of notifications to log
	SamplingRate float32 `mapstructure:"sampling_rate"`
	// Only log failures
	FailOnly bool `mapstructure:"fail_only"`
}

func (cfg *TimeoutNotification) validate(errs configErrors) configErrors {
	if cfg.SamplingRate < 0.0 || cfg.SamplingRate > 1.0 {
		errs = append(errs, fmt.Errorf("debug.timeout_notification.sampling_rate must be positive and not greater than 1.0. Got %f", cfg.SamplingRate))
	}
	return errs
}

// New uses viper to get our server configurations.
func New(v *viper.Viper) (*Configuration, error) {
	var c Configuration
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("viper failed to unmarshal app config: %v", err)
	}
	c.setDerivedDefaults()

	if err := c.RequestValidation.Parse(); err != nil {
		return nil, err
	}

	if err := isValidCookieSize(c.HostCookie.MaxCookieSizeBytes); err != nil {
		glog.Fatal(fmt.Printf("Max cookie size %d cannot be less than %d \n", c.HostCookie.MaxCookieSizeBytes, MIN_COOKIE_SIZE_BYTES))
		return nil, err
	}

	// Update account defaults and generate base json for patch
	c.AccountDefaults.CacheTTL = c.CacheURL.DefaultTTLs // comment this out to set explicitly in config
	if err := c.MarshalAccountDefaults(); err != nil {
		return nil, err
	}

	// To look for a request's publisher_id in the NonStandardPublishers list in
	// O(1) time, we fill this hash table located in the NonStandardPublisherMap field of GDPR
	var s struct{}
	c.GDPR.NonStandardPublisherMap = make(map[string]struct{})
	for i := 0; i < len(c.GDPR.NonStandardPublishers); i++ {
		c.GDPR.NonStandardPublisherMap[c.GDPR.NonStandardPublishers[i]] = s
	}

	c.GDPR.EEACountriesMap = make(map[string]struct{})
	for i := 0; i < len(c.GDPR.EEACountriesMap); i++ {
		c.GDPR.NonStandardPublisherMap[c.GDPR.EEACountries[i]] = s
	}

	// To look for a request's app_id in O(1) time, we fill this hash table located in the
	// the BlacklistedApps field of the Configuration struct defined in this file
	c.BlacklistedAppMap = make(map[string]bool)
	for i := 0; i < len(c.BlacklistedApps); i++ {
		c.BlacklistedAppMap[c.BlacklistedApps[i]] = true
	}

	// To look for a request's account id in O(1) time, we fill this hash table located in the
	// the BlacklistedAccts field of the Configuration struct defined in this file
	c.BlacklistedAcctMap = make(map[string]bool)
	for i := 0; i < len(c.BlacklistedAccts); i++ {
		c.BlacklistedAcctMap[c.BlacklistedAccts[i]] = true
	}

	// Migrate combo stored request config to separate stored_reqs and amp stored_reqs configs.
	resolvedStoredRequestsConfig(&c)

	glog.Info("Logging the resolved configuration:")
	logGeneral(reflect.ValueOf(c), "  \t")
	if errs := c.validate(); len(errs) > 0 {
		return &c, errs
	}

	return &c, nil
}

// MarshalAccountDefaults compiles AccountDefaults into the JSON format used for merge patch
func (cfg *Configuration) MarshalAccountDefaults() error {
	var err error
	if cfg.accountDefaultsJSON, err = json.Marshal(cfg.AccountDefaults); err != nil {
		glog.Warningf("converting %+v to json: %v", cfg.AccountDefaults, err)
	}
	return err
}

// AccountDefaultsJSON returns the precompiled JSON form of account_defaults
func (cfg *Configuration) AccountDefaultsJSON() json.RawMessage {
	return cfg.accountDefaultsJSON
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

// Initialize any default config values which have sensible defaults, but those defaults depend on other config values.
//
// For example, the typical Bidder's usersync URL includes the PBS config.external_url, because it redirects to the `external_url/setuid` endpoint.
//
func (cfg *Configuration) setDerivedDefaults() {
	externalURL := cfg.ExternalURL
	setDefaultUsersync(cfg.Adapters, openrtb_ext.Bidder33Across, "https://ic.tynt.com/r/d?m=xch&rt=html&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&ru="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3D33across%26uid%3D33XUSERID33X&id=zzz000000000002zzz")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdform, "https://cm.adform.net/cookie?redirect_url="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadform%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	// openrtb_ext.BidderAdgeneration doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdkernel, "https://sync.adkernel.com/user-sync?t=image&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadkernel%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdkernelAdn, "https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3DadkernelAdn%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdpone, "https://usersync.adpone.com/csync?redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadpone%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdtarget, "https://sync.console.adtarget.com.tr/csync?t=p&ep=0&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadtarget%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdtelligent, "https://sync.adtelligent.com/csync?t=p&ep=0&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdmixer, "https://inv-nets.admixer.net/adxcm.aspx?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir=1&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadmixer%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%24visitor_cookie%24%24")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdman, "https://sync.admanmedia.com/pbs.gif?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadman%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5BUID%5D")
	// openrtb_ext.BidderAdOcean doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdvangelists, "https://nep.advangelists.com/xp/user-sync?acctid={aid}&&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadvangelists%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAJA, "https://ad.as.amanad.adtdp.com/v1/sync/ssp?ssp=4&gdpr={{.GDPR}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Daja%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%25s")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAppnexus, "https://ib.adnxs.com/getuid?"+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAvocet, "https://ads.avct.cloud/getuid?&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&url="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Davocet%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7B%7BUUID%7D%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderBeachfront, "https://sync.bfmio.com/sync_s2s?gdpr={{.GDPR}}&us_privacy={{.USPrivacy}}&url="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dbeachfront%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bio_cid%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderBeintoo, "https://ib.beintoo.com/um?ssp=pbs&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dbeintoo%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderBrightroll, "https://pr-bh.ybp.yahoo.com/sync/appnexusprebidserver/?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&url="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderColossus, "https://sync.colossusssp.com/pbs.gif?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dcolossus%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5BUID%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderConnectAd, "https://cdn.connectad.io/connectmyusers.php?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&cb="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dconnectad%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderConsumable, "https://e.serverbid.com/udb/9969/match?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderConversant, "https://prebid-match.dotomi.com/match/bounce/current?version=1&networkId=72582&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderCpmstar, "https://server.cpmstar.com/usersync.aspx?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dcpmstar%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderDatablocks, "https://sync.v5prebid.datablocks.net/s2ssync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Ddatablocks%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7Buid%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderDmx, "https://dmx.districtm.io/s/v1/img/s/10007?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Ddatablocks%26gdpr%3D%24%7Bgdpr%7D%26gdpr_consent%3D%24%7Bgdpr_consent%7D%26uid%3D%24%7Buid%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderEmxDigital, "https://cs.emxdgt.com/um?ssp=pbs&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Demx_digital%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderEngageBDR, "https://match.bnmla.com/usersync/s2s_sync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dengagebdr%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderEPlanning, "https://ads.us.e-planning.net/uspd/1/?du="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Deplanning%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	// openrtb_ext.BidderFacebook doesn't have a good default.
	// openrtb_ext.BidderGamma doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderGamoshi, "https://rtb.gamoshi.io/user_sync_prebid?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dgamoshi%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bgusr%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderGrid, "https://x.bidswitch.net/check_uuid/"+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dgrid%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BBSW_UUID%7D?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderGumGum, "https://rtb.gumgum.com/usync/prbds2s?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dgumgum%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderImprovedigital, "https://ad.360yield.com/server_match?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dimprovedigital%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7BPUB_USER_ID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderIx, "https://ssum.casalemedia.com/usermatchredir?s=184932&cb="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dix%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderInvibes, "https://u2.videostep.com/User/getLid?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dinvibes%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderKrushmedia, "https://cs.krushmedia.com/4e4abdd5ecc661643458a730b1aa927d.gif?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dkrushmedia%26uid%3D%5BUID%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderLifestreet, "https://ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%24visitor_cookie%24%24")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderLockerDome, "https://lockerdome.com/usync/prebidserver?pid="+cfg.Adapters["lockerdome"].PlatformID+"&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dlockerdome%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7B%7Buid%7D%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderLogicad, "https://cr-p31.ladsp.jp/cookiesender/31?r=true&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&ru="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dlogicad%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderLunaMedia, "https://api.lunamedia.io/xp/user-sync?redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dlunamedia%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderMarsmedia, "https://dmp.rtbsrv.com/dmp/profiles/cm?p_id=179&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dmarsmedia%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderMgid, "https://cm.mgid.com/m?cdsp=363893&adu="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dmgid%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Bmuidn%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderNanoInteractive, "https://ad.audiencemanager.de/hbs/cookie_sync?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dnanointeractive%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderNinthDecimal, "https://rtb.ninthdecimal.com/xp/user-sync?acctid={aid}&&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dninthdecimal%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderOpenx, "https://rtb.openx.net/sync/prebid?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderPubmatic, "https://ads.pubmatic.com/AdServer/js/user_sync.html?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&predirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderPulsepoint, "https://bh.contextweb.com/rtset?pid=561205&ev=1&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%25%25VGUID%25%25")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderRhythmone, "https://sync.1rx.io/usersync2/rmphb?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5BRX_UUID%5D")
	// openrtb_ext.BidderRTBHouse doesn't have a good default.
	// openrtb_ext.BidderRubicon doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSharethrough, "https://match.sharethrough.com/FGMrCMMc/v1?redirectUri="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsharethrough%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSmartadserver, "https://ssbsync.smartadserver.com/api/sync?callerId=5&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsmartadserver%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bssb_sync_pid%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSmartRTB, "https://market-global.smrtb.com/sync/all?nid=smartrtb&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&rr="+url.QueryEscape(externalURL)+"%252Fsetuid%253Fbidder%253Dsmartrtb%2526gdpr%253D{{.GDPR}}%2526gdpr_consent%253D{{.GDPRConsent}}%2526uid%253D%257BXID%257D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSmartyAds, "https://as.ck-ie.com/prebid.gif?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsmartyads%26uid%3D%5BUID%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSomoaudience, "https://publisher-east.mobileadtrading.com/usersync?ru="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSonobi, "https://sync.go.sonobi.com/us.gif?loc="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsonobi%26consent_string%3D{{.GDPR}}%26gdpr%3D{{.GDPRConsent}}%26uid%3D%5BUID%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSovrn, "https://ap.lijit.com/pixel?redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSynacormedia, "https://sync.technoratimedia.com/services?srv=cs&pid=70&cb="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsynacormedia%26uid%3D%5BUSER_ID%5D")
	// openrtb_ext.BidderTappx doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderTelaria, "https://pbs.publishers.tremorhub.com/pubsync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dtelaria%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Btvid%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderTriplelift, "https://eb2.3lift.com/getuid?gdpr={{.GDPR}}&cmp_cs={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dtriplelift%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderTripleliftNative, "https://eb2.3lift.com/getuid?gdpr={{.GDPR}}&cmp_cs={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dtriplelift_native%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderUcfunnel, "https://sync.aralego.com/idsync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&usprivacy={{.USPrivacy}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Ducfunnel%26uid%3DSspCookieUserId")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderUnruly, "https://usermatch.targeting.unrulymedia.com/pbsync?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dunruly%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderValueImpression, "https://rtb.valueimpression.com/usersync?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dvalueimpression%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderVisx, "https://t.visx.net/s2s_sync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dvisx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUUID%7D")
	// openrtb_ext.BidderVrtcal doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderYieldlab, "https://ad.yieldlab.net/mr?t=2&pid=9140838&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dyieldlab%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%25%25YL_UID%25%25")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderYieldmo, "https://ads.yieldmo.com/pbsync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dyieldmo%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderYieldone, "https://y.one.impact-ad.jp/hbs_cs?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redirectUri="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dyieldone%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderZeroClickFraud, "https://s.0cf.io/sync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dzeroclickfraud%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7Buid%7D")
}

func setDefaultUsersync(m map[string]Adapter, bidder openrtb_ext.BidderName, defaultValue string) {
	lowercased := strings.ToLower(string(bidder))
	if m[lowercased].UserSyncURL == "" {
		// Go doesnt let us edit the properties of a value inside a map directly.
		editable := m[lowercased]
		editable.UserSyncURL = defaultValue
		m[lowercased] = editable
	}
}

// Set the default config values for the viper object we are using.
func SetupViper(v *viper.Viper, filename string) {
	if filename != "" {
		v.SetConfigName(filename)
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/config")
	}

	// Fixes #475: Some defaults will be set just so they are accessible via environment variables
	// (basically so viper knows they exist)
	v.SetDefault("external_url", "http://localhost:8000")
	v.SetDefault("host", "")
	v.SetDefault("port", 8000)
	v.SetDefault("admin_port", 6060)
	v.SetDefault("enable_gzip", false)
	v.SetDefault("status_response", "")
	v.SetDefault("auction_timeouts_ms.default", 0)
	v.SetDefault("auction_timeouts_ms.max", 0)
	v.SetDefault("cache.scheme", "")
	v.SetDefault("cache.host", "")
	v.SetDefault("cache.query", "")
	v.SetDefault("cache.expected_millis", 10)
	v.SetDefault("cache.default_ttl_seconds.banner", 0)
	v.SetDefault("cache.default_ttl_seconds.video", 0)
	v.SetDefault("cache.default_ttl_seconds.native", 0)
	v.SetDefault("cache.default_ttl_seconds.audio", 0)
	v.SetDefault("external_cache.scheme", "")
	v.SetDefault("external_cache.host", "")
	v.SetDefault("external_cache.path", "")
	v.SetDefault("recaptcha_secret", "")
	v.SetDefault("host_cookie.domain", "")
	v.SetDefault("host_cookie.family", "")
	v.SetDefault("host_cookie.cookie_name", "")
	v.SetDefault("host_cookie.opt_out_url", "")
	v.SetDefault("host_cookie.opt_in_url", "")
	v.SetDefault("host_cookie.optout_cookie.name", "")
	v.SetDefault("host_cookie.value", "")
	v.SetDefault("host_cookie.ttl_days", 90)
	v.SetDefault("host_cookie.max_cookie_size_bytes", 0)
	v.SetDefault("http_client.max_connections_per_host", 0) // unlimited
	v.SetDefault("http_client.max_idle_connections", 400)
	v.SetDefault("http_client.max_idle_connections_per_host", 10)
	v.SetDefault("http_client.idle_connection_timeout_seconds", 60)
	v.SetDefault("http_client_cache.max_connections_per_host", 0) // unlimited
	v.SetDefault("http_client_cache.max_idle_connections", 10)
	v.SetDefault("http_client_cache.max_idle_connections_per_host", 2)
	v.SetDefault("http_client_cache.idle_connection_timeout_seconds", 60)
	// no metrics configured by default (metrics{host|database|username|password})
	v.SetDefault("metrics.disabled_metrics.account_adapter_details", false)
	v.SetDefault("metrics.disabled_metrics.adapter_connections_metrics", true)
	v.SetDefault("metrics.influxdb.host", "")
	v.SetDefault("metrics.influxdb.database", "")
	v.SetDefault("metrics.influxdb.username", "")
	v.SetDefault("metrics.influxdb.password", "")
	v.SetDefault("metrics.influxdb.metric_send_interval", 20)
	v.SetDefault("metrics.prometheus.port", 0)
	v.SetDefault("metrics.prometheus.namespace", "")
	v.SetDefault("metrics.prometheus.subsystem", "")
	v.SetDefault("metrics.prometheus.timeout_ms", 10000)
	v.SetDefault("datacache.type", "dummy")
	v.SetDefault("datacache.filename", "")
	v.SetDefault("datacache.cache_size", 0)
	v.SetDefault("datacache.ttl_seconds", 0)
	v.SetDefault("category_mapping.filesystem.enabled", true)
	v.SetDefault("category_mapping.filesystem.directorypath", "./static/category-mapping")
	v.SetDefault("category_mapping.http.endpoint", "")
	v.SetDefault("stored_requests.filesystem.enabled", false)
	v.SetDefault("stored_requests.filesystem.directorypath", "./stored_requests/data/by_id")
	v.SetDefault("stored_requests.directorypath", "./stored_requests/data/by_id")
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
	v.SetDefault("stored_requests.postgres.poll_for_updates.refresh_rate_seconds", 0)
	v.SetDefault("stored_requests.postgres.poll_for_updates.timeout_ms", 0)
	v.SetDefault("stored_requests.postgres.poll_for_updates.query", "")
	v.SetDefault("stored_requests.postgres.poll_for_updates.amp_query", "")
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
	// stored_video is short for stored_video_requests.
	// PBS is not in the business of storing video content beyond the normal prebid cache system.
	v.SetDefault("stored_video_req.filesystem.enabled", false)
	v.SetDefault("stored_video_req.filesystem.directorypath", "")
	v.SetDefault("stored_video_req.postgres.connection.dbname", "")
	v.SetDefault("stored_video_req.postgres.connection.host", "")
	v.SetDefault("stored_video_req.postgres.connection.port", 0)
	v.SetDefault("stored_video_req.postgres.connection.user", "")
	v.SetDefault("stored_video_req.postgres.connection.password", "")
	v.SetDefault("stored_video_req.postgres.fetcher.query", "")
	v.SetDefault("stored_video_req.postgres.initialize_caches.timeout_ms", 0)
	v.SetDefault("stored_video_req.postgres.initialize_caches.query", "")
	v.SetDefault("stored_video_req.postgres.poll_for_updates.refresh_rate_seconds", 0)
	v.SetDefault("stored_video_req.postgres.poll_for_updates.timeout_ms", 0)
	v.SetDefault("stored_video_req.postgres.poll_for_updates.query", "")
	v.SetDefault("stored_video_req.http.endpoint", "")
	v.SetDefault("stored_video_req.in_memory_cache.type", "none")
	v.SetDefault("stored_video_req.in_memory_cache.ttl_seconds", 0)
	v.SetDefault("stored_video_req.in_memory_cache.request_cache_size_bytes", 0)
	v.SetDefault("stored_video_req.in_memory_cache.imp_cache_size_bytes", 0)
	v.SetDefault("stored_video_req.cache_events.enabled", false)
	v.SetDefault("stored_video_req.cache_events.endpoint", "")
	v.SetDefault("stored_video_req.http_events.endpoint", "")
	v.SetDefault("stored_video_req.http_events.refresh_rate_seconds", 0)
	v.SetDefault("stored_video_req.http_events.timeout_ms", 0)

	v.SetDefault("vtrack.timeout_ms", 2000)
	v.SetDefault("vtrack.allow_unknown_bidder", true)
	v.SetDefault("vtrack.enabled", true)

	v.SetDefault("event.timeout_ms", 1000)

	v.SetDefault("accounts.filesystem.enabled", false)
	v.SetDefault("accounts.filesystem.directorypath", "./stored_requests/data/by_id")
	v.SetDefault("accounts.in_memory_cache.type", "none")

	for _, bidder := range openrtb_ext.BidderMap {
		setBidderDefaults(v, strings.ToLower(string(bidder)))
	}

	// Disabling adapters by default that require some specific config params.
	// If you're using one of these, make sure you check out the documentation (https://github.com/prebid/prebid-server/tree/master/docs/bidders)
	// for them and specify all the parameters they need for them to work correctly.
	v.SetDefault("adapters.33across.endpoint", "http://ssc.33across.com/api/v1/hb")
	v.SetDefault("adapters.33across.partner_id", "")
	v.SetDefault("adapters.adform.endpoint", "http://adx.adform.net/adx")
	v.SetDefault("adapters.adgeneration.endpoint", "https://d.socdm.com/adsv/v1")
	v.SetDefault("adapters.adhese.endpoint", "https://ads-{{.AccountID}}.adhese.com/json")
	v.SetDefault("adapters.adkernel.endpoint", "http://{{.Host}}/hb?zone={{.ZoneID}}")
	v.SetDefault("adapters.adkerneladn.endpoint", "http://{{.Host}}/rtbpub?account={{.PublisherID}}")
	v.SetDefault("adapters.adman.endpoint", "http://pub.admanmedia.com/?c=o&m=ortb")
	v.SetDefault("adapters.admixer.endpoint", "http://inv-nets.admixer.net/pbs.aspx")
	v.SetDefault("adapters.adocean.endpoint", "https://{{.Host}}")
	v.SetDefault("adapters.adoppler.endpoint", "http://app.trustedmarketplace.io/ads")
	v.SetDefault("adapters.adpone.endpoint", "http://rtb.adpone.com/bid-request?src=prebid_server")
	v.SetDefault("adapters.adprime.endpoint", "http://delta.adprime.com/?c=o&m=ortb")
	v.SetDefault("adapters.adtarget.endpoint", "http://ghb.console.adtarget.com.tr/pbs/ortb")
	v.SetDefault("adapters.adtelligent.endpoint", "http://ghb.adtelligent.com/pbs/ortb")
	v.SetDefault("adapters.advangelists.endpoint", "http://nep.advangelists.com/xp/get?pubid={{.PublisherID}}")
	v.SetDefault("adapters.aja.endpoint", "https://ad.as.amanad.adtdp.com/v1/bid/4")
	v.SetDefault("adapters.applogy.endpoint", "http://rtb.applogy.com/v1/prebid")
	v.SetDefault("adapters.appnexus.endpoint", "http://ib.adnxs.com/openrtb2") // Docs: https://wiki.appnexus.com/display/supply/Incoming+Bid+Request+from+SSPs
	v.SetDefault("adapters.appnexus.platform_id", "5")
	v.SetDefault("adapters.audiencenetwork.disabled", true)
	v.SetDefault("adapters.avocet.disabled", true)
	v.SetDefault("adapters.beachfront.endpoint", "https://display.bfmio.com/prebid_display")
	v.SetDefault("adapters.beachfront.extra_info", "{\"video_endpoint\":\"https://reachms.bfmio.com/bid.json?exchange_id\"}")
	v.SetDefault("adapters.beintoo.endpoint", "https://ib.beintoo.com/um")
	v.SetDefault("adapters.between.endpoint", "http://{{.Host}}.betweendigital.com")
	v.SetDefault("adapters.brightroll.endpoint", "http://east-bid.ybp.yahoo.com/bid/appnexuspbs")
	v.SetDefault("adapters.colossus.endpoint", "http://colossusssp.com/?c=o&m=rtb")
	v.SetDefault("adapters.connectad.endpoint", "http://bidder.connectad.io/API?src=pbs")
	v.SetDefault("adapters.consumable.endpoint", "https://e.serverbid.com/api/v2")
	v.SetDefault("adapters.conversant.endpoint", "http://api.hb.ad.cpe.dotomi.com/cvx/server/hb/ortb/25")
	v.SetDefault("adapters.cpmstar.endpoint", "https://server.cpmstar.com/openrtbbidrq.aspx")
	v.SetDefault("adapters.datablocks.endpoint", "http://{{.Host}}/openrtb2?sid={{.SourceId}}")
	v.SetDefault("adapters.dmx.endpoint", "https://dmx-direct.districtm.io/b/v2")
	v.SetDefault("adapters.emx_digital.endpoint", "https://hb.emxdgt.com")
	v.SetDefault("adapters.engagebdr.endpoint", "http://dsp.bnmla.com/hb")
	v.SetDefault("adapters.eplanning.endpoint", "http://rtb.e-planning.net/pbs/1")
	v.SetDefault("adapters.gamma.endpoint", "https://hb.gammaplatform.com/adx/request/")
	v.SetDefault("adapters.gamoshi.endpoint", "https://rtb.gamoshi.io")
	v.SetDefault("adapters.grid.endpoint", "http://grid.bidswitch.net/sp_bid?sp=prebid")
	v.SetDefault("adapters.gumgum.endpoint", "https://g2.gumgum.com/providers/prbds2s/bid")
	v.SetDefault("adapters.improvedigital.endpoint", "http://ad.360yield.com/pbs")
	v.SetDefault("adapters.inmobi.endpoint", "https://api.w.inmobi.com/showad/openrtb/bidder/prebid")
	v.SetDefault("adapters.ix.endpoint", "http://appnexus-us-east.lb.indexww.com/transbidder?p=184932")
	v.SetDefault("adapters.krushmedia.endpoint", "http://ads4.krushmedia.com/?c=rtb&m=req&key={{.AccountID}}")
	v.SetDefault("adapters.invibes.endpoint", "https://{{.Host}}/bid/ServerBidAdContent")
	v.SetDefault("adapters.kidoz.endpoint", "http://prebid-adapter.kidoz.net/openrtb2/auction?src=prebid-server")
	v.SetDefault("adapters.kubient.endpoint", "https://kssp.kbntx.ch/prebid")
	v.SetDefault("adapters.lifestreet.endpoint", "https://prebid.s2s.lfstmedia.com/adrequest")
	v.SetDefault("adapters.lockerdome.endpoint", "https://lockerdome.com/ladbid/prebidserver/openrtb2")
	v.SetDefault("adapters.logicad.endpoint", "https://pbs.ladsp.com/adrequest/prebidserver")
	v.SetDefault("adapters.lunamedia.endpoint", "http://api.lunamedia.io/xp/get?pubid={{.PublisherID}}")
	v.SetDefault("adapters.marsmedia.endpoint", "https://bid306.rtbsrv.com/bidder/?bid=f3xtet")
	v.SetDefault("adapters.mgid.endpoint", "https://prebid.mgid.com/prebid/")
	v.SetDefault("adapters.mobilefuse.endpoint", "http://mfx-us-east.mobilefuse.com/openrtb?pub_id={{.PublisherID}}")
	v.SetDefault("adapters.nanointeractive.endpoint", "https://ad.audiencemanager.de/hbs")
	v.SetDefault("adapters.ninthdecimal.endpoint", "http://rtb.ninthdecimal.com/xp/get?pubid={{.PublisherID}}")
	v.SetDefault("adapters.openx.endpoint", "http://rtb.openx.net/prebid")
	v.SetDefault("adapters.orbidder.endpoint", "https://orbidder.otto.de/openrtb2")
	v.SetDefault("adapters.pubmatic.endpoint", "https://hbopenbid.pubmatic.com/translator?source=prebid-server")
	v.SetDefault("adapters.pubnative.endpoint", "http://dsp.pubnative.net/bid/v1/request")
	v.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	v.SetDefault("adapters.rhythmone.endpoint", "http://tag.1rx.io/rmp")
	v.SetDefault("adapters.rtbhouse.endpoint", "http://prebidserver-s2s-ams.creativecdn.com/bidder/prebidserver/bids")
	v.SetDefault("adapters.rubicon.disabled", true)
	v.SetDefault("adapters.rubicon.endpoint", "http://exapi-us-east.rubiconproject.com/a/api/exchange.json")
	v.SetDefault("adapters.sharethrough.endpoint", "http://btlr.sharethrough.com/FGMrCMMc/v1")
	v.SetDefault("adapters.smaato.endpoint", "https://prebid.ad.smaato.net/oapi/prebid")
	v.SetDefault("adapters.smartadserver.endpoint", "https://ssb.smartadserver.com")
	v.SetDefault("adapters.smartrtb.endpoint", "http://market-east.smrtb.com/json/publisher/rtb?pubid={{.PublisherID}}")
	v.SetDefault("adapters.smartyads.endpoint", "http://{{.Host}}.smartyads.com/bid?rtb_seat_id={{.SourceId}}&secret_key={{.AccountID}}")
	v.SetDefault("adapters.somoaudience.endpoint", "http://publisher-east.mobileadtrading.com/rtb/bid")
	v.SetDefault("adapters.sonobi.endpoint", "https://apex.go.sonobi.com/prebid?partnerid=71d9d3d8af")
	v.SetDefault("adapters.sovrn.endpoint", "http://ap.lijit.com/rtb/bid?src=prebid_server")
	v.SetDefault("adapters.synacormedia.endpoint", "http://{{.Host}}.technoratimedia.com/openrtb/bids/{{.Host}}")
	v.SetDefault("adapters.tappx.endpoint", "https://{{.Host}}")
	v.SetDefault("adapters.telaria.endpoint", "https://ads.tremorhub.com/ad/rtb/prebid")
	v.SetDefault("adapters.triplelift_native.disabled", true)
	v.SetDefault("adapters.triplelift_native.extra_info", "{\"publisher_whitelist\":[]}")
	v.SetDefault("adapters.triplelift.endpoint", "https://tlx.3lift.com/s2s/auction?sra=1&supplier_id=20")
	v.SetDefault("adapters.ucfunnel.endpoint", "http://pbs.aralego.com/prebid")
	v.SetDefault("adapters.unruly.endpoint", "http://targeting.unrulymedia.com/openrtb/2.2")
	v.SetDefault("adapters.valueimpression.endpoint", "https://rtb.valueimpression.com/endpoint")
	v.SetDefault("adapters.verizonmedia.disabled", true)
	v.SetDefault("adapters.visx.endpoint", "https://t.visx.net/s2s_bid?wrapperType=s2s_prebid_standard")
	v.SetDefault("adapters.vrtcal.endpoint", "http://rtb.vrtcal.com/bidder_prebid.vap?ssp=1804")
	v.SetDefault("adapters.yeahmobi.endpoint", "https://{{.Host}}/prebid/bid")
	v.SetDefault("adapters.yieldlab.endpoint", "https://ad.yieldlab.net/yp/")
	v.SetDefault("adapters.yieldmo.endpoint", "https://ads.yieldmo.com/exchange/prebid-server")
	v.SetDefault("adapters.yieldone.endpoint", "https://y.one.impact-ad.jp/hbs_imp")
	v.SetDefault("adapters.zeroclickfraud.endpoint", "http://{{.Host}}/openrtb2?sid={{.SourceId}}")

	v.SetDefault("max_request_size", 1024*256)
	v.SetDefault("analytics.file.filename", "")
	v.SetDefault("analytics.pubstack.endpoint", "https://s2s.pbstck.com/v1")
	v.SetDefault("analytics.pubstack.scopeid", "change-me")
	v.SetDefault("analytics.pubstack.enabled", false)
	v.SetDefault("analytics.pubstack.configuration_refresh_delay", "2h")
	v.SetDefault("analytics.pubstack.buffers.size", "2MB")
	v.SetDefault("analytics.pubstack.buffers.count", 100)
	v.SetDefault("analytics.pubstack.buffers.timeout", "900s")
	v.SetDefault("amp_timeout_adjustment_ms", 0)
	v.SetDefault("gdpr.host_vendor_id", 0)
	v.SetDefault("gdpr.usersync_if_ambiguous", false)
	v.SetDefault("gdpr.timeouts_ms.init_vendorlist_fetches", 0)
	v.SetDefault("gdpr.timeouts_ms.active_vendorlist_fetch", 0)
	v.SetDefault("gdpr.non_standard_publishers", []string{""})
	v.SetDefault("gdpr.tcf1.fetch_gvl", true)
	v.SetDefault("gdpr.tcf1.fallback_gvl_path", "./static/tcf1/fallback_gvl.json")
	v.SetDefault("gdpr.tcf2.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose1.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose2.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose4.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose7.enabled", true)
	v.SetDefault("gdpr.tcf2.special_purpose1.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose_one_treatement.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose_one_treatement.access_allowed", true)
	v.SetDefault("gdpr.amp_exception", false)
	v.SetDefault("gdpr.eea_countries", []string{"ALA", "AUT", "BEL", "BGR", "HRV", "CYP", "CZE", "DNK", "EST",
		"FIN", "FRA", "GUF", "DEU", "GIB", "GRC", "GLP", "GGY", "HUN", "ISL", "IRL", "IMN", "ITA", "JEY", "LVA",
		"LIE", "LTU", "LUX", "MLT", "MTQ", "MYT", "NLD", "NOR", "POL", "PRT", "REU", "ROU", "BLM", "MAF", "SPM",
		"SVK", "SVN", "ESP", "SWE", "GBR"})
	v.SetDefault("ccpa.enforce", false)
	v.SetDefault("lmt.enforce", true)
	v.SetDefault("currency_converter.fetch_url", "https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json")
	v.SetDefault("currency_converter.fetch_interval_seconds", 1800) // fetch currency rates every 30 minutes
	v.SetDefault("currency_converter.stale_rates_seconds", 0)
	v.SetDefault("default_request.type", "")
	v.SetDefault("default_request.file.name", "")
	v.SetDefault("default_request.alias_info", false)
	v.SetDefault("blacklisted_apps", []string{""})
	v.SetDefault("blacklisted_accts", []string{""})
	v.SetDefault("account_required", false)
	v.SetDefault("account_defaults.disabled", false)
	v.SetDefault("certificates_file", "")
	v.SetDefault("auto_gen_source_tid", true)

	v.SetDefault("request_timeout_headers.request_time_in_queue", "")
	v.SetDefault("request_timeout_headers.request_timeout_in_queue", "")

	v.SetDefault("debug.timeout_notification.log", false)
	v.SetDefault("debug.timeout_notification.sampling_rate", 0.0)
	v.SetDefault("debug.timeout_notification.fail_only", false)

	/* IPv4
	/*  Site Local: 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	/*  Link Local: 169.254.0.0/16
	/*  Loopback:   127.0.0.0/8
	/*
	/* IPv6
	/*  Loopback:      ::1/128
	/*  Documentation: 2001:db8::/32
	/*  Unique Local:  fc00::/7
	/*  Link Local:    fe80::/10
	/*  Multicast:     ff00::/8
	*/
	v.SetDefault("request_validation.ipv4_private_networks", []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "169.254.0.0/16", "127.0.0.0/8"})
	v.SetDefault("request_validation.ipv6_private_networks", []string{"::1/128", "fc00::/7", "fe80::/10", "ff00::/8", "2001:db8::/32"})

	// Set environment variable support:
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetTypeByDefaultValue(true)
	v.SetEnvPrefix("PBS")
	v.AutomaticEnv()
	v.ReadInConfig()

	// Migrate config settings to maintain compatibility with old configs
	migrateConfig(v)
}

func migrateConfig(v *viper.Viper) {
	// if stored_requests.filesystem is not a map in conf file as expected from defaults,
	// means we have old-style settings; migrate them to new filesystem map to avoid breaking viper
	if _, ok := v.Get("stored_requests.filesystem").(map[string]interface{}); !ok {
		glog.Warning("stored_requests.filesystem should be changed to stored_requests.filesystem.enabled")
		glog.Warning("stored_requests.directorypath should be changed to stored_requests.filesystem.directorypath")
		m := v.GetStringMap("stored_requests.filesystem")
		m["enabled"] = v.GetBool("stored_requests.filesystem")
		m["directorypath"] = v.GetString("stored_requests.directorypath")
		v.Set("stored_requests.filesystem", m)
	}
}

func setBidderDefaults(v *viper.Viper, bidder string) {
	adapterCfgPrefix := "adapters."
	v.SetDefault(adapterCfgPrefix+bidder+".endpoint", "")
	v.SetDefault(adapterCfgPrefix+bidder+".usersync_url", "")
	v.SetDefault(adapterCfgPrefix+bidder+".platform_id", "")
	v.SetDefault(adapterCfgPrefix+bidder+".app_secret", "")
	v.SetDefault(adapterCfgPrefix+bidder+".xapi.username", "")
	v.SetDefault(adapterCfgPrefix+bidder+".xapi.password", "")
	v.SetDefault(adapterCfgPrefix+bidder+".xapi.tracker", "")
	v.SetDefault(adapterCfgPrefix+bidder+".disabled", false)
	v.SetDefault(adapterCfgPrefix+bidder+".partner_id", "")
	v.SetDefault(adapterCfgPrefix+bidder+".extra_info", "")
}

func isValidCookieSize(maxCookieSize int) error {
	// If a non-zero-less-than-500-byte "host_cookie.max_cookie_size_bytes" value was specified in the
	// environment configuration of prebid-server, default to 500 bytes
	if maxCookieSize != 0 && maxCookieSize < MIN_COOKIE_SIZE_BYTES {
		return fmt.Errorf("Configured cookie size is less than allowed minimum size of %d \n", MIN_COOKIE_SIZE_BYTES)
	}
	return nil
}
