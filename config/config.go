package config

import (
	"bytes"
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

// Configuration
type Configuration struct {
	ExternalURL string     `mapstructure:"external_url"`
	Host        string     `mapstructure:"host"`
	Port        int        `mapstructure:"port"`
	Client      HTTPClient `mapstructure:"http_client"`
	AdminPort   int        `mapstructure:"admin_port"`
	EnableGzip  bool       `mapstructure:"enable_gzip"`
	// StatusResponse is the string which will be returned by the /status endpoint when things are OK.
	// If empty, it will return a 204 with no content.
	StatusResponse  string             `mapstructure:"status_response"`
	AuctionTimeouts AuctionTimeouts    `mapstructure:"auction_timeouts_ms"`
	CacheURL        Cache              `mapstructure:"cache"`
	RecaptchaSecret string             `mapstructure:"recaptcha_secret"`
	HostCookie      HostCookie         `mapstructure:"host_cookie"`
	Metrics         Metrics            `mapstructure:"metrics"`
	DataCache       DataCache          `mapstructure:"datacache"`
	StoredRequests  StoredRequests     `mapstructure:"stored_requests"`
	CategoryMapping StoredRequestsSlim `mapstructure:"category_mapping"`

	// Adapters should have a key for every openrtb_ext.BidderName, converted to lower-case.
	// Se also: https://github.com/spf13/viper/issues/371#issuecomment-335388559
	Adapters             map[string]Adapter `mapstructure:"adapters"`
	MaxRequestSize       int64              `mapstructure:"max_request_size"`
	Analytics            Analytics          `mapstructure:"analytics"`
	AMPTimeoutAdjustment int64              `mapstructure:"amp_timeout_adjustment_ms"`
	GDPR                 GDPR               `mapstructure:"gdpr"`
	CurrencyConverter    CurrencyConverter  `mapstructure:"currency_converter"`
	DefReqConfig         DefReqConfig       `mapstructure:"default_request"`
}

type HTTPClient struct {
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
	errs = cfg.Metrics.validate(errs)
	if cfg.MaxRequestSize < 0 {
		errs = append(errs, fmt.Errorf("cfg.max_request_size must be >= 0. Got %d", cfg.MaxRequestSize))
	}
	errs = cfg.GDPR.validate(errs)
	errs = cfg.CurrencyConverter.validate(errs)
	errs = validateAdapters(cfg.Adapters, errs)
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

type CurrencyConverter struct {
	FetchURL             string `mapstructure:"fetch_url"`
	FetchIntervalSeconds int    `mapstructure:"fetch_interval_seconds"`
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

func (cfg *HostCookie) TTLDuration() time.Duration {
	return time.Duration(cfg.TTL) * time.Hour * 24
}

const (
	dummyHost        string = "dummyhost.com"
	dummyPublisherID int    = 12
	dummyGDPR        string = "0"
	dummyGDPRConsent string = "someGDPRConsentString"
)

type Adapter struct {
	Endpoint string `mapstructure:"endpoint"` // Required
	// UserSyncURL is the URL returned by /cookie_sync for this Bidder. It is _usually_ optional.
	// If not defined, sensible defaults will be derved based on the config.external_url.
	// Note that some Bidders don't have sensible defaults, because their APIs require an ID that will vary
	// from one PBS host to another.
	//
	// For these bidders, there will be a warning logged on startup that usersyncs will not work if you have not
	// defined one in the app config. Check your app logs for more info.
	//
	// This value will be interpreted as a Golang Template. At runtime, the following Template variables will be replaced.
	//
	//   {{.GDPR}} -- This will be replaced with the "gdpr" property sent to /cookie_sync
	//   {{.Consent}} -- This will be replaced with the "consent" property sent to /cookie_sync
	//
	// For more info on templates, see: https://golang.org/pkg/text/template/
	UserSyncURL string `mapstructure:"usersync_url"`
	PlatformID  string `mapstructure:"platform_id"` // needed for Facebook
	XAPI        struct {
		Username string `mapstructure:"username"`
		Password string `mapstructure:"password"`
		Tracker  string `mapstructure:"tracker"`
	} `mapstructure:"xapi"` // needed for Rubicon
	Disabled bool `mapstructure:"disabled"`
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
	resolvedEndpoint, err := macros.ResolveMacros(*endpointTemplate, macros.EndpointTemplateParams{Host: dummyHost, PublisherID: dummyPublisherID})
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
		resolvedUserSyncURL, err := macros.ResolveMacros(*userSyncTemplate, macros.UserSyncTemplateParams{GDPR: dummyGDPR, GDPRConsent: dummyGDPRConsent})
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
}

func (cfg *Metrics) validate(errs configErrors) configErrors {
	return cfg.Prometheus.validate(errs)
}

type InfluxMetrics struct {
	Host     string `mapstructure:"host"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
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

// New uses viper to get our server configurations.
func New(v *viper.Viper) (*Configuration, error) {
	var c Configuration
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("viper failed to unmarshal app config: %v", err)
	}
	c.setDerivedDefaults()
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

// Initialize any default config values which have sensible defaults, but those defaults depend on other config values.
//
// For example, the typical Bidder's usersync URL includes the PBS config.external_url, because it redirects to the `external_url/setuid` endpoint.
//
func (cfg *Configuration) setDerivedDefaults() {
	externalURL := cfg.ExternalURL
	// openrtb_ext.Bidder33Across doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdkernelAdn, "https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3DadkernelAdn%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdtelligent, "https://sync.adtelligent.com/csync?t=p&ep=0&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadtelligent%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%7Buid%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAdform, "https://cm.adform.net/cookie?redirect_url="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadform%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderAppnexus, "https://ib.adnxs.com/getuid?"+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dadnxs%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderBrightroll, "https://pr-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&url="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dbrightroll%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderBeachfront, "https://sync.bfmio.com/sync_s2s?gdpr={{.GDPR}}&url="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dbeachfront%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bio_cid%5D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderConsumable, "https://e.serverbid.com/udb/9969/match?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dconsumable%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderConversant, "https://prebid-match.dotomi.com/prebid/match?rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dconversant%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderEPlanning, "https://ads.us.e-planning.net/getuid/1/5a1ad71d2d53a0f5?"+externalURL+"/setuid?bidder=eplanning&gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&uid=$UID")
	// openrtb_ext.BidderFacebook doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderGrid, "https://grid.bidswitch.net/sp_sync?sp_id=prebid&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dgrid%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderGumGum, "https://rtb.gumgum.com/usync/prbds2s?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dgumgum%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderIx, "https://ssum.casalemedia.com/usermatchredir?s=184932&cb="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dix%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderLifestreet, "https://ads.lfstmedia.com/idsync/137062?synced=1&ttl=1s&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dlifestreet%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%24visitor_cookie%24%24")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderOpenx, "https://rtb.openx.net/sync/prebid?r="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dopenx%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderPubmatic, "https://ads.pubmatic.com/AdServer/js/user_sync.html?predirect="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dpubmatic%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderPulsepoint, "https://bh.contextweb.com/rtset?pid=561205&ev=1&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dpulsepoint%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%25%25VGUID%25%25")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderRhythmone, "https://sync.1rx.io/usersync2/rmphb?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5BRX_UUID%5D")
	// openrtb_ext.BidderRubicon doesn't have a good default.
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSomoaudience, "https://publisher-east.mobileadtrading.com/usersync?ru="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsomoaudience%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24%7BUID%7D")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSovrn, "https://ap.lijit.com/pixel?redir="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsovrn%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderSonobi, "https://sync.go.sonobi.com/us.gif?loc="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dsonobi%26consent_string%3D{{.GDPR}}%26gdpr%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderYieldmo, "https://ads.yieldmo.com/pbsync?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redirectUri="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dyieldmo%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%24UID")
	setDefaultUsersync(cfg.Adapters, openrtb_ext.BidderGamoshi, "https://rtb.gamoshi.io/pix/0000/scm?gdpr={{.GDPR}}&consent={{.GDPRConsent}}&rurl="+url.QueryEscape(externalURL)+"%2Fsetuid%3Fbidder%3Dgamoshi%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5Bgusr%5D")

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
	v.SetDefault("recaptcha_secret", "")
	v.SetDefault("host_cookie.domain", "")
	v.SetDefault("host_cookie.family", "")
	v.SetDefault("host_cookie.cookie_name", "")
	v.SetDefault("host_cookie.opt_out_url", "")
	v.SetDefault("host_cookie.opt_in_url", "")
	v.SetDefault("host_cookie.optout_cookie.name", "")
	v.SetDefault("host_cookie.value", "")
	v.SetDefault("host_cookie.ttl_days", 90)
	v.SetDefault("http_client.max_idle_connections", 400)
	v.SetDefault("http_client.max_idle_connections_per_host", 10)
	v.SetDefault("http_client.idle_connection_timeout_seconds", 60)
	// no metrics configured by default (metrics{host|database|username|password})
	v.SetDefault("metrics.influxdb.host", "")
	v.SetDefault("metrics.influxdb.database", "")
	v.SetDefault("metrics.influxdb.username", "")
	v.SetDefault("metrics.influxdb.password", "")
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
	v.SetDefault("stored_requests.filesystem", false)
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

	for _, bidder := range openrtb_ext.BidderMap {
		setBidderDefaults(v, strings.ToLower(string(bidder)))
	}

	// Disabling adapters by default that require some specific config params.
	// If you're using one of these, make sure you check out the documentation (https://github.com/prebid/prebid-server/tree/master/docs/bidders)
	// for them and specify all the parameters they need for them to work correctly.
	v.SetDefault("adapters.audiencenetwork.disabled", true)
	v.SetDefault("adapters.rubicon.disabled", true)

	v.SetDefault("adapters.adtelligent.endpoint", "http://hb.adtelligent.com/auction")
	v.SetDefault("adapters.adform.endpoint", "http://adx.adform.net/adx")
	v.SetDefault("adapters.appnexus.endpoint", "http://ib.adnxs.com/openrtb2") // Docs: https://wiki.appnexus.com/display/supply/Incoming+Bid+Request+from+SSPs
	v.SetDefault("adapters.beachfront.endpoint", "https://display.bfmio.com/prebid_display")
	v.SetDefault("adapters.beachfront.platform_id", "155")
	v.SetDefault("adapters.brightroll.endpoint", "http://east-bid.ybp.yahoo.com/bid/appnexuspbs")
	v.SetDefault("adapters.consumable.endpoint", "https://e.serverbid.com/api/v2")
	v.SetDefault("adapters.conversant.endpoint", "http://api.hb.ad.cpe.dotomi.com/s2s/header/24")
	v.SetDefault("adapters.eplanning.endpoint", "http://ads.us.e-planning.net/hb/1")
	v.SetDefault("adapters.ix.endpoint", "http://appnexus-us-east.lb.indexww.com/transbidder?p=184932")
	v.SetDefault("adapters.lifestreet.endpoint", "https://prebid.s2s.lfstmedia.com/adrequest")
	v.SetDefault("adapters.openx.endpoint", "http://rtb.openx.net/prebid")
	v.SetDefault("adapters.pubmatic.endpoint", "http://hbopenbid.pubmatic.com/translator?source=prebid-server")
	v.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	v.SetDefault("adapters.rubicon.endpoint", "http://exapi-us-east.rubiconproject.com/a/api/exchange.json")
	v.SetDefault("adapters.somoaudience.endpoint", "http://publisher-east.mobileadtrading.com/rtb/bid")
	v.SetDefault("adapters.sovrn.endpoint", "http://ap.lijit.com/rtb/bid?src=prebid_server")
	v.SetDefault("adapters.adkerneladn.endpoint", "http://{{.Host}}/rtbpub?account={{.PublisherID}}")
	v.SetDefault("adapters.33across.partner_id", "")
	v.SetDefault("adapters.33across.endpoint", "http://ssc.33across.com/api/v1/hb")
	v.SetDefault("adapters.rhythmone.endpoint", "http://tag.1rx.io/rmp")
	v.SetDefault("adapters.gumgum.endpoint", "https://g2.gumgum.com/providers/prbds2s/bid")
	v.SetDefault("adapters.grid.endpoint", "http://grid.bidswitch.net/sp_bid?sp=prebid")
	v.SetDefault("adapters.sonobi.endpoint", "https://apex.go.sonobi.com/bid?partnerid=71d9d3d8af")
	v.SetDefault("adapters.yieldmo.endpoint", "http://ads.yieldmo.com/exchange/prebid-server")
	v.SetDefault("adapters.gamoshi.endpoint", "https://rtb.gamoshi.io")

	v.SetDefault("max_request_size", 1024*256)
	v.SetDefault("analytics.file.filename", "")
	v.SetDefault("amp_timeout_adjustment_ms", 0)
	v.SetDefault("gdpr.host_vendor_id", 0)
	v.SetDefault("gdpr.usersync_if_ambiguous", false)
	v.SetDefault("gdpr.timeouts_ms.init_vendorlist_fetches", 0)
	v.SetDefault("gdpr.timeouts_ms.active_vendorlist_fetch", 0)
	v.SetDefault("currency_converter.fetch_url", "https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json")
	v.SetDefault("currency_converter.fetch_interval_seconds", 0) // #280 Not activated for the time being
	v.SetDefault("default_request.type", "")
	v.SetDefault("default_request.file.name", "")
	v.SetDefault("default_request.alias_info", false)

	// Set environment variable support:
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix("PBS")
	v.AutomaticEnv()
	v.ReadInConfig()
}

func setBidderDefaults(v *viper.Viper, bidder string) {
	v.SetDefault("adapters."+bidder+".endpoint", "")
	v.SetDefault("adapters."+bidder+".usersync_url", "")
	v.SetDefault("adapters."+bidder+".platform_id", "")
	v.SetDefault("adapters."+bidder+".xapi.username", "")
	v.SetDefault("adapters."+bidder+".xapi.password", "")
	v.SetDefault("adapters."+bidder+".xapi.tracker", "")
	v.SetDefault("adapters."+bidder+".disabled", false)
	v.SetDefault("adapters."+bidder+".partner_id", "")
}
