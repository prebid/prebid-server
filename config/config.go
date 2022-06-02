package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/spf13/viper"
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
	// GarbageCollectorThreshold allocates virtual memory (in bytes) which is not used by PBS but
	// serves as a hack to trigger the garbage collector only when the heap reaches at least this size.
	// More info: https://github.com/golang/go/issues/48409
	GarbageCollectorThreshold int `mapstructure:"garbage_collector_threshold"`
	// StatusResponse is the string which will be returned by the /status endpoint when things are OK.
	// If empty, it will return a 204 with no content.
	StatusResponse    string          `mapstructure:"status_response"`
	AuctionTimeouts   AuctionTimeouts `mapstructure:"auction_timeouts_ms"`
	CacheURL          Cache           `mapstructure:"cache"`
	ExtCacheURL       ExternalCache   `mapstructure:"external_cache"`
	RecaptchaSecret   string          `mapstructure:"recaptcha_secret"`
	HostCookie        HostCookie      `mapstructure:"host_cookie"`
	Metrics           Metrics         `mapstructure:"metrics"`
	StoredRequests    StoredRequests  `mapstructure:"stored_requests"`
	StoredRequestsAMP StoredRequests  `mapstructure:"stored_amp_req"`
	CategoryMapping   StoredRequests  `mapstructure:"category_mapping"`
	VTrack            VTrack          `mapstructure:"vtrack"`
	Event             Event           `mapstructure:"event"`
	Accounts          StoredRequests  `mapstructure:"accounts"`
	UserSync          UserSync        `mapstructure:"user_sync"`
	// Note that StoredVideo refers to stored video requests, and has nothing to do with caching video creatives.
	StoredVideo     StoredRequests `mapstructure:"stored_video_req"`
	StoredResponses StoredRequests `mapstructure:"stored_responses"`

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
	//When true, new bid id will be generated in seatbid[].bid[].ext.prebid.bidid and used in event urls instead
	GenerateBidID bool `mapstructure:"generate_bid_id"`
	// GenerateRequestID overrides the bidrequest.id in an AMP Request or an App Stored Request with a generated UUID if set to true. The default is false.
	GenerateRequestID bool `mapstructure:"generate_request_id"`
}

const MIN_COOKIE_SIZE_BYTES = 500

type HTTPClient struct {
	MaxConnsPerHost     int `mapstructure:"max_connections_per_host"`
	MaxIdleConns        int `mapstructure:"max_idle_connections"`
	MaxIdleConnsPerHost int `mapstructure:"max_idle_connections_per_host"`
	IdleConnTimeout     int `mapstructure:"idle_connection_timeout_seconds"`
}

func (cfg *Configuration) validate(v *viper.Viper) []error {
	var errs []error
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
	errs = cfg.GDPR.validate(v, errs)
	errs = cfg.CurrencyConverter.validate(errs)
	errs = validateAdapters(cfg.Adapters, errs)
	errs = cfg.Debug.validate(errs)
	errs = cfg.ExtCacheURL.validate(errs)
	if cfg.AccountDefaults.Disabled {
		glog.Warning(`With account_defaults.disabled=true, host-defined accounts must exist and have "disabled":false. All other requests will be rejected.`)
	}
	if cfg.AccountDefaults.Events.Enabled {
		glog.Warning(`account_defaults.events will currently not do anything as the feature is still under development. Please follow https://github.com/prebid/prebid-server/issues/1725 for more updates`)
	}
	return errs
}

type AuctionTimeouts struct {
	// The default timeout is used if the user's request didn't define one. Use 0 if there's no default.
	Default uint64 `mapstructure:"default"`
	// The max timeout is used as an absolute cap, to prevent excessively long ones. Use 0 for no cap
	Max uint64 `mapstructure:"max"`
}

func (cfg *AuctionTimeouts) validate(errs []error) []error {
	if cfg.Max < cfg.Default {
		errs = append(errs, fmt.Errorf("auction_timeouts_ms.max cannot be less than auction_timeouts_ms.default. max=%d, default=%d", cfg.Max, cfg.Default))
	}
	return errs
}

