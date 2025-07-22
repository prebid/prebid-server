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
	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/spf13/viper"
)

// Configuration specifies the static application config.
type Configuration struct {
	ExternalURL      string      `mapstructure:"external_url"`
	Host             string      `mapstructure:"host"`
	Port             int         `mapstructure:"port"`
	UnixSocketEnable bool        `mapstructure:"unix_socket_enable"`
	UnixSocketName   string      `mapstructure:"unix_socket_name"`
	Client           HTTPClient  `mapstructure:"http_client"`
	CacheClient      HTTPClient  `mapstructure:"http_client_cache"`
	Admin            Admin       `mapstructure:"admin"`
	AdminPort        int         `mapstructure:"admin_port"`
	Compression      Compression `mapstructure:"compression"`
	// GarbageCollectorThreshold allocates virtual memory (in bytes) which is not used by PBS but
	// serves as a hack to trigger the garbage collector only when the heap reaches at least this size.
	// More info: https://github.com/golang/go/issues/48409
	GarbageCollectorThreshold int `mapstructure:"garbage_collector_threshold"`
	// StatusResponse is the string which will be returned by the /status endpoint when things are OK.
	// If empty, it will return a 204 with no content.
	StatusResponse    string          `mapstructure:"status_response"`
	AuctionTimeouts   AuctionTimeouts `mapstructure:"auction_timeouts_ms"`
	TmaxAdjustments   TmaxAdjustments `mapstructure:"tmax_adjustments"`
	TmaxDefault       int             `mapstructure:"tmax_default"`
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
	// StoredRequestsTimeout defines the number of milliseconds before a timeout occurs with stored requests fetch
	StoredRequestsTimeout int `mapstructure:"stored_requests_timeout_ms"`

	MaxRequestSize       int64             `mapstructure:"max_request_size"`
	Analytics            Analytics         `mapstructure:"analytics"`
	AMPTimeoutAdjustment int64             `mapstructure:"amp_timeout_adjustment_ms"`
	GDPR                 GDPR              `mapstructure:"gdpr"`
	CCPA                 CCPA              `mapstructure:"ccpa"`
	LMT                  LMT               `mapstructure:"lmt"`
	CurrencyConverter    CurrencyConverter `mapstructure:"currency_converter"`
	DefReqConfig         DefReqConfig      `mapstructure:"default_request"`

	VideoStoredRequestRequired bool `mapstructure:"video_stored_request_required"`

	// Array of blocked apps that is used to create the hash table BlockedAppsLookup so App.ID's can be instantly accessed.
	BlockedApps       []string `mapstructure:"blocked_apps,flow"`
	BlockedAppsLookup map[string]bool
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
	GenerateRequestID bool                      `mapstructure:"generate_request_id"`
	HostSChainNode    *openrtb2.SupplyChainNode `mapstructure:"host_schain_node"`
	// Experiment configures non-production ready features.
	Experiment Experiment `mapstructure:"experiment"`
	DataCenter string     `mapstructure:"datacenter"`
	// BidderInfos supports adapter overrides in extra configs like pbs.json, pbs.yaml, etc.
	// Refers to main.go `configFileName` constant
	BidderInfos BidderInfos `mapstructure:"adapters"`
	// Hooks provides a way to specify hook execution plan for specific endpoints and stages
	Hooks       Hooks       `mapstructure:"hooks"`
	Validations Validations `mapstructure:"validations"`
	PriceFloors PriceFloors `mapstructure:"price_floors"`
}

type Admin struct {
	Enabled bool `mapstructure:"enabled"`
}
type PriceFloors struct {
	Enabled bool              `mapstructure:"enabled"`
	Fetcher PriceFloorFetcher `mapstructure:"fetcher"`
}

type PriceFloorFetcher struct {
	HttpClient HTTPClient `mapstructure:"http_client"`
	CacheSize  int        `mapstructure:"cache_size_mb"`
	Worker     int        `mapstructure:"worker"`
	Capacity   int        `mapstructure:"capacity"`
	MaxRetries int        `mapstructure:"max_retries"`
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
	if cfg.StoredRequestsTimeout <= 0 {
		errs = append(errs, fmt.Errorf("cfg.stored_requests_timeout_ms must be > 0. Got %d", cfg.StoredRequestsTimeout))
	}
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
	errs = cfg.Debug.validate(errs)
	errs = cfg.ExtCacheURL.validate(errs)
	errs = cfg.AccountDefaults.PriceFloors.validate(errs)
	if cfg.AccountDefaults.Disabled {
		glog.Warning(`With account_defaults.disabled=true, host-defined accounts must exist and have "disabled":false. All other requests will be rejected.`)
	}

	if cfg.AccountDefaults.Events.Enabled {
		glog.Warning(`account_defaults.events has no effect as the feature is under development.`)
	}

	errs = cfg.Experiment.validate(errs)
	errs = cfg.BidderInfos.validate(errs)
	errs = cfg.AccountDefaults.Privacy.IPv6Config.Validate(errs)
	errs = cfg.AccountDefaults.Privacy.IPv4Config.Validate(errs)

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
		return append(errs, errors.New("external cache Host and Path must both be specified"))
	}
	if strings.HasSuffix(data.Host, "/") {
		return append(errs, fmt.Errorf("external cache Host '%s' must not end with a path separator", data.Host))
	}
	if strings.Contains(data.Host, "://") {
		return append(errs, fmt.Errorf("external cache Host must not specify a protocol. '%s'", data.Host))
	}
	if !strings.HasPrefix(data.Path, "/") {
		return append(errs, fmt.Errorf("external cache Path '%s' must begin with a path separator", data.Path))
	}

	urlObj, err := url.Parse("https://" + data.Host + data.Path)
	if err != nil {
		return append(errs, fmt.Errorf("external cache Path validation error: %s ", err.Error()))
	}
	if urlObj.Host != data.Host {
		return append(errs, fmt.Errorf("external cache Host '%s' is invalid", data.Host))
	}
	if urlObj.Path != data.Path {
		return append(errs, fmt.Errorf("external cache Path is invalid"))
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
	if cfg.AMPException {
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
		enforceAlgoValue := purposeConfigs[i].EnforceAlgo
		enforceAlgoField := fmt.Sprintf("gdpr.tcf2.purpose%d.enforce_algo", (i + 1))

		if enforceAlgoValue != TCF2EnforceAlgoFull && enforceAlgoValue != TCF2EnforceAlgoBasic {
			errs = append(errs, fmt.Errorf("%s must be \"basic\" or \"full\". Got %s", enforceAlgoField, enforceAlgoValue))
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
	TCF2EnforceAlgoBasic = "basic"
	TCF2EnforceAlgoFull  = "full"
)

type TCF2EnforcementAlgo int

const (
	TCF2UndefinedEnforcement TCF2EnforcementAlgo = iota
	TCF2BasicEnforcement
	TCF2FullEnforcement
)

// TCF2 defines the TCF2 specific configurations for GDPR
type TCF2 struct {
	Enabled   bool        `mapstructure:"enabled"`
	Purpose1  TCF2Purpose `mapstructure:"purpose1"`
	Purpose2  TCF2Purpose `mapstructure:"purpose2"`
	Purpose3  TCF2Purpose `mapstructure:"purpose3"`
	Purpose4  TCF2Purpose `mapstructure:"purpose4"`
	Purpose5  TCF2Purpose `mapstructure:"purpose5"`
	Purpose6  TCF2Purpose `mapstructure:"purpose6"`
	Purpose7  TCF2Purpose `mapstructure:"purpose7"`
	Purpose8  TCF2Purpose `mapstructure:"purpose8"`
	Purpose9  TCF2Purpose `mapstructure:"purpose9"`
	Purpose10 TCF2Purpose `mapstructure:"purpose10"`
	// Map of purpose configs for easy purpose lookup
	PurposeConfigs      map[consentconstants.Purpose]*TCF2Purpose
	SpecialFeature1     TCF2SpecialFeature      `mapstructure:"special_feature1"`
	PurposeOneTreatment TCF2PurposeOneTreatment `mapstructure:"purpose_one_treatment"`
}

// ChannelEnabled checks if a given channel type is enabled. All channel types are considered either
// enabled or disabled based on the Enabled flag.
func (t *TCF2) ChannelEnabled(channelType ChannelType) bool {
	return t.Enabled
}

// IsEnabled indicates if TCF2 is enabled
func (t *TCF2) IsEnabled() bool {
	return t.Enabled
}

// PurposeEnforced checks if full enforcement is turned on for a given purpose. With full enforcement enabled, the
// GDPR full enforcement algorithm will execute for that purpose determining legal basis; otherwise it's skipped.
func (t *TCF2) PurposeEnforced(purpose consentconstants.Purpose) (enforce bool) {
	if t.PurposeConfigs[purpose] == nil {
		return false
	}
	return t.PurposeConfigs[purpose].EnforcePurpose
}

// PurposeEnforcementAlgo returns the default enforcement algorithm for a given purpose
func (t *TCF2) PurposeEnforcementAlgo(purpose consentconstants.Purpose) (enforcement TCF2EnforcementAlgo) {
	if c, exists := t.PurposeConfigs[purpose]; exists {
		return c.EnforceAlgoID
	}
	return TCF2FullEnforcement
}

// PurposeEnforcingVendors checks if enforcing vendors is turned on for a given purpose. With enforcing vendors
// enabled, the GDPR full enforcement algorithm considers the GVL when determining legal basis; otherwise it's skipped.
func (t *TCF2) PurposeEnforcingVendors(purpose consentconstants.Purpose) (enforce bool) {
	if t.PurposeConfigs[purpose] == nil {
		return false
	}
	return t.PurposeConfigs[purpose].EnforceVendors
}

// PurposeVendorExceptions returns the vendor exception map for a given purpose if it exists, otherwise it returns
// an empty map of vendor exceptions
func (t *TCF2) PurposeVendorExceptions(purpose consentconstants.Purpose) (vendorExceptions map[string]struct{}) {
	c, exists := t.PurposeConfigs[purpose]

	if exists && c.VendorExceptionMap != nil {
		return c.VendorExceptionMap
	}
	return make(map[string]struct{}, 0)
}

// FeatureOneEnforced checks if special feature one is enforced. If it is enforced, PBS will determine whether geo
// information may be passed through in the bid request.
func (t *TCF2) FeatureOneEnforced() bool {
	return t.SpecialFeature1.Enforce
}

// FeatureOneVendorException checks if the specified bidder is considered a vendor exception for special feature one.
// If a bidder is a vendor exception, PBS will bypass the pass geo calculation passing the geo information in the bid request.
func (t *TCF2) FeatureOneVendorException(bidder openrtb_ext.BidderName) bool {
	if _, ok := t.SpecialFeature1.VendorExceptionMap[bidder]; ok {
		return true
	}
	return false
}

// PurposeOneTreatmentEnabled checks if purpose one treatment is enabled.
func (t *TCF2) PurposeOneTreatmentEnabled() bool {
	return t.PurposeOneTreatment.Enabled
}

// PurposeOneTreatmentAccessAllowed checks if purpose one treatment access is allowed.
func (t *TCF2) PurposeOneTreatmentAccessAllowed() bool {
	return t.PurposeOneTreatment.AccessAllowed
}

// Making a purpose struct so purpose specific details can be added later.
type TCF2Purpose struct {
	EnforceAlgo string `mapstructure:"enforce_algo"`
	// Integer representation of enforcement algo for performance improvement on compares
	EnforceAlgoID  TCF2EnforcementAlgo
	EnforcePurpose bool `mapstructure:"enforce_purpose"`
	EnforceVendors bool `mapstructure:"enforce_vendors"`
	// Array of vendor exceptions that is used to create the hash table VendorExceptionMap so vendor names can be instantly accessed
	VendorExceptions   []string `mapstructure:"vendor_exceptions"`
	VendorExceptionMap map[string]struct{}
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
	File     FileLogs      `mapstructure:"file"`
	Agma     AgmaAnalytics `mapstructure:"agma"`
	Pubstack Pubstack      `mapstructure:"pubstack"`
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

type AgmaAnalytics struct {
	Enabled  bool                      `mapstructure:"enabled"`
	Endpoint AgmaAnalyticsHttpEndpoint `mapstructure:"endpoint"`
	Buffers  AgmaAnalyticsBuffer       `mapstructure:"buffers"`
	Accounts []AgmaAnalyticsAccount    `mapstructure:"accounts"`
}

type AgmaAnalyticsHttpEndpoint struct {
	Url     string `mapstructure:"url"`
	Timeout string `mapstructure:"timeout"`
	Gzip    bool   `mapstructure:"gzip"`
}

type AgmaAnalyticsBuffer struct {
	BufferSize string `mapstructure:"size"`
	EventCount int    `mapstructure:"count"`
	Timeout    string `mapstructure:"timeout"`
}

type AgmaAnalyticsAccount struct {
	Code        string `mapstructure:"code"`
	PublisherId string `mapstructure:"publisher_id"`
	SiteAppId   string `mapstructure:"site_app_id"`
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

	// True if we want to stop collecting account debug request metrics
	AccountDebug bool `mapstructure:"account_debug"`

	// True if we want to stop collecting account stored respponses metrics
	AccountStoredResponses bool `mapstructure:"account_stored_responses"`

	// True if we don't want to collect metrics about the connections prebid
	// server establishes with bidder servers such as the number of connections
	// that were created or reused.
	AdapterConnectionMetrics bool `mapstructure:"adapter_connections_metrics"`

	// True if we don't want to collect the per adapter buyer UID scrubbed metric
	AdapterBuyerUIDScrubbed bool `mapstructure:"adapter_buyeruid_scrubbed"`

	// True if we don't want to collect the per adapter GDPR request blocked metric
	AdapterGDPRRequestBlocked bool `mapstructure:"adapter_gdpr_request_blocked"`

	// True if we want to stop collecting account modules metrics
	AccountModulesMetrics bool `mapstructure:"account_modules_metrics"`
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

type Server struct {
	ExternalUrl string
	GvlID       int
	DataCenter  string
}

func (server *Server) Empty() bool {
	return server == nil || (server.DataCenter == "" && server.ExternalUrl == "" && server.GvlID == 0)
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

type Validations struct {
	BannerCreativeMaxSize string `mapstructure:"banner_creative_max_size" json:"banner_creative_max_size"`
	SecureMarkup          string `mapstructure:"secure_markup" json:"secure_markup"`
	MaxCreativeWidth      int64  `mapstructure:"max_creative_width" json:"max_creative_width"`
	MaxCreativeHeight     int64  `mapstructure:"max_creative_height" json:"max_creative_height"`
}

const (
	ValidationEnforce string = "enforce"
	ValidationWarn    string = "warn"
	ValidationSkip    string = "skip"
)

func (host *Validations) SetBannerCreativeMaxSize(account Validations) {
	if len(account.BannerCreativeMaxSize) > 0 {
		host.BannerCreativeMaxSize = account.BannerCreativeMaxSize
	}
}

func (cfg *TimeoutNotification) validate(errs []error) []error {
	if cfg.SamplingRate < 0.0 || cfg.SamplingRate > 1.0 {
		errs = append(errs, fmt.Errorf("debug.timeout_notification.sampling_rate must be positive and not greater than 1.0. Got %f", cfg.SamplingRate))
	}
	return errs
}

// New uses viper to get our server configurations.
func New(v *viper.Viper, bidderInfos BidderInfos, normalizeBidderName openrtb_ext.BidderNameNormalizer) (*Configuration, error) {
	var c Configuration
	if err := v.Unmarshal(&c, viper.DecodeHook(AccountModulesHookFunc())); err != nil {
		return nil, fmt.Errorf("viper failed to unmarshal app config: %v", err)
	}

	if err := c.RequestValidation.Parse(); err != nil {
		return nil, err
	}

	if err := isValidCookieSize(c.HostCookie.MaxCookieSizeBytes); err != nil {
		glog.Fatal(fmt.Printf("Max cookie size %d cannot be less than %d \n", c.HostCookie.MaxCookieSizeBytes, MIN_COOKIE_SIZE_BYTES))
		return nil, err
	}

	if err := UnpackDSADefault(c.AccountDefaults.Privacy.DSA); err != nil {
		return nil, fmt.Errorf("invalid default account DSA: %v", err)
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

	// for each purpose we capture a reference to the purpose config in a map for easy purpose config lookup
	c.GDPR.TCF2.PurposeConfigs = map[consentconstants.Purpose]*TCF2Purpose{
		1:  &c.GDPR.TCF2.Purpose1,
		2:  &c.GDPR.TCF2.Purpose2,
		3:  &c.GDPR.TCF2.Purpose3,
		4:  &c.GDPR.TCF2.Purpose4,
		5:  &c.GDPR.TCF2.Purpose5,
		6:  &c.GDPR.TCF2.Purpose6,
		7:  &c.GDPR.TCF2.Purpose7,
		8:  &c.GDPR.TCF2.Purpose8,
		9:  &c.GDPR.TCF2.Purpose9,
		10: &c.GDPR.TCF2.Purpose10,
	}

	// As an alternative to performing several string compares per request, we set the integer representation of
	// the enforcement algorithm on each purpose config
	for _, pc := range c.GDPR.TCF2.PurposeConfigs {
		if pc.EnforceAlgo == TCF2EnforceAlgoBasic {
			pc.EnforceAlgoID = TCF2BasicEnforcement
		} else {
			pc.EnforceAlgoID = TCF2FullEnforcement
		}
	}

	// To look for a purpose's vendor exceptions in O(1) time, for each purpose we fill this hash table with bidders/analytics
	// adapters located in the VendorExceptions field of the GDPR.TCF2.PurposeX struct defined in this file
	for _, pc := range c.GDPR.TCF2.PurposeConfigs {
		pc.VendorExceptionMap = make(map[string]struct{})
		for v := 0; v < len(pc.VendorExceptions); v++ {
			adapterName := pc.VendorExceptions[v]
			pc.VendorExceptionMap[adapterName] = struct{}{}
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
	// the BlockedApps field of the Configuration struct defined in this file
	c.BlockedAppsLookup = make(map[string]bool)
	for i := 0; i < len(c.BlockedApps); i++ {
		c.BlockedAppsLookup[c.BlockedApps[i]] = true
	}

	// Migrate combo stored request config to separate stored_reqs and amp stored_reqs configs.
	resolvedStoredRequestsConfig(&c)

	configBidderInfosWithNillableFields, err := setConfigBidderInfoNillableFields(v, c.BidderInfos)
	if err != nil {
		return nil, err
	}
	mergedBidderInfos, err := applyBidderInfoConfigOverrides(configBidderInfosWithNillableFields, bidderInfos, normalizeBidderName)
	if err != nil {
		return nil, err
	}
	c.BidderInfos = mergedBidderInfos

	glog.Info("Logging the resolved configuration:")
	logGeneral(reflect.ValueOf(c), "  \t")
	if errs := c.validate(v); len(errs) > 0 {
		return &c, errortypes.NewAggregateError("validation errors", errs)
	}

	return &c, nil
}

type bidderInfoNillableFields struct {
	Disabled                *bool `yaml:"disabled" mapstructure:"disabled"`
	ModifyingVastXmlAllowed *bool `yaml:"modifyingVastXmlAllowed" mapstructure:"modifyingVastXmlAllowed"`
}
type nillableFieldBidderInfos map[string]nillableFieldBidderInfo
type nillableFieldBidderInfo struct {
	nillableFields bidderInfoNillableFields
	bidderInfo     BidderInfo
}

func setConfigBidderInfoNillableFields(v *viper.Viper, bidderInfos BidderInfos) (nillableFieldBidderInfos, error) {
	if len(bidderInfos) == 0 || v == nil {
		return nil, nil
	}
	infos := make(nillableFieldBidderInfos, len(bidderInfos))

	for bidderName, bidderInfo := range bidderInfos {
		info := nillableFieldBidderInfo{bidderInfo: bidderInfo}

		if err := v.UnmarshalKey("adapters."+bidderName+".disabled", &info.nillableFields.Disabled); err != nil {
			return nil, fmt.Errorf("viper failed to unmarshal bidder config disabled: %v", err)
		}
		if err := v.UnmarshalKey("adapters."+bidderName+".modifyingvastxmlallowed", &info.nillableFields.ModifyingVastXmlAllowed); err != nil {
			return nil, fmt.Errorf("viper failed to unmarshal bidder config modifyingvastxmlallowed: %v", err)
		}
		infos[bidderName] = info
	}
	return infos, nil
}

// MarshalAccountDefaults compiles AccountDefaults into the JSON format used for merge patch
func (cfg *Configuration) MarshalAccountDefaults() error {
	var err error
	if cfg.accountDefaultsJSON, err = jsonutil.Marshal(cfg.AccountDefaults); err != nil {
		glog.Warningf("converting %+v to json: %v", cfg.AccountDefaults, err)
	}
	return err
}

// UnpackDSADefault validates the JSON DSA default object string by unmarshaling and maps it to a struct
func UnpackDSADefault(dsa *AccountDSA) error {
	if dsa == nil || len(dsa.Default) == 0 {
		return nil
	}
	return jsonutil.Unmarshal([]byte(dsa.Default), &dsa.DefaultUnpacked)
}

// AccountDefaultsJSON returns the precompiled JSON form of account_defaults
func (cfg *Configuration) AccountDefaultsJSON() json.RawMessage {
	return cfg.accountDefaultsJSON
}

// GetBaseURL allows for protocol relative URL if scheme is empty
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
func SetupViper(v *viper.Viper, filename string, bidderInfos BidderInfos) {
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
	v.SetDefault("unix_socket_enable", false)              // boolean which decide if the socket-server will be started.
	v.SetDefault("unix_socket_name", "prebid-server.sock") // path of the socket's file which must be listened.
	v.SetDefault("admin_port", 6060)
	v.SetDefault("admin.enabled", true) // boolean to determine if admin listener will be started.
	v.SetDefault("garbage_collector_threshold", 0)
	v.SetDefault("status_response", "")
	v.SetDefault("datacenter", "")
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
	v.SetDefault("host_schain_node", nil)
	v.SetDefault("validations.banner_creative_max_size", ValidationSkip)
	v.SetDefault("validations.secure_markup", ValidationSkip)
	v.SetDefault("validations.max_creative_size.height", 0)
	v.SetDefault("validations.max_creative_size.width", 0)
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
	v.SetDefault("metrics.disabled_metrics.account_debug", true)
	v.SetDefault("metrics.disabled_metrics.account_stored_responses", true)
	v.SetDefault("metrics.disabled_metrics.adapter_connections_metrics", true)
	v.SetDefault("metrics.disabled_metrics.adapter_buyeruid_scrubbed", true)
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
	v.SetDefault("stored_requests_timeout_ms", 50)
	v.SetDefault("stored_requests.database.connection.driver", "")
	v.SetDefault("stored_requests.database.connection.dbname", "")
	v.SetDefault("stored_requests.database.connection.host", "")
	v.SetDefault("stored_requests.database.connection.port", 0)
	v.SetDefault("stored_requests.database.connection.user", "")
	v.SetDefault("stored_requests.database.connection.password", "")
	v.SetDefault("stored_requests.database.connection.query_string", "")
	v.SetDefault("stored_requests.database.connection.tls.root_cert", "")
	v.SetDefault("stored_requests.database.connection.tls.client_cert", "")
	v.SetDefault("stored_requests.database.connection.tls.client_key", "")
	v.SetDefault("stored_requests.database.fetcher.query", "")
	v.SetDefault("stored_requests.database.fetcher.amp_query", "")
	v.SetDefault("stored_requests.database.initialize_caches.timeout_ms", 0)
	v.SetDefault("stored_requests.database.initialize_caches.query", "")
	v.SetDefault("stored_requests.database.initialize_caches.amp_query", "")
	v.SetDefault("stored_requests.database.poll_for_updates.refresh_rate_seconds", 0)
	v.SetDefault("stored_requests.database.poll_for_updates.timeout_ms", 0)
	v.SetDefault("stored_requests.database.poll_for_updates.query", "")
	v.SetDefault("stored_requests.database.poll_for_updates.amp_query", "")
	v.SetDefault("stored_requests.filesystem.enabled", false)
	v.SetDefault("stored_requests.filesystem.directorypath", "./stored_requests/data/by_id")
	v.SetDefault("stored_requests.directorypath", "./stored_requests/data/by_id")
	v.SetDefault("stored_requests.http.endpoint", "")
	v.SetDefault("stored_requests.http.amp_endpoint", "")
	v.SetDefault("stored_requests.http.use_rfc3986_compliant_request_builder", false)
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
	v.SetDefault("stored_video_req.database.connection.driver", "")
	v.SetDefault("stored_video_req.database.connection.dbname", "")
	v.SetDefault("stored_video_req.database.connection.host", "")
	v.SetDefault("stored_video_req.database.connection.port", 0)
	v.SetDefault("stored_video_req.database.connection.user", "")
	v.SetDefault("stored_video_req.database.connection.password", "")
	v.SetDefault("stored_video_req.database.connection.query_string", "")
	v.SetDefault("stored_video_req.database.connection.tls.root_cert", "")
	v.SetDefault("stored_video_req.database.connection.tls.client_cert", "")
	v.SetDefault("stored_video_req.database.connection.tls.client_key", "")
	v.SetDefault("stored_video_req.database.fetcher.query", "")
	v.SetDefault("stored_video_req.database.fetcher.amp_query", "")
	v.SetDefault("stored_video_req.database.initialize_caches.timeout_ms", 0)
	v.SetDefault("stored_video_req.database.initialize_caches.query", "")
	v.SetDefault("stored_video_req.database.initialize_caches.amp_query", "")
	v.SetDefault("stored_video_req.database.poll_for_updates.refresh_rate_seconds", 0)
	v.SetDefault("stored_video_req.database.poll_for_updates.timeout_ms", 0)
	v.SetDefault("stored_video_req.database.poll_for_updates.query", "")
	v.SetDefault("stored_video_req.database.poll_for_updates.amp_query", "")
	v.SetDefault("stored_video_req.filesystem.enabled", false)
	v.SetDefault("stored_video_req.filesystem.directorypath", "")
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
	v.SetDefault("stored_responses.database.connection.driver", "")
	v.SetDefault("stored_responses.database.connection.dbname", "")
	v.SetDefault("stored_responses.database.connection.host", "")
	v.SetDefault("stored_responses.database.connection.port", 0)
	v.SetDefault("stored_responses.database.connection.user", "")
	v.SetDefault("stored_responses.database.connection.password", "")
	v.SetDefault("stored_responses.database.connection.query_string", "")
	v.SetDefault("stored_responses.database.connection.tls.root_cert", "")
	v.SetDefault("stored_responses.database.connection.tls.client_cert", "")
	v.SetDefault("stored_responses.database.connection.tls.client_key", "")
	v.SetDefault("stored_responses.database.fetcher.query", "")
	v.SetDefault("stored_responses.database.fetcher.amp_query", "")
	v.SetDefault("stored_responses.database.initialize_caches.timeout_ms", 0)
	v.SetDefault("stored_responses.database.initialize_caches.query", "")
	v.SetDefault("stored_responses.database.initialize_caches.amp_query", "")
	v.SetDefault("stored_responses.database.poll_for_updates.refresh_rate_seconds", 0)
	v.SetDefault("stored_responses.database.poll_for_updates.timeout_ms", 0)
	v.SetDefault("stored_responses.database.poll_for_updates.query", "")
	v.SetDefault("stored_responses.database.poll_for_updates.amp_query", "")
	v.SetDefault("stored_responses.filesystem.enabled", false)
	v.SetDefault("stored_responses.filesystem.directorypath", "")
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

	v.SetDefault("user_sync.priority_groups", [][]string{})

	v.SetDefault("accounts.filesystem.enabled", false)
	v.SetDefault("accounts.filesystem.directorypath", "./stored_requests/data/by_id")
	v.SetDefault("accounts.in_memory_cache.type", "none")

	v.BindEnv("user_sync.external_url")
	v.BindEnv("user_sync.coop_sync.default")

	// some adapters append the user id to the end of the redirect url instead of using
	// macro substitution. it is important for the uid to be the last query parameter.
	v.SetDefault("user_sync.redirect_url", "{{.ExternalURL}}/setuid?bidder={{.SyncerKey}}&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&gpp={{.GPP}}&gpp_sid={{.GPPSID}}&f={{.SyncType}}&uid={{.UserMacro}}")

	v.SetDefault("max_request_size", 1024*256)
	v.SetDefault("analytics.file.filename", "")
	v.SetDefault("analytics.pubstack.endpoint", "https://s2s.pbstck.com/v1")
	v.SetDefault("analytics.pubstack.scopeid", "change-me")
	v.SetDefault("analytics.pubstack.enabled", false)
	v.SetDefault("analytics.pubstack.configuration_refresh_delay", "2h")
	v.SetDefault("analytics.pubstack.buffers.size", "2MB")
	v.SetDefault("analytics.pubstack.buffers.count", 100)
	v.SetDefault("analytics.pubstack.buffers.timeout", "900s")
	v.SetDefault("analytics.agma.enabled", false)
	v.SetDefault("analytics.agma.endpoint.url", "https://go.pbs.agma-analytics.de/v1/prebid-server")
	v.SetDefault("analytics.agma.endpoint.timeout", "2s")
	v.SetDefault("analytics.agma.endpoint.gzip", false)
	v.SetDefault("analytics.agma.buffers.size", "2MB")
	v.SetDefault("analytics.agma.buffers.count", 100)
	v.SetDefault("analytics.agma.buffers.timeout", "15m")
	v.SetDefault("analytics.agma.accounts", []AgmaAnalyticsAccount{})
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
	v.SetDefault("gdpr.tcf2.purpose1.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose2.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose3.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose4.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose5.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose6.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose7.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose8.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose9.vendor_exceptions", []string{})
	v.SetDefault("gdpr.tcf2.purpose10.vendor_exceptions", []string{})
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
	v.SetDefault("blocked_apps", []string{""})
	v.SetDefault("account_required", false)
	v.SetDefault("account_defaults.disabled", false)
	v.SetDefault("account_defaults.debug_allow", true)
	v.SetDefault("account_defaults.price_floors.enabled", false)
	v.SetDefault("account_defaults.price_floors.enforce_floors_rate", 100)
	v.SetDefault("account_defaults.price_floors.adjust_for_bid_adjustment", true)
	v.SetDefault("account_defaults.price_floors.enforce_deal_floors", false)
	v.SetDefault("account_defaults.price_floors.use_dynamic_data", false)
	v.SetDefault("account_defaults.price_floors.max_rules", 100)
	v.SetDefault("account_defaults.price_floors.max_schema_dims", 3)
	v.SetDefault("account_defaults.price_floors.fetch.enabled", false)
	v.SetDefault("account_defaults.price_floors.fetch.url", "")
	v.SetDefault("account_defaults.price_floors.fetch.timeout_ms", 3000)
	v.SetDefault("account_defaults.price_floors.fetch.max_file_size_kb", 100)
	v.SetDefault("account_defaults.price_floors.fetch.max_rules", 1000)
	v.SetDefault("account_defaults.price_floors.fetch.max_age_sec", 86400)
	v.SetDefault("account_defaults.price_floors.fetch.period_sec", 3600)
	v.SetDefault("account_defaults.price_floors.fetch.max_schema_dims", 0)
	v.SetDefault("account_defaults.privacy.privacysandbox.topicsdomain", "")
	v.SetDefault("account_defaults.privacy.privacysandbox.cookiedeprecation.enabled", false)
	v.SetDefault("account_defaults.privacy.privacysandbox.cookiedeprecation.ttl_sec", 604800)

	v.SetDefault("account_defaults.events_enabled", false)
	v.BindEnv("account_defaults.privacy.dsa.default")
	v.BindEnv("account_defaults.privacy.dsa.gdpr_only")
	v.SetDefault("account_defaults.privacy.ipv6.anon_keep_bits", 56)
	v.SetDefault("account_defaults.privacy.ipv4.anon_keep_bits", 24)

	//Defaults for Price floor fetcher
	v.SetDefault("price_floors.fetcher.worker", 20)
	v.SetDefault("price_floors.fetcher.capacity", 20000)
	v.SetDefault("price_floors.fetcher.cache_size_mb", 64)
	v.SetDefault("price_floors.fetcher.http_client.max_connections_per_host", 0) // unlimited
	v.SetDefault("price_floors.fetcher.http_client.max_idle_connections", 40)
	v.SetDefault("price_floors.fetcher.http_client.max_idle_connections_per_host", 2)
	v.SetDefault("price_floors.fetcher.http_client.idle_connection_timeout_seconds", 60)
	v.SetDefault("price_floors.fetcher.max_retries", 10)

	v.SetDefault("account_defaults.events_enabled", false)
	v.SetDefault("compression.response.enable_gzip", false)
	v.SetDefault("compression.request.enable_gzip", false)

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

	v.SetDefault("tmax_adjustments.enabled", false)
	v.SetDefault("tmax_adjustments.bidder_response_duration_min_ms", 0)
	v.SetDefault("tmax_adjustments.bidder_network_latency_buffer_ms", 0)
	v.SetDefault("tmax_adjustments.pbs_response_preparation_duration_ms", 0)

	v.SetDefault("tmax_default", 0)

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

	// These defaults must be set after the migrate functions because those functions look for the presence of these
	// config fields and there isn't a way to detect presence of a config field using the viper package if a default
	// is set. Viper IsSet and Get functions consider default values.
	v.SetDefault("gdpr.tcf2.purpose1.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose2.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose3.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose4.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose5.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose6.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose7.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose8.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose9.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose10.enforce_algo", TCF2EnforceAlgoFull)
	v.SetDefault("gdpr.tcf2.purpose1.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose2.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose3.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose4.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose5.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose6.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose7.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose8.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose9.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose10.enforce_purpose", true)
	v.SetDefault("gdpr.tcf2.purpose_one_treatment.enabled", true)
	v.SetDefault("gdpr.tcf2.purpose_one_treatment.access_allowed", true)
	v.SetDefault("gdpr.tcf2.special_feature1.enforce", true)
	v.SetDefault("gdpr.tcf2.special_feature1.vendor_exceptions", []openrtb_ext.BidderName{})
	v.SetDefault("price_floors.enabled", false)

	// Defaults for account_defaults.events.default_url
	v.SetDefault("account_defaults.events.default_url", "https://PBS_HOST/event?t=##PBS-EVENTTYPE##&vtype=##PBS-VASTEVENT##&b=##PBS-BIDID##&f=i&a=##PBS-ACCOUNTID##&ts=##PBS-TIMESTAMP##&bidder=##PBS-BIDDER##&int=##PBS-INTEGRATION##&mt=##PBS-MEDIATYPE##&ch=##PBS-CHANNEL##&aid=##PBS-AUCTIONID##&l=##PBS-LINEID##")
	v.SetDefault("account_defaults.events.enabled", false)

	v.SetDefault("experiment.adscert.mode", "off")
	v.SetDefault("experiment.adscert.inprocess.origin", "")
	v.SetDefault("experiment.adscert.inprocess.key", "")
	v.SetDefault("experiment.adscert.inprocess.domain_check_interval_seconds", 30)
	v.SetDefault("experiment.adscert.inprocess.domain_renewal_interval_seconds", 30)
	v.SetDefault("experiment.adscert.remote.url", "")
	v.SetDefault("experiment.adscert.remote.signing_timeout_ms", 5)

	v.SetDefault("hooks.enabled", false)

	for bidderName := range bidderInfos {
		setBidderDefaults(v, strings.ToLower(bidderName))
	}
}

func isConfigInfoPresent(v *viper.Viper, prefix string, fields []string) bool {
	prefix = prefix + "."
	for _, field := range fields {
		fieldName := prefix + field
		if v.IsSet(fieldName) {
			return true
		}
	}
	return false
}

func setBidderDefaults(v *viper.Viper, bidder string) {
	adapterCfgPrefix := "adapters." + bidder
	v.BindEnv(adapterCfgPrefix + ".disabled")
	v.BindEnv(adapterCfgPrefix + ".endpoint")
	v.BindEnv(adapterCfgPrefix + ".extra_info")
	v.BindEnv(adapterCfgPrefix + ".modifyingVastXmlAllowed")
	v.BindEnv(adapterCfgPrefix + ".debug.allow")
	v.BindEnv(adapterCfgPrefix + ".gvlVendorID")
	v.BindEnv(adapterCfgPrefix + ".usersync_url")
	v.BindEnv(adapterCfgPrefix + ".experiment.adsCert.enabled")
	v.BindEnv(adapterCfgPrefix + ".platform_id")
	v.BindEnv(adapterCfgPrefix + ".app_secret")
	v.BindEnv(adapterCfgPrefix + ".xapi.username")
	v.BindEnv(adapterCfgPrefix + ".xapi.password")
	v.BindEnv(adapterCfgPrefix + ".xapi.tracker")
	v.BindEnv(adapterCfgPrefix + ".endpointCompression")
	v.BindEnv(adapterCfgPrefix + ".openrtb.version")
	v.BindEnv(adapterCfgPrefix + ".openrtb.gpp-supported")

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

// Tmax Adjustments enables PBS to estimate the tmax value for bidders, indicating the allotted time for them to respond to a request.
// It's important to note that the calculated tmax is just an estimate and will not be entirely precise.
// PBS will calculate the bidder tmax as follows:
// bidderTmax = request.tmax - reqProcessingTime - BidderNetworkLatencyBuffer - PBSResponsePreparationDuration
// Note that reqProcessingTime is time taken by PBS to process a given request before it is sent to bid adapters and is computed at run time.
type TmaxAdjustments struct {
	// Enabled indicates whether bidder tmax should be calculated and passed on to bid adapters
	Enabled bool `mapstructure:"enabled"`
	// BidderNetworkLatencyBuffer accounts for network delays between PBS and bidder servers.
	// A value of 0 indicates no network latency buffer should be accounted for when calculating the bidder tmax.
	BidderNetworkLatencyBuffer uint `mapstructure:"bidder_network_latency_buffer_ms"`
	// PBSResponsePreparationDuration accounts for amount of time required for PBS to process all bidder responses and generate final response for a request.
	// A value of 0 indicates PBS response preparation time shouldn't be accounted for when calculating bidder tmax.
	PBSResponsePreparationDuration uint `mapstructure:"pbs_response_preparation_duration_ms"`
	// BidderResponseDurationMin is the minimum amount of time expected to get a response from a bidder request.
	// PBS won't send a request to the bidder if the bidder tmax calculated is less than the BidderResponseDurationMin value
	BidderResponseDurationMin uint `mapstructure:"bidder_response_duration_min_ms"`
}