func (data *ExternalCache) validate(errs []error) []error {
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
	if strings.Contains(data.Host, "://") {
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
	Enabled                 bool         `mapstructure:"enabled"`
	HostVendorID            int          `mapstructure:"host_vendor_id"`
	DefaultValue            string       `mapstructure:"default_value"`
	Timeouts                GDPRTimeouts `mapstructure:"timeouts_ms"`
	NonStandardPublishers   []string     `mapstructure:"non_standard_publishers,flow"`
	NonStandardPublisherMap map[string]struct{}
	TCF2                    TCF2 `mapstructure:"tcf2"`
	AMPException            bool `mapstructure:"amp_exception"` // Deprecated: Use account-level GDPR settings (gdpr.integration_enabled.amp) instead
	// EEACountries (EEA = European Economic Area) are a list of countries where we should assume GDPR applies.
	// If the gdpr flag is unset in a request, but geo.country is set, we will assume GDPR applies if and only
	// if the country matches one on this list. If both the GDPR flag and country are not set, we default
	// to DefaultValue
	EEACountries    []string `mapstructure:"eea_countries"`
	EEACountriesMap map[string]struct{}
}

func (cfg *GDPR) validate(v *viper.Viper, errs []error) []error {
	if !v.IsSet("gdpr.default_value") {
		errs = append(errs, fmt.Errorf("gdpr.default_value is required and must be specified"))
	} else if cfg.DefaultValue != "0" && cfg.DefaultValue != "1" {
		errs = append(errs, fmt.Errorf("gdpr.default_value must be 0 or 1"))
	}
	if cfg.HostVendorID < 0 || cfg.HostVendorID > 0xffff {
		errs = append(errs, fmt.Errorf("gdpr.host_vendor_id must be in the range [0, %d]. Got %d", 0xffff, cfg.HostVendorID))
	}
	if cfg.HostVendorID == 0 {
		glog.Warning("gdpr.host_vendor_id was not specified. Host company GDPR checks will be skipped.")
	}
	if cfg.AMPException == true {
		errs = append(errs, fmt.Errorf("gdpr.amp_exception has been discontinued and must be removed from your config. If you need to disable GDPR for AMP, you may do so per-account (gdpr.integration_enabled.amp) or at the host level for the default account (account_defaults.gdpr.integration_enabled.amp)"))
	}
	return cfg.validatePurposes(errs)
}

func (cfg *GDPR) validatePurposes(errs []error) []error {
	purposeConfigs := []TCF2Purpose{
		cfg.TCF2.Purpose1,
		cfg.TCF2.Purpose2,
		cfg.TCF2.Purpose3,
		cfg.TCF2.Purpose4,
		cfg.TCF2.Purpose5,
		cfg.TCF2.Purpose6,
		cfg.TCF2.Purpose7,
		cfg.TCF2.Purpose8,
		cfg.TCF2.Purpose9,
		cfg.TCF2.Purpose10,
	}

	for i := 0; i < len(purposeConfigs); i++ {
		enforcePurposeValue := purposeConfigs[i].EnforcePurpose
		enforcePurposeField := fmt.Sprintf("gdpr.tcf2.purpose%d.enforce_purpose", (i + 1))

		if enforcePurposeValue != TCF2NoEnforcement && enforcePurposeValue != TCF2FullEnforcement {
			errs = append(errs, fmt.Errorf("%s must be \"no\" or \"full\". Got %s", enforcePurposeField, enforcePurposeValue))
		}
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

const (
	TCF2FullEnforcement = "full"
	TCF2NoEnforcement   = "no"
)

// TCF2 defines the TCF2 specific configurations for GDPR
type TCF2 struct {
	Enabled             bool                    `mapstructure:"enabled"`
	Purpose1            TCF2Purpose             `mapstructure:"purpose1"`
	Purpose2            TCF2Purpose             `mapstructure:"purpose2"`
	Purpose3            TCF2Purpose             `mapstructure:"purpose3"`
	Purpose4            TCF2Purpose             `mapstructure:"purpose4"`
	Purpose5            TCF2Purpose             `mapstructure:"purpose5"`
	Purpose6            TCF2Purpose             `mapstructure:"purpose6"`
	Purpose7            TCF2Purpose             `mapstructure:"purpose7"`
	Purpose8            TCF2Purpose             `mapstructure:"purpose8"`
	Purpose9            TCF2Purpose             `mapstructure:"purpose9"`
	Purpose10           TCF2Purpose             `mapstructure:"purpose10"`
	SpecialFeature1     TCF2SpecialFeature      `mapstructure:"special_feature1"`
	PurposeOneTreatment TCF2PurposeOneTreatment `mapstructure:"purpose_one_treatment"`
}

// Making a purpose struct so purpose specific details can be added later.
type TCF2Purpose struct {
	Enabled        bool   `mapstructure:"enabled"` // Deprecated: Use enforce_purpose instead
	EnforcePurpose string `mapstructure:"enforce_purpose"`
	EnforceVendors bool   `mapstructure:"enforce_vendors"`
	// Array of vendor exceptions that is used to create the hash table VendorExceptionMap so vendor names can be instantly accessed
	VendorExceptions   []openrtb_ext.BidderName `mapstructure:"vendor_exceptions"`
	VendorExceptionMap map[openrtb_ext.BidderName]struct{}
}

type TCF2SpecialFeature struct {
	Enforce bool `mapstructure:"enforce"`
	// Array of vendor exceptions that is used to create the hash table VendorExceptionMap so vendor names can be instantly accessed
	VendorExceptions   []openrtb_ext.BidderName `mapstructure:"vendor_exceptions"`
	VendorExceptionMap map[openrtb_ext.BidderName]struct{}
}

type TCF2PurposeOneTreatment struct {
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

func (cfg *CurrencyConverter) validate(errs []error) []error {
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

	// True if we don't want to collect the per adapter GDPR request blocked metric
	AdapterGDPRRequestBlocked bool `mapstructure:"adapter_gdpr_request_blocked"`
}

func (cfg *Metrics) validate(errs []error) []error {
	return cfg.Prometheus.validate(errs)
}

type InfluxMetrics struct {
	Host               string `mapstructure:"host"`
	Database           string `mapstructure:"database"`
	Measurement        string `mapstructure:"measurement"`
	Username           string `mapstructure:"username"`
	Password           string `mapstructure:"password"`
	AlignTimestamps    bool   `mapstructure:"align_timestamps"`
	MetricSendInterval int    `mapstructure:"metric_send_interval"`
}

type PrometheusMetrics struct {
	Port             int    `mapstructure:"port"`
	Namespace        string `mapstructure:"namespace"`
	Subsystem        string `mapstructure:"subsystem"`
	TimeoutMillisRaw int    `mapstructure:"timeout_ms"`
}

func (cfg *PrometheusMetrics) validate(errs []error) []error {
	if cfg.Port > 0 && cfg.TimeoutMillisRaw <= 0 {
		errs = append(errs, fmt.Errorf("metrics.prometheus.timeout_ms must be positive if metrics.prometheus.port is defined. Got timeout=%d and port=%d", cfg.TimeoutMillisRaw, cfg.Port))
	}
	return errs
}

func (m *PrometheusMetrics) Timeout() time.Duration {
	return time.Duration(m.TimeoutMillisRaw) * time.Millisecond
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
	OverrideToken       string              `mapstructure:"override_token"`
}

func (cfg *Debug) validate(errs []error) []error {
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

func (cfg *TimeoutNotification) validate(errs []error) []error {
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

	c.GDPR.EEACountriesMap = make(map[string]struct{}, len(c.GDPR.EEACountries))
	for _, v := range c.GDPR.EEACountries {
		c.GDPR.EEACountriesMap[v] = s
	}

	// To look for a purpose's vendor exceptions in O(1) time, for each purpose we fill this hash table located in the
	// VendorExceptions field of the GDPR.TCF2.PurposeX struct defined in this file
	purposeConfigs := []*TCF2Purpose{
		&c.GDPR.TCF2.Purpose1,
		&c.GDPR.TCF2.Purpose2,
		&c.GDPR.TCF2.Purpose3,
		&c.GDPR.TCF2.Purpose4,
		&c.GDPR.TCF2.Purpose5,
		&c.GDPR.TCF2.Purpose6,
		&c.GDPR.TCF2.Purpose7,
		&c.GDPR.TCF2.Purpose8,
		&c.GDPR.TCF2.Purpose9,
		&c.GDPR.TCF2.Purpose10,
	}
	for c := 0; c < len(purposeConfigs); c++ {
		purposeConfigs[c].VendorExceptionMap = make(map[openrtb_ext.BidderName]struct{})

		for v := 0; v < len(purposeConfigs[c].VendorExceptions); v++ {
			bidderName := purposeConfigs[c].VendorExceptions[v]
			purposeConfigs[c].VendorExceptionMap[bidderName] = struct{}{}
		}
	}

	// To look for a special feature's vendor exceptions in O(1) time, we fill this hash table with bidders located in the
	// VendorExceptions field of the GDPR.TCF2.SpecialFeature1 struct defined in this file
	c.GDPR.TCF2.SpecialFeature1.VendorExceptionMap = make(map[openrtb_ext.BidderName]struct{})
	for v := 0; v < len(c.GDPR.TCF2.SpecialFeature1.VendorExceptions); v++ {
		bidderName := c.GDPR.TCF2.SpecialFeature1.VendorExceptions[v]
		c.GDPR.TCF2.SpecialFeature1.VendorExceptionMap[bidderName] = struct{}{}
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
	if errs := c.validate(v); len(errs) > 0 {
		return &c, errortypes.NewAggregateError("validation errors", errs)
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
	v.SetDefault("garbage_collector_threshold", 0)
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
	v.SetDefault("metrics.disabled_metrics.adapter_gdpr_request_blocked", false)
	v.SetDefault("metrics.influxdb.host", "")
	v.SetDefault("metrics.influxdb.database", "")
	v.SetDefault("metrics.influxdb.measurement", "")
	v.SetDefault("metrics.influxdb.username", "")
	v.SetDefault("metrics.influxdb.password", "")
	v.SetDefault("metrics.influxdb.align_timestamps", false)
	v.SetDefault("metrics.influxdb.metric_send_interval", 20)
	v.SetDefault("metrics.prometheus.port", 0)
	v.SetDefault("metrics.prometheus.namespace", "")
	v.SetDefault("metrics.prometheus.subsystem", "")
	v.SetDefault("metrics.prometheus.timeout_ms", 10000)
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
	v.SetDefault("stored_requests.in_memory_cache.resp_cache_size_bytes", 0)
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
	v.SetDefault("stored_video_req.in_memory_cache.resp_cache_size_bytes", 0)
	v.SetDefault("stored_video_req.cache_events.enabled", false)
	v.SetDefault("stored_video_req.cache_events.endpoint", "")
	v.SetDefault("stored_video_req.http_events.endpoint", "")
	v.SetDefault("stored_video_req.http_events.refresh_rate_seconds", 0)
	v.SetDefault("stored_video_req.http_events.timeout_ms", 0)
	v.SetDefault("stored_responses.filesystem.enabled", false)
	v.SetDefault("stored_responses.filesystem.directorypath", "")
	v.SetDefault("stored_responses.postgres.connection.dbname", "")
	v.SetDefault("stored_responses.postgres.connection.host", "")
	v.SetDefault("stored_responses.postgres.connection.port", 0)
	v.SetDefault("stored_responses.postgres.connection.user", "")
	v.SetDefault("stored_responses.postgres.connection.password", "")
	v.SetDefault("stored_responses.postgres.fetcher.query", "")
	v.SetDefault("stored_responses.postgres.initialize_caches.timeout_ms", 0)
	v.SetDefault("stored_responses.postgres.initialize_caches.query", "")
	v.SetDefault("stored_responses.postgres.poll_for_updates.refresh_rate_seconds", 0)
	v.SetDefault("stored_responses.postgres.poll_for_updates.timeout_ms", 0)
	v.SetDefault("stored_responses.postgres.poll_for_updates.query", "")
	v.SetDefault("stored_responses.http.endpoint", "")
	v.SetDefault("stored_responses.in_memory_cache.type", "none")
	v.SetDefault("stored_responses.in_memory_cache.ttl_seconds", 0)
	v.SetDefault("stored_responses.in_memory_cache.request_cache_size_bytes", 0)
	v.SetDefault("stored_responses.in_memory_cache.imp_cache_size_bytes", 0)
	v.SetDefault("stored_responses.in_memory_cache.resp_cache_size_bytes", 0)
	v.SetDefault("stored_responses.cache_events.enabled", false)
	v.SetDefault("stored_responses.cache_events.endpoint", "")
	v.SetDefault("stored_responses.http_events.endpoint", "")
	v.SetDefault("stored_responses.http_events.refresh_rate_seconds", 0)
	v.SetDefault("stored_responses.http_events.timeout_ms", 0)

	v.SetDefault("vtrack.timeout_ms", 2000)
	v.SetDefault("vtrack.allow_unknown_bidder", true)
	v.SetDefault("vtrack.enabled", true)

	v.SetDefault("event.timeout_ms", 1000)

	v.SetDefault("accounts.filesystem.enabled", false)
	v.SetDefault("accounts.filesystem.directorypath", "./stored_requests/data/by_id")
	v.SetDefault("accounts.in_memory_cache.type", "none")

	// some adapters append the user id to the end of the redirect url instead of using
	// macro substitution. it is important for the uid to be the last query parameter.
	v.SetDefault("user_sync.redirect_url", "{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&f={{.SyncType}}&uid={{.UserMacro}}")

	for _, bidder := range openrtb_ext.CoreBidderNames() {
		setBidderDefaults(v, strings.ToLower(string(bidder)))
	}

	// Disabling adapters by default that require some specific config params.
	// If you're using one of these, make sure you check out the documentation (https://github.com/prebid/prebid-server/tree/master/docs/bidders)
	// for them and specify all the parameters they need for them to work correctly.
	v.SetDefault("adapters.33across.endpoint", "https://ssc.33across.com/api/v1/s2s")
	v.SetDefault("adapters.33across.partner_id", "")
	v.SetDefault("adapters.aceex.endpoint", "http://bl-us.aceex.io/?uqhash={{.AccountID}}")
	v.SetDefault("adapters.acuityads.endpoint", "http://{{.Host}}.admanmedia.com/bid?token={{.AccountID}}")
	v.SetDefault("adapters.adf.endpoint", "https://adx.adform.net/adx/openrtb")
	v.SetDefault("adapters.adform.endpoint", "https://adx.adform.net/adx/openrtb")
	v.SetDefault("adapters.adgeneration.endpoint", "https://d.socdm.com/adsv/v1")
	v.SetDefault("adapters.adhese.endpoint", "https://ads-{{.AccountID}}.adhese.com/json")
	v.SetDefault("adapters.adkernel.endpoint", "https://pbs.adksrv.com/hb?zone={{.ZoneID}}")
	v.SetDefault("adapters.adkerneladn.endpoint", "https://pbs2.adksrv.com/rtbpub?account={{.PublisherID}}")
	v.SetDefault("adapters.adman.endpoint", "http://pub.admanmedia.com/?c=o&m=ortb")
	v.SetDefault("adapters.admixer.endpoint", "http://inv-nets.admixer.net/pbs.aspx")
	v.SetDefault("adapters.adocean.endpoint", "https://{{.Host}}")
	v.SetDefault("adapters.adnuntius.endpoint", "https://ads.adnuntius.delivery/i")
	v.SetDefault("adapters.adoppler.endpoint", "http://{{.AccountID}}.trustedmarketplace.io/ads/processHeaderBid/{{.AdUnit}}")
	v.SetDefault("adapters.adot.endpoint", "https://dsp.adotmob.com/headerbidding{PUBLISHER_PATH}/bidrequest")
	v.SetDefault("adapters.adpone.endpoint", "http://rtb.adpone.com/bid-request?src=prebid_server")
	v.SetDefault("adapters.adprime.endpoint", "http://delta.adprime.com/pserver")
	v.SetDefault("adapters.adtarget.endpoint", "http://ghb.console.adtarget.com.tr/pbs/ortb")
	v.SetDefault("adapters.adtelligent.endpoint", "http://ghb.adtelligent.com/pbs/ortb")
	v.SetDefault("adapters.advangelists.endpoint", "http://nep.advangelists.com/xp/get?pubid={{.PublisherID}}")
	v.SetDefault("adapters.adview.endpoint", "https://bid.adview.com/agent/thirdAdxService/{{.AccountID}}")
	v.SetDefault("adapters.adxcg.disabled", true)
	v.SetDefault("adapters.adyoulike.endpoint", "https://broker.omnitagjs.com/broker/bid?partnerId=19340f4f097d16f41f34fc0274981ca4")
	v.SetDefault("adapters.aja.endpoint", "https://ad.as.amanad.adtdp.com/v1/bid/4")
	v.SetDefault("adapters.algorix.endpoint", "https://{{.Host}}.svr-algorix.com/rtb/sa?sid={{.SourceId}}&token={{.AccountID}}")
	v.SetDefault("adapters.amx.endpoint", "http://pbs.amxrtb.com/auction/openrtb")
	v.SetDefault("adapters.apacdex.endpoint", "http://useast.quantumdex.io/auction/pbs")
	v.SetDefault("adapters.applogy.endpoint", "http://rtb.applogy.com/v1/prebid")
	v.SetDefault("adapters.appnexus.endpoint", "http://ib.adnxs.com/openrtb2") // Docs: https://wiki.appnexus.com/display/supply/Incoming+Bid+Request+from+SSPs
	v.SetDefault("adapters.appnexus.platform_id", "5")
	v.SetDefault("adapters.audiencenetwork.disabled", true)
	v.SetDefault("adapters.audiencenetwork.endpoint", "https://an.facebook.com/placementbid.ortb")
	v.SetDefault("adapters.avocet.disabled", true)
	v.SetDefault("adapters.axonix.endpoint", "https://openrtb-us-east-1.axonix.com/supply/prebid-server/{{.AccountID}}")
	v.SetDefault("adapters.beachfront.endpoint", "https://display.bfmio.com/prebid_display")
	v.SetDefault("adapters.beachfront.extra_info", "{\"video_endpoint\":\"https://reachms.bfmio.com/bid.json?exchange_id\"}")
	v.SetDefault("adapters.beintoo.endpoint", "https://ib.beintoo.com/um")
	v.SetDefault("adapters.between.endpoint", "http://{{.Host}}.betweendigital.com/openrtb_bid?sspId={{.PublisherID}}")
	v.SetDefault("adapters.bidmachine.endpoint", "https://{{.Host}}.bidmachine.io")
	v.SetDefault("adapters.bidmyadz.endpoint", "http://endpoint.bidmyadz.com/c0f68227d14ed938c6c49f3967cbe9bc")
	v.SetDefault("adapters.bidscube.endpoint", "http://supply.bidscube.com/?c=o&m=rtb")
	v.SetDefault("adapters.bizzclick.endpoint", "http://us-e-node1.bizzclick.com/bid?rtb_seat_id={{.SourceId}}&secret_key={{.AccountID}}")
	v.SetDefault("adapters.bmtm.endpoint", "https://one.elitebidder.com/api/pbs")
	v.SetDefault("adapters.brightroll.endpoint", "http://east-bid.ybp.yahoo.com/bid/appnexuspbs")
	v.SetDefault("adapters.coinzilla.endpoint", "http://request-global.czilladx.com/serve/prebid-server.php")
	v.SetDefault("adapters.colossus.endpoint", "http://colossusssp.com/?c=o&m=rtb")
	v.SetDefault("adapters.compass.endpoint", "http://sa-lb.deliverimp.com/pserver")
	v.SetDefault("adapters.connectad.endpoint", "http://bidder.connectad.io/API?src=pbs")
	v.SetDefault("adapters.consumable.endpoint", "https://e.serverbid.com/api/v2")
	v.SetDefault("adapters.conversant.endpoint", "http://api.hb.ad.cpe.dotomi.com/cvx/server/hb/ortb/25")
	v.SetDefault("adapters.cpmstar.endpoint", "https://server.cpmstar.com/openrtbbidrq.aspx")
	v.SetDefault("adapters.criteo.endpoint", "https://bidder.criteo.com/cdb?profileId=230")
	v.SetDefault("adapters.datablocks.endpoint", "http://{{.Host}}/openrtb2?sid={{.SourceId}}")
	v.SetDefault("adapters.decenterads.endpoint", "http://supply.decenterads.com/?c=o&m=rtb")
	v.SetDefault("adapters.deepintent.endpoint", "https://prebid.deepintent.com/prebid")
	v.SetDefault("adapters.dmx.endpoint", "https://dmx-direct.districtm.io/b/v2")
	v.SetDefault("adapters.emx_digital.endpoint", "https://hb.emxdgt.com")
	v.SetDefault("adapters.engagebdr.endpoint", "http://dsp.bnmla.com/hb")
	v.SetDefault("adapters.eplanning.endpoint", "http://rtb.e-planning.net/pbs/1")
	v.SetDefault("adapters.epom.endpoint", "https://an.epom.com/ortb")
	v.SetDefault("adapters.epom.disabled", true)
	v.SetDefault("adapters.e_volution.endpoint", "http://service.e-volution.ai/pbserver")
	v.SetDefault("adapters.gamma.endpoint", "https://hb.gammaplatform.com/adx/request/")
	v.SetDefault("adapters.gamoshi.endpoint", "https://rtb.gamoshi.io")
	v.SetDefault("adapters.grid.endpoint", "https://grid.bidswitch.net/sp_bid?sp=prebid")
	v.SetDefault("adapters.groupm.endpoint", "https://hbopenbid.pubmatic.com/translator?source=prebid-server")
	v.SetDefault("adapters.gumgum.endpoint", "https://g2.gumgum.com/providers/prbds2s/bid")
	v.SetDefault("adapters.huaweiads.endpoint", "https://acd.op.hicloud.com/ppsadx/getResult")
	v.SetDefault("adapters.huaweiads.disabled", true)
	v.SetDefault("adapters.impactify.endpoint", "https://sonic.impactify.media/bidder")
	v.SetDefault("adapters.improvedigital.endpoint", "http://ad.360yield.com/pbs")
	v.SetDefault("adapters.inmobi.endpoint", "https://api.w.inmobi.com/showad/openrtb/bidder/prebid")
	v.SetDefault("adapters.interactiveoffers.endpoint", "https://prebid-server.ioadx.com/bidRequest/?partnerId={{.AccountID}}")
	v.SetDefault("adapters.ix.disabled", true)
	v.SetDefault("adapters.janet.endpoint", "http://ghb.bidder.jmgads.com/pbs/ortb")
	v.SetDefault("adapters.jixie.endpoint", "https://hb.jixie.io/v2/hbsvrpost")
	v.SetDefault("adapters.kargo.endpoint", "https://krk.kargo.com/api/v1/openrtb")
	v.SetDefault("adapters.kayzen.endpoint", "https://bids-{{.ZoneID}}.bidder.kayzen.io/?exchange={{.AccountID}}")
	v.SetDefault("adapters.krushmedia.endpoint", "http://ads4.krushmedia.com/?c=rtb&m=req&key={{.AccountID}}")
	v.SetDefault("adapters.invibes.endpoint", "https://{{.ZoneID}}.videostep.com/bid/ServerBidAdContent")
	v.SetDefault("adapters.iqzone.endpoint", "http://smartssp-us-east.iqzone.com/pserver")
	v.SetDefault("adapters.kidoz.endpoint", "http://prebid-adapter.kidoz.net/openrtb2/auction?src=prebid-server")
	v.SetDefault("adapters.kubient.endpoint", "https://kssp.kbntx.ch/prebid")
	v.SetDefault("adapters.lockerdome.endpoint", "https://lockerdome.com/ladbid/prebidserver/openrtb2")
	v.SetDefault("adapters.logicad.endpoint", "https://pbs.ladsp.com/adrequest/prebidserver")
	v.SetDefault("adapters.lunamedia.endpoint", "http://api.lunamedia.io/xp/get?pubid={{.PublisherID}}")
	v.SetDefault("adapters.sa_lunamedia.endpoint", "http://balancer.lmgssp.com/pserver")
	v.SetDefault("adapters.madvertise.endpoint", "https://mobile.mng-ads.com/bidrequest{{.ZoneID}}")
	v.SetDefault("adapters.marsmedia.endpoint", "https://bid306.rtbsrv.com/bidder/?bid=f3xtet")
	v.SetDefault("adapters.mediafuse.endpoint", "http://ghb.hbmp.mediafuse.com/pbs/ortb")
	v.SetDefault("adapters.medianet.endpoint", "https://prebid-adapter.media.net/rtb/pb/prebids2s")
	v.SetDefault("adapters.medianet.extra_info", "https://medianet.golang.pbs.com")
	v.SetDefault("adapters.mgid.endpoint", "https://prebid.mgid.com/prebid/")
	v.SetDefault("adapters.mobilefuse.endpoint", "http://mfx.mobilefuse.com/openrtb?pub_id={{.PublisherID}}")
	v.SetDefault("adapters.mobfoxpb.endpoint", "http://bes.mobfox.com/?c=__route__&m=__method__&key=__key__")
	v.SetDefault("adapters.nanointeractive.endpoint", "https://ad.audiencemanager.de/hbs")
	v.SetDefault("adapters.nextmillennium.endpoint", "https://pbs.nextmillmedia.com/openrtb2/auction")
	v.SetDefault("adapters.ninthdecimal.endpoint", "http://rtb.ninthdecimal.com/xp/get?pubid={{.PublisherID}}")
	v.SetDefault("adapters.nobid.endpoint", "https://ads.servenobid.com/ortb_adreq?tek=pbs&ver=1")
	v.SetDefault("adapters.onetag.endpoint", "https://prebid-server.onetag-sys.com/prebid-server/{{.PublisherID}}")
	v.SetDefault("adapters.openweb.endpoint", "http://ghb.spotim.market/pbs/ortb")
	v.SetDefault("adapters.openx.endpoint", "http://rtb.openx.net/prebid")
	v.SetDefault("adapters.operaads.endpoint", "https://s.adx.opera.com/ortb/v2/{{.PublisherID}}?ep={{.AccountID}}")
	v.SetDefault("adapters.orbidder.endpoint", "https://orbidder.otto.de/openrtb2")
	v.SetDefault("adapters.outbrain.endpoint", "https://prebidtest.zemanta.com/api/bidder/prebidtest/bid/")
	v.SetDefault("adapters.pangle.disabled", true)
	v.SetDefault("adapters.pgam.endpoint", "http://ghb.pgamssp.com/pbs/ortb")
	v.SetDefault("adapters.pubmatic.endpoint", "https://hbopenbid.pubmatic.com/translator?source=prebid-server")
	v.SetDefault("adapters.pubnative.endpoint", "http://dsp.pubnative.net/bid/v1/request")
	v.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	v.SetDefault("adapters.quantumdex.endpoint", "http://useast.quantumdex.io/auction/pbs")
	v.SetDefault("adapters.revcontent.disabled", true)
	v.SetDefault("adapters.revcontent.endpoint", "https://trends.revcontent.com/rtb")
	v.SetDefault("adapters.rhythmone.endpoint", "http://tag.1rx.io/rmp")
	v.SetDefault("adapters.richaudience.endpoint", "http://ortb.richaudience.com/ortb/?bidder=pbs")
	v.SetDefault("adapters.rtbhouse.endpoint", "http://prebidserver-s2s-ams.creativecdn.com/bidder/prebidserver/bids")
	v.SetDefault("adapters.rubicon.disabled", true)
	v.SetDefault("adapters.rubicon.endpoint", "http://exapi-us-east.rubiconproject.com/a/api/exchange.json")
	v.SetDefault("adapters.sharethrough.endpoint", "https://btlr.sharethrough.com/universal/v1?supply_id=FGMrCMMc")
	v.SetDefault("adapters.silvermob.endpoint", "http://{{.Host}}.silvermob.com/marketplace/api/dsp/bid/{{.ZoneID}}")
	v.SetDefault("adapters.smaato.endpoint", "https://prebid.ad.smaato.net/oapi/prebid")
	v.SetDefault("adapters.smartadserver.endpoint", "https://ssb-global.smartadserver.com")
	v.SetDefault("adapters.smarthub.endpoint", "http://{{.Host}}-prebid.smart-hub.io/?seat={{.AccountID}}&token={{.SourceId}}")
	v.SetDefault("adapters.smartrtb.endpoint", "http://market-east.smrtb.com/json/publisher/rtb?pubid={{.PublisherID}}")
	v.SetDefault("adapters.smartyads.endpoint", "http://{{.Host}}.smartyads.com/bid?rtb_seat_id={{.SourceId}}&secret_key={{.AccountID}}")
	v.SetDefault("adapters.smilewanted.endpoint", "http://prebid-server.smilewanted.com")
	v.SetDefault("adapters.sonobi.endpoint", "https://apex.go.sonobi.com/prebid?partnerid=71d9d3d8af")
	v.SetDefault("adapters.sovrn.endpoint", "http://ap.lijit.com/rtb/bid?src=prebid_server")
	v.SetDefault("adapters.streamkey.endpoint", "http://ghb.hb.streamkey.net/pbs/ortb")
	v.SetDefault("adapters.synacormedia.endpoint", "http://{{.Host}}.technoratimedia.com/openrtb/bids/{{.Host}}")
	v.SetDefault("adapters.tappx.endpoint", "http://{{.Host}}")
	v.SetDefault("adapters.telaria.endpoint", "https://ads.tremorhub.com/ad/rtb/prebid")
	v.SetDefault("adapters.triplelift_native.disabled", true)
	v.SetDefault("adapters.triplelift_native.extra_info", "{\"publisher_whitelist\":[]}")
	v.SetDefault("adapters.triplelift.endpoint", "https://tlx.3lift.com/s2s/auction?sra=1&supplier_id=20")
	v.SetDefault("adapters.trustx.endpoint", "https://grid.bidswitch.net/sp_bid?sp=trustx")
	v.SetDefault("adapters.ucfunnel.endpoint", "https://pbs.aralego.com/prebid")
	v.SetDefault("adapters.unicorn.endpoint", "https://ds.uncn.jp/pb/0/bid.json")
	v.SetDefault("adapters.unruly.endpoint", "https://targeting.unrulymedia.com/unruly_prebid_server")
	v.SetDefault("adapters.valueimpression.endpoint", "http://useast.quantumdex.io/auction/pbs")
	v.SetDefault("adapters.verizonmedia.disabled", true)
	v.SetDefault("adapters.videobyte.endpoint", "https://x.videobyte.com/ortbhb")
	v.SetDefault("adapters.vidoomy.endpoint", "https://p.vidoomy.com/api/rtbserver/pbs")
	v.SetDefault("adapters.viewdeos.endpoint", "http://ghb.sync.viewdeos.com/pbs/ortb")
	v.SetDefault("adapters.visx.endpoint", "https://t.visx.net/s2s_bid?wrapperType=s2s_prebid_standard:0.1.0")
	v.SetDefault("adapters.vrtcal.endpoint", "http://rtb.vrtcal.com/bidder_prebid.vap?ssp=1804")
	v.SetDefault("adapters.yahoossp.disabled", true)
	v.SetDefault("adapters.yeahmobi.endpoint", "https://{{.Host}}/prebid/bid")
	v.SetDefault("adapters.yieldlab.endpoint", "https://ad.yieldlab.net/yp/")
	v.SetDefault("adapters.yieldmo.endpoint", "https://ads.yieldmo.com/exchange/prebid-server")
	v.SetDefault("adapters.yieldone.endpoint", "https://y.one.impact-ad.jp/hbs_imp")
	v.SetDefault("adapters.yssp.disabled", true)
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
	v.BindEnv("gdpr.default_value")
	v.SetDefault("gdpr.enabled", true)
	v.SetDefault("gdpr.host_vendor_id", 0)
	v.SetDefault("gdpr.timeouts_ms.init_vendorlist_fetches", 0)
	v.SetDefault("gdpr.timeouts_ms.active_vendorlist_fetch", 0)
	v.SetDefault("gdpr.non_standard_publishers", []string{""})
	v.SetDefault("gdpr.tcf2.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose1.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose2.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose3.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose4.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose5.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose6.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose7.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose8.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose9.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose10.enforce_vendors", true)
	v.SetDefault("gdpr.tcf2.purpose1.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose2.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose3.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose4.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose5.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose6.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose7.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose8.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose9.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("gdpr.tcf2.purpose10.vendor_exceptions", []openrtb_ext.BidderName{})
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
	v.SetDefault("account_defaults.debug_allow", true)
	v.SetDefault("certificates_file", "")
	v.SetDefault("auto_gen_source_tid", true)
	v.SetDefault("generate_bid_id", false)
	v.SetDefault("generate_request_id", false)

	v.SetDefault("request_timeout_headers.request_time_in_queue", "")
	v.SetDefault("request_timeout_headers.request_timeout_in_queue", "")

	v.SetDefault("debug.timeout_notification.log", false)
	v.SetDefault("debug.timeout_notification.sampling_rate", 0.0)
	v.SetDefault("debug.timeout_notification.fail_only", false)
	v.SetDefault("debug.override_token", "")

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
	migrateConfigPurposeOneTreatment(v)
	migrateConfigSpecialFeature1(v)
	migrateConfigTCF2PurposeEnabledFlags(v)

	// These defaults must be set after the migrate functions because those functions look for the presence of these
	// config fields and there isn't a way to detect presence of a config field using the viper package if a default
	// is set. Viper IsSet and Get functions consider default values.
	v.SetDefault("gdpr.tcf2.purpose1.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose2.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose3.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose4.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose5.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose6.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose7.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose8.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose9.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose10.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose1.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose2.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose3.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose4.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose5.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose6.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose7.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose8.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose9.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose10.enforce_purpose", TCF2FullEnforcement)
	v.SetDefault("gdpr.tcf2.purpose_one_treatment.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose_one_treatment.access_allowed", true)
	v.SetDefault("gdpr.tcf2.special_feature1.enforce", true)
	v.SetDefault("gdpr.tcf2.special_feature1.vendor_exceptions", []openrtb_ext.BidderName{})

	// Defaults for account_defaults.events.default_url
	v.SetDefault("account_defaults.events.default_url", "https://PBS_HOST/event?t=##PBS-EVENTTYPE##&vtype=##PBS-VASTEVENT##&b=##PBS-BIDID##&f=i&a=##PBS-ACCOUNTID##&ts=##PBS-TIMESTAMP##&bidder=##PBS-BIDDER##&int=##PBS-INTEGRATION##&mt=##PBS-MEDIATYPE##&ch=##PBS-CHANNEL##&aid=##PBS-AUCTIONID##&l=##PBS-LINEID##")
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

func migrateConfigPurposeOneTreatment(v *viper.Viper) {
	if oldConfig, ok := v.Get("gdpr.tcf2.purpose_one_treatement").(map[string]interface{}); ok {
		if v.IsSet("gdpr.tcf2.purpose_one_treatment") {
			glog.Warning("using gdpr.tcf2.purpose_one_treatment and ignoring deprecated gdpr.tcf2.purpose_one_treatement")
		} else {
			glog.Warning("gdpr.tcf2.purpose_one_treatement.enabled should be changed to gdpr.tcf2.purpose_one_treatment.enabled")
			glog.Warning("gdpr.tcf2.purpose_one_treatement.access_allowed should be changed to gdpr.tcf2.purpose_one_treatment.access_allowed")
			v.Set("gdpr.tcf2.purpose_one_treatment", oldConfig)
		}
	}
}

func migrateConfigSpecialFeature1(v *viper.Viper) {
	if oldConfig, ok := v.Get("gdpr.tcf2.special_purpose1").(map[string]interface{}); ok {
		if v.IsSet("gdpr.tcf2.special_feature1") {
			glog.Warning("using gdpr.tcf2.special_feature1 and ignoring deprecated gdpr.tcf2.special_purpose1")
		} else {
			glog.Warning("gdpr.tcf2.special_purpose1.enabled is deprecated and should be changed to gdpr.tcf2.special_feature1.enforce")
			glog.Warning("gdpr.tcf2.special_purpose1.vendor_exceptions is deprecated and should be changed to gdpr.tcf2.special_feature1.vendor_exceptions")
			v.Set("gdpr.tcf2.special_feature1.enforce", oldConfig["enabled"])
			v.Set("gdpr.tcf2.special_feature1.vendor_exceptions", oldConfig["vendor_exceptions"])
		}
	}
}

func migrateConfigTCF2PurposeEnabledFlags(v *viper.Viper) {
	for i := 1; i <= 10; i++ {
		oldField := fmt.Sprintf("gdpr.tcf2.purpose%d.enabled", i)
		newField := fmt.Sprintf("gdpr.tcf2.purpose%d.enforce_purpose", i)

		if v.IsSet(oldField) {
			oldConfig := v.GetBool(oldField)
			if v.IsSet(newField) {
				glog.Warningf("using %s and ignoring deprecated %s", newField, oldField)
			} else {
				glog.Warningf("%s is deprecated and should be changed to %s", oldField, newField)
				if oldConfig {
					v.Set(newField, TCF2FullEnforcement)
				} else {
					v.Set(newField, TCF2NoEnforcement)
				}
			}
		}

		if v.IsSet(newField) {
			if v.GetString(newField) == TCF2FullEnforcement {
				v.Set(oldField, "true")
			} else {
				v.Set(oldField, "false")
			}
		}
	}
}

func setBidderDefaults(v *viper.Viper, bidder string) {
	adapterCfgPrefix := "adapters." + bidder
	v.SetDefault(adapterCfgPrefix+".endpoint", "")
	v.SetDefault(adapterCfgPrefix+".usersync_url", "")
	v.SetDefault(adapterCfgPrefix+".platform_id", "")
	v.SetDefault(adapterCfgPrefix+".app_secret", "")
	v.SetDefault(adapterCfgPrefix+".xapi.username", "")
	v.SetDefault(adapterCfgPrefix+".xapi.password", "")
	v.SetDefault(adapterCfgPrefix+".xapi.tracker", "")
	v.SetDefault(adapterCfgPrefix+".disabled", false)
	v.SetDefault(adapterCfgPrefix+".partner_id", "")
	v.SetDefault(adapterCfgPrefix+".extra_info", "")

	v.BindEnv(adapterCfgPrefix + ".usersync.key")
	v.BindEnv(adapterCfgPrefix + ".usersync.default")
	v.BindEnv(adapterCfgPrefix + ".usersync.iframe.url")
	v.BindEnv(adapterCfgPrefix + ".usersync.iframe.redirect_url")
	v.BindEnv(adapterCfgPrefix + ".usersync.iframe.external_url")
	v.BindEnv(adapterCfgPrefix + ".usersync.iframe.user_macro")
	v.BindEnv(adapterCfgPrefix + ".usersync.redirect.url")
	v.BindEnv(adapterCfgPrefix + ".usersync.redirect.redirect_url")
	v.BindEnv(adapterCfgPrefix + ".usersync.redirect.external_url")
	v.BindEnv(adapterCfgPrefix + ".usersync.redirect.user_macro")
	v.BindEnv(adapterCfgPrefix + ".usersync.external_url")
	v.BindEnv(adapterCfgPrefix + ".usersync.support_cors")
}

func isValidCookieSize(maxCookieSize int) error {
	// If a non-zero-less-than-500-byte "host_cookie.max_cookie_size_bytes" value was specified in the
	// environment configuration of prebid-server, default to 500 bytes
	if maxCookieSize != 0 && maxCookieSize < MIN_COOKIE_SIZE_BYTES {
		return fmt.Errorf("Configured cookie size is less than allowed minimum size of %d \n", MIN_COOKIE_SIZE_BYTES)
	}
	return nil
}
