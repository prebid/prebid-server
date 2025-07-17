package config

import (
	"bytes"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

var bidderInfos = BidderInfos{
	"bidder1": BidderInfo{
		Endpoint:   "http://bidder1.com",
		Maintainer: &MaintainerInfo{Email: "maintainer@bidder1.com"},
		Capabilities: &CapabilitiesInfo{
			App: &PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	},
	"bidder2": BidderInfo{
		Endpoint:   "http://bidder2.com",
		Maintainer: &MaintainerInfo{Email: "maintainer@bidder2.com"},
		Capabilities: &CapabilitiesInfo{
			App: &PlatformInfo{
				MediaTypes: []openrtb_ext.BidType{openrtb_ext.BidTypeBanner},
			},
		},
	},
}

func TestExternalCacheURLValidate(t *testing.T) {
	testCases := []struct {
		desc      string
		data      ExternalCache
		expErrors int
	}{
		{
			desc:      "With http://",
			data:      ExternalCache{Host: "http://www.google.com", Path: "/path/v1"},
			expErrors: 1,
		},
		{
			desc:      "Without http://",
			data:      ExternalCache{Host: "www.google.com", Path: "/path/v1"},
			expErrors: 0,
		},
		{
			desc:      "No scheme but '//' prefix",
			data:      ExternalCache{Host: "//www.google.com", Path: "/path/v1"},
			expErrors: 1,
		},
		{
			desc:      "// appears twice",
			data:      ExternalCache{Host: "//www.google.com//", Path: "path/v1"},
			expErrors: 1,
		},
		{
			desc:      "Host has an only // value",
			data:      ExternalCache{Host: "//", Path: "path/v1"},
			expErrors: 1,
		},
		{
			desc:      "only scheme host, valid path",
			data:      ExternalCache{Host: "http://", Path: "/path/v1"},
			expErrors: 1,
		},
		{
			desc:      "No host, path only",
			data:      ExternalCache{Host: "", Path: "path/v1"},
			expErrors: 1,
		},
		{
			desc:      "No host, nor path",
			data:      ExternalCache{Host: "", Path: ""},
			expErrors: 0,
		},
		{
			desc:      "Invalid http at the end",
			data:      ExternalCache{Host: "www.google.com", Path: "http://"},
			expErrors: 1,
		},
		{
			desc:      "Host has an unknown scheme",
			data:      ExternalCache{Host: "unknownscheme://host", Path: "/path/v1"},
			expErrors: 1,
		},
		{
			desc:      "Wrong colon side in scheme",
			data:      ExternalCache{Host: "http//:www.appnexus.com", Path: "/path/v1"},
			expErrors: 1,
		},
		{
			desc:      "Missing '/' in scheme",
			data:      ExternalCache{Host: "http:/www.appnexus.com", Path: "/path/v1"},
			expErrors: 1,
		},
		{
			desc:      "host with scheme, no path",
			data:      ExternalCache{Host: "http://www.appnexus.com", Path: ""},
			expErrors: 1,
		},
		{
			desc:      "scheme, no host nor path",
			data:      ExternalCache{Host: "http://", Path: ""},
			expErrors: 1,
		},
		{
			desc:      "Scheme Invalid",
			data:      ExternalCache{Scheme: "invalid", Host: "www.google.com", Path: "/path/v1"},
			expErrors: 1,
		},
		{
			desc:      "Scheme HTTP",
			data:      ExternalCache{Scheme: "http", Host: "www.google.com", Path: "/path/v1"},
			expErrors: 0,
		},
		{
			desc:      "Scheme HTTPS",
			data:      ExternalCache{Scheme: "https", Host: "www.google.com", Path: "/path/v1"},
			expErrors: 0,
		},
		{
			desc:      "Host with port",
			data:      ExternalCache{Scheme: "https", Host: "localhost:2424", Path: "/path/v1"},
			expErrors: 0,
		},
	}
	for _, test := range testCases {
		errs := test.data.validate([]error{})

		assert.Equal(t, test.expErrors, len(errs), "Test case threw unexpected number of errors. Desc: %s errMsg = %v \n", test.desc, errs)
	}
}

func TestDefaults(t *testing.T) {
	cfg, _ := newDefaultConfig(t)

	cmpInts(t, "port", 8000, cfg.Port)
	cmpInts(t, "admin_port", 6060, cfg.AdminPort)
	cmpInts(t, "auction_timeouts_ms.max", 0, int(cfg.AuctionTimeouts.Max))
	cmpInts(t, "max_request_size", 1024*256, int(cfg.MaxRequestSize))
	cmpInts(t, "host_cookie.ttl_days", 90, int(cfg.HostCookie.TTL))
	cmpInts(t, "host_cookie.max_cookie_size_bytes", 0, cfg.HostCookie.MaxCookieSizeBytes)
	cmpInts(t, "currency_converter.fetch_interval_seconds", 1800, cfg.CurrencyConverter.FetchIntervalSeconds)
	cmpStrings(t, "currency_converter.fetch_url", "https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json", cfg.CurrencyConverter.FetchURL)
	cmpBools(t, "account_required", false, cfg.AccountRequired)
	cmpInts(t, "metrics.influxdb.collection_rate_seconds", 20, cfg.Metrics.Influxdb.MetricSendInterval)
	cmpBools(t, "account_adapter_details", false, cfg.Metrics.Disabled.AccountAdapterDetails)
	cmpBools(t, "account_debug", true, cfg.Metrics.Disabled.AccountDebug)
	cmpBools(t, "account_stored_responses", true, cfg.Metrics.Disabled.AccountStoredResponses)
	cmpBools(t, "adapter_connections_metrics", true, cfg.Metrics.Disabled.AdapterConnectionMetrics)
	cmpBools(t, "adapter_buyeruid_scrubbed", true, cfg.Metrics.Disabled.AdapterBuyerUIDScrubbed)
	cmpBools(t, "adapter_gdpr_request_blocked", false, cfg.Metrics.Disabled.AdapterGDPRRequestBlocked)
	cmpStrings(t, "certificates_file", "", cfg.PemCertsFile)
	cmpInts(t, "stored_requests_timeout_ms", 50, cfg.StoredRequestsTimeout)
	cmpBools(t, "stored_requests.filesystem.enabled", false, cfg.StoredRequests.Files.Enabled)
	cmpStrings(t, "stored_requests.filesystem.directorypath", "./stored_requests/data/by_id", cfg.StoredRequests.Files.Path)
	cmpStrings(t, "stored_requests.http.endpoint", "", cfg.StoredRequests.HTTP.Endpoint)
	cmpStrings(t, "stored_requests.http.amp_endpoint", "", cfg.StoredRequests.HTTP.AmpEndpoint)
	cmpBools(t, "stored_requests.http.use_rfc3986_compliant_request_builder", false, cfg.StoredRequests.HTTP.UseRfcCompliantBuilder)
	cmpBools(t, "auto_gen_source_tid", true, cfg.AutoGenSourceTID)
	cmpBools(t, "generate_bid_id", false, cfg.GenerateBidID)
	cmpStrings(t, "experiment.adscert.mode", "off", cfg.Experiment.AdCerts.Mode)
	cmpStrings(t, "experiment.adscert.inprocess.origin", "", cfg.Experiment.AdCerts.InProcess.Origin)
	cmpStrings(t, "experiment.adscert.inprocess.key", "", cfg.Experiment.AdCerts.InProcess.PrivateKey)
	cmpInts(t, "experiment.adscert.inprocess.domain_check_interval_seconds", 30, cfg.Experiment.AdCerts.InProcess.DNSCheckIntervalInSeconds)
	cmpInts(t, "experiment.adscert.inprocess.domain_renewal_interval_seconds", 30, cfg.Experiment.AdCerts.InProcess.DNSRenewalIntervalInSeconds)
	cmpStrings(t, "experiment.adscert.remote.url", "", cfg.Experiment.AdCerts.Remote.Url)
	cmpInts(t, "experiment.adscert.remote.signing_timeout_ms", 5, cfg.Experiment.AdCerts.Remote.SigningTimeoutMs)
	cmpNils(t, "host_schain_node", cfg.HostSChainNode)
	cmpStrings(t, "datacenter", "", cfg.DataCenter)

	//Assert the price floor default values
	cmpBools(t, "price_floors.enabled", false, cfg.PriceFloors.Enabled)
	cmpInts(t, "price_floors.fetcher.worker", 20, cfg.PriceFloors.Fetcher.Worker)
	cmpInts(t, "price_floors.fetcher.capacity", 20000, cfg.PriceFloors.Fetcher.Capacity)
	cmpInts(t, "price_floors.fetcher.cache_size_mb", 64, cfg.PriceFloors.Fetcher.CacheSize)
	cmpInts(t, "price_floors.fetcher.http_client.max_connections_per_host", 0, cfg.PriceFloors.Fetcher.HttpClient.MaxConnsPerHost)
	cmpInts(t, "price_floors.fetcher.http_client.max_idle_connections", 40, cfg.PriceFloors.Fetcher.HttpClient.MaxIdleConns)
	cmpInts(t, "price_floors.fetcher.http_client.max_idle_connections_per_host", 2, cfg.PriceFloors.Fetcher.HttpClient.MaxIdleConnsPerHost)
	cmpInts(t, "price_floors.fetcher.http_client.idle_connection_timeout_seconds", 60, cfg.PriceFloors.Fetcher.HttpClient.IdleConnTimeout)
	cmpInts(t, "price_floors.fetcher.max_retries", 10, cfg.PriceFloors.Fetcher.MaxRetries)

	// Assert compression related defaults
	cmpBools(t, "compression.request.enable_gzip", false, cfg.Compression.Request.GZIP)
	cmpBools(t, "compression.response.enable_gzip", false, cfg.Compression.Response.GZIP)

	cmpBools(t, "account_defaults.price_floors.enabled", false, cfg.AccountDefaults.PriceFloors.Enabled)
	cmpInts(t, "account_defaults.price_floors.enforce_floors_rate", 100, cfg.AccountDefaults.PriceFloors.EnforceFloorsRate)
	cmpBools(t, "account_defaults.price_floors.adjust_for_bid_adjustment", true, cfg.AccountDefaults.PriceFloors.AdjustForBidAdjustment)
	cmpBools(t, "account_defaults.price_floors.enforce_deal_floors", false, cfg.AccountDefaults.PriceFloors.EnforceDealFloors)
	cmpBools(t, "account_defaults.price_floors.use_dynamic_data", false, cfg.AccountDefaults.PriceFloors.UseDynamicData)
	cmpInts(t, "account_defaults.price_floors.max_rules", 100, cfg.AccountDefaults.PriceFloors.MaxRule)
	cmpInts(t, "account_defaults.price_floors.max_schema_dims", 3, cfg.AccountDefaults.PriceFloors.MaxSchemaDims)
	cmpBools(t, "account_defaults.price_floors.fetch.enabled", false, cfg.AccountDefaults.PriceFloors.Fetcher.Enabled)
	cmpStrings(t, "account_defaults.price_floors.fetch.url", "", cfg.AccountDefaults.PriceFloors.Fetcher.URL)
	cmpInts(t, "account_defaults.price_floors.fetch.timeout_ms", 3000, cfg.AccountDefaults.PriceFloors.Fetcher.Timeout)
	cmpInts(t, "account_defaults.price_floors.fetch.max_file_size_kb", 100, cfg.AccountDefaults.PriceFloors.Fetcher.MaxFileSizeKB)
	cmpInts(t, "account_defaults.price_floors.fetch.max_rules", 1000, cfg.AccountDefaults.PriceFloors.Fetcher.MaxRules)
	cmpInts(t, "account_defaults.price_floors.fetch.period_sec", 3600, cfg.AccountDefaults.PriceFloors.Fetcher.Period)
	cmpInts(t, "account_defaults.price_floors.fetch.max_age_sec", 86400, cfg.AccountDefaults.PriceFloors.Fetcher.MaxAge)
	cmpInts(t, "account_defaults.price_floors.fetch.max_schema_dims", 0, cfg.AccountDefaults.PriceFloors.Fetcher.MaxSchemaDims)
	cmpStrings(t, "account_defaults.privacy.topicsdomain", "", cfg.AccountDefaults.Privacy.PrivacySandbox.TopicsDomain)
	cmpBools(t, "account_defaults.privacy.privacysandbox.cookiedeprecation.enabled", false, cfg.AccountDefaults.Privacy.PrivacySandbox.CookieDeprecation.Enabled)
	cmpInts(t, "account_defaults.privacy.privacysandbox.cookiedeprecation.ttl_sec", 604800, cfg.AccountDefaults.Privacy.PrivacySandbox.CookieDeprecation.TTLSec)

	cmpBools(t, "account_defaults.events.enabled", false, cfg.AccountDefaults.Events.Enabled)

	cmpBools(t, "hooks.enabled", false, cfg.Hooks.Enabled)
	cmpStrings(t, "validations.banner_creative_max_size", "skip", cfg.Validations.BannerCreativeMaxSize)
	cmpStrings(t, "validations.secure_markup", "skip", cfg.Validations.SecureMarkup)
	cmpInts(t, "validations.max_creative_width", 0, int(cfg.Validations.MaxCreativeWidth))
	cmpInts(t, "validations.max_creative_height", 0, int(cfg.Validations.MaxCreativeHeight))
	cmpBools(t, "account_modules_metrics", false, cfg.Metrics.Disabled.AccountModulesMetrics)

	cmpBools(t, "tmax_adjustments.enabled", false, cfg.TmaxAdjustments.Enabled)
	cmpUnsignedInts(t, "tmax_adjustments.bidder_response_duration_min_ms", 0, cfg.TmaxAdjustments.BidderResponseDurationMin)
	cmpUnsignedInts(t, "tmax_adjustments.bidder_network_latency_buffer_ms", 0, cfg.TmaxAdjustments.BidderNetworkLatencyBuffer)
	cmpUnsignedInts(t, "tmax_adjustments.pbs_response_preparation_duration_ms", 0, cfg.TmaxAdjustments.PBSResponsePreparationDuration)

	cmpInts(t, "tmax_default", 0, cfg.TmaxDefault)

	cmpInts(t, "account_defaults.privacy.ipv6.anon_keep_bits", 56, cfg.AccountDefaults.Privacy.IPv6Config.AnonKeepBits)
	cmpInts(t, "account_defaults.privacy.ipv4.anon_keep_bits", 24, cfg.AccountDefaults.Privacy.IPv4Config.AnonKeepBits)

	//Assert purpose VendorExceptionMap hash tables were built correctly
	cmpBools(t, "analytics.agma.enabled", false, cfg.Analytics.Agma.Enabled)
	cmpStrings(t, "analytics.agma.endpoint.timeout", "2s", cfg.Analytics.Agma.Endpoint.Timeout)
	cmpBools(t, "analytics.agma.endpoint.gzip", false, cfg.Analytics.Agma.Endpoint.Gzip)
	cmpStrings(t, "analytics.agma.endppoint.url", "https://go.pbs.agma-analytics.de/v1/prebid-server", cfg.Analytics.Agma.Endpoint.Url)
	cmpStrings(t, "analytics.agma.buffers.size", "2MB", cfg.Analytics.Agma.Buffers.BufferSize)
	cmpInts(t, "analytics.agma.buffers.count", 100, cfg.Analytics.Agma.Buffers.EventCount)
	cmpStrings(t, "analytics.agma.buffers.timeout", "15m", cfg.Analytics.Agma.Buffers.Timeout)
	cmpInts(t, "analytics.agma.accounts", 0, len(cfg.Analytics.Agma.Accounts))
	expectedTCF2 := TCF2{
		Enabled: true,
		Purpose1: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose2: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose3: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose4: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose5: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose6: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose7: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose8: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose9: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		Purpose10: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []string{},
			VendorExceptionMap: map[string]struct{}{},
		},
		SpecialFeature1: TCF2SpecialFeature{
			Enforce:            true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		PurposeOneTreatment: TCF2PurposeOneTreatment{
			Enabled:       true,
			AccessAllowed: true,
		},
	}
	expectedTCF2.PurposeConfigs = map[consentconstants.Purpose]*TCF2Purpose{
		1:  &expectedTCF2.Purpose1,
		2:  &expectedTCF2.Purpose2,
		3:  &expectedTCF2.Purpose3,
		4:  &expectedTCF2.Purpose4,
		5:  &expectedTCF2.Purpose5,
		6:  &expectedTCF2.Purpose6,
		7:  &expectedTCF2.Purpose7,
		8:  &expectedTCF2.Purpose8,
		9:  &expectedTCF2.Purpose9,
		10: &expectedTCF2.Purpose10,
	}
	assert.Equal(t, expectedTCF2, cfg.GDPR.TCF2, "gdpr.tcf2")
}

// When adding a new field, make sure the indentations are spaces not tabs otherwise read config may fail to parse the new field value.
var fullConfig = []byte(`
gdpr:
  host_vendor_id: 15
  default_value: "1"
  non_standard_publishers: ["pub1", "pub2"]
  eea_countries: ["eea1", "eea2"]
  tcf2:
    purpose1:
      enforce_vendors: false
      vendor_exceptions: ["foo1a", "foo1b"]
    purpose2:
      enforce_algo: "full"
      enforce_purpose: false
      enforce_vendors: false
      vendor_exceptions: ["foo2"]
    purpose3:
      enforce_algo: "basic"
      enforce_vendors: false
      vendor_exceptions: ["foo3"]
    purpose4:
      enforce_vendors: false
      vendor_exceptions: ["foo4"]
    purpose5:
      enforce_vendors: false
      vendor_exceptions: ["foo5"]
    purpose6:
      enforce_vendors: false
      vendor_exceptions: ["foo6"]
    purpose7:
      enforce_vendors: false
      vendor_exceptions: ["foo7"]
    purpose8:
      enforce_vendors: false
      vendor_exceptions: ["foo8"]
    purpose9:
      enforce_vendors: false
      vendor_exceptions: ["foo9"]
    purpose10:
      enforce_vendors: false
      vendor_exceptions: ["foo10"]
    special_feature1:
      vendor_exceptions: ["fooSP1"]
ccpa:
  enforce: true
lmt:
  enforce: true
host_cookie:
  cookie_name: userid
  family: prebid
  domain: cookies.prebid.org
  opt_out_url: http://prebid.org/optout
  opt_in_url: http://prebid.org/optin
  max_cookie_size_bytes: 32768
external_url: http://prebid-server.prebid.org/
host: prebid-server.prebid.org
port: 1234
admin_port: 5678
stored_requests_timeout_ms: 75
compression:
    request:
        enable_gzip: true
    response:
        enable_gzip: false
garbage_collector_threshold: 1
datacenter: "1"
auction_timeouts_ms:
  max: 123
  default: 50
cache:
  scheme: http
  host: prebidcache.net
  query: uuid=%PBS_CACHE_UUID%
external_cache:
  scheme: https
  host: www.externalprebidcache.net
  path: /endpoints/cache
http_client:
  max_connections_per_host: 10
  max_idle_connections: 500
  max_idle_connections_per_host: 20
  idle_connection_timeout_seconds: 30
http_client_cache:
  max_connections_per_host: 5
  max_idle_connections: 1
  max_idle_connections_per_host: 2
  idle_connection_timeout_seconds: 3
currency_converter:
  fetch_url: https://currency.prebid.org
  fetch_interval_seconds: 1800
recaptcha_secret: asdfasdfasdfasdf
metrics:
  influxdb:
    host: upstream:8232
    database: metricsdb
    measurement: anyMeasurement
    username: admin
    password: admin1324
    align_timestamps: true
    metric_send_interval: 30
  disabled_metrics:
    account_adapter_details: true
    account_debug: false
    account_stored_responses: false
    adapter_connections_metrics: true
    adapter_buyeruid_scrubbed: false
    adapter_gdpr_request_blocked: true
    account_modules_metrics: true
blocked_apps: ["spamAppID","sketchy-app-id"]
account_required: true
auto_gen_source_tid: false
certificates_file: /etc/ssl/cert.pem
request_validation:
    ipv4_private_networks: ["1.1.1.0/24"]
    ipv6_private_networks: ["1111::/16", "2222::/16"]
generate_bid_id: true
host_schain_node:
    asi: "pbshostcompany.com"
    sid: "00001"
    rid: "BidRequest"
    hp: 1
validations:
    banner_creative_max_size: "skip"
    secure_markup: "skip"
    max_creative_width: 0
    max_creative_height: 0
experiment:
    adscert:
        mode: inprocess
        inprocess:
            origin: "http://test.com"
            key: "ABC123"
            domain_check_interval_seconds: 40
            domain_renewal_interval_seconds : 60
        remote:
            url: ""
            signing_timeout_ms: 10
hooks:
    enabled: true
price_floors:
    enabled: true
    fetcher:
      worker: 20
      capacity: 20000
      cache_size_mb: 8
      http_client:
        max_connections_per_host: 5
        max_idle_connections: 1
        max_idle_connections_per_host: 2
        idle_connection_timeout_seconds: 10
      max_retries: 5
account_defaults:
    events:
        enabled: true
    price_floors:
        enabled: true
        enforce_floors_rate: 50
        adjust_for_bid_adjustment: false
        enforce_deal_floors: true
        use_dynamic_data: true
        max_rules: 120
        max_schema_dims: 5
        fetch:
          enabled: true
          url: http://test.com/floors
          timeout_ms: 500
          max_file_size_kb: 200
          max_rules: 500
          period_sec: 2000
          max_age_sec: 6000
          max_schema_dims: 10
    bidadjustments:
        mediatype:
            '*':
                '*':
                    '*':
                        - adjtype: multiplier
                          value: 1.01
                          currency: USD
            video-instream:
                bidder:
                    deal_id:
                        - adjtype: cpm
                          value: 1.02
                          currency: EUR
    privacy:
        ipv6:
            anon_keep_bits: 50
        ipv4:
            anon_keep_bits: 20
        dsa:
            default: "{\"dsarequired\":3,\"pubrender\":1,\"datatopub\":2,\"transparency\":[{\"domain\":\"domain.com\",\"dsaparams\":[1]}]}"
            gdpr_only: true
        privacysandbox:
            topicsdomain: "test.com"
            cookiedeprecation:
                enabled: true
                ttl_sec: 86400
tmax_adjustments:
  enabled: true
  bidder_response_duration_min_ms: 700
  bidder_network_latency_buffer_ms: 100
  pbs_response_preparation_duration_ms: 100
tmax_default: 600
analytics:
  agma:
    enabled: true
    endpoint:
      url: "http://test.com"
      timeout: "5s"
      gzip: false
    buffers:
      size: 10MB
      count: 111
      timeout: 5m
    accounts:
    - code: agma-code
      publisher_id: publisher-id
      site_app_id: site-or-app-id
`)

func cmpStrings(t *testing.T, key, expected, actual string) {
	t.Helper()
	assert.Equal(t, expected, actual, "%s: %s != %s", key, expected, actual)
}

func cmpInts(t *testing.T, key string, expected, actual int) {
	t.Helper()
	assert.Equal(t, expected, actual, "%s: %d != %d", key, expected, actual)
}

func cmpUnsignedInts(t *testing.T, key string, expected, actual uint) {
	t.Helper()
	assert.Equal(t, expected, actual, "%s: %d != %d", key, expected, actual)
}

func cmpInt8s(t *testing.T, key string, expected, actual *int8) {
	t.Helper()
	assert.Equal(t, expected, actual, "%s: %d != %d", key, expected, actual)
}

func cmpBools(t *testing.T, key string, expected, actual bool) {
	t.Helper()
	assert.Equal(t, expected, actual, "%s: %t != %t", key, expected, actual)
}

func cmpNils(t *testing.T, key string, a interface{}) {
	t.Helper()
	assert.Nilf(t, a, "%s: %t != nil", key, a)
}

func TestFullConfig(t *testing.T) {
	int8One := int8(1)

	v := viper.New()
	SetupViper(v, "", bidderInfos)
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(fullConfig))
	cfg, err := New(v, bidderInfos, mockNormalizeBidderName)
	assert.NoError(t, err, "Setting up config should work but it doesn't")
	cmpStrings(t, "cookie domain", "cookies.prebid.org", cfg.HostCookie.Domain)
	cmpStrings(t, "cookie name", "userid", cfg.HostCookie.CookieName)
	cmpStrings(t, "cookie family", "prebid", cfg.HostCookie.Family)
	cmpStrings(t, "opt out", "http://prebid.org/optout", cfg.HostCookie.OptOutURL)
	cmpStrings(t, "opt in", "http://prebid.org/optin", cfg.HostCookie.OptInURL)
	cmpStrings(t, "external url", "http://prebid-server.prebid.org/", cfg.ExternalURL)
	cmpStrings(t, "host", "prebid-server.prebid.org", cfg.Host)
	cmpInts(t, "port", 1234, cfg.Port)
	cmpInts(t, "admin_port", 5678, cfg.AdminPort)
	cmpInts(t, "garbage_collector_threshold", 1, cfg.GarbageCollectorThreshold)
	cmpInts(t, "auction_timeouts_ms.default", 50, int(cfg.AuctionTimeouts.Default))
	cmpInts(t, "auction_timeouts_ms.max", 123, int(cfg.AuctionTimeouts.Max))
	cmpInts(t, "stored_request_timeout_ms", 75, cfg.StoredRequestsTimeout)
	cmpStrings(t, "stored_requests.http.endpoint", "", cfg.StoredRequests.HTTP.Endpoint)
	cmpStrings(t, "stored_requests.http.amp_endpoint", "", cfg.StoredRequests.HTTP.AmpEndpoint)
	cmpBools(t, "stored_requests.http.use_rfc3986_compliant_request_builder", false, cfg.StoredRequests.HTTP.UseRfcCompliantBuilder)
	cmpStrings(t, "cache.scheme", "http", cfg.CacheURL.Scheme)
	cmpStrings(t, "cache.host", "prebidcache.net", cfg.CacheURL.Host)
	cmpStrings(t, "cache.query", "uuid=%PBS_CACHE_UUID%", cfg.CacheURL.Query)
	cmpStrings(t, "external_cache.scheme", "https", cfg.ExtCacheURL.Scheme)
	cmpStrings(t, "external_cache.host", "www.externalprebidcache.net", cfg.ExtCacheURL.Host)
	cmpStrings(t, "external_cache.path", "/endpoints/cache", cfg.ExtCacheURL.Path)
	cmpInts(t, "http_client.max_connections_per_host", 10, cfg.Client.MaxConnsPerHost)
	cmpInts(t, "http_client.max_idle_connections", 500, cfg.Client.MaxIdleConns)
	cmpInts(t, "http_client.max_idle_connections_per_host", 20, cfg.Client.MaxIdleConnsPerHost)
	cmpInts(t, "http_client.idle_connection_timeout_seconds", 30, cfg.Client.IdleConnTimeout)
	cmpInts(t, "http_client_cache.max_connections_per_host", 5, cfg.CacheClient.MaxConnsPerHost)
	cmpInts(t, "http_client_cache.max_idle_connections", 1, cfg.CacheClient.MaxIdleConns)
	cmpInts(t, "http_client_cache.max_idle_connections_per_host", 2, cfg.CacheClient.MaxIdleConnsPerHost)
	cmpInts(t, "http_client_cache.idle_connection_timeout_seconds", 3, cfg.CacheClient.IdleConnTimeout)
	cmpInts(t, "gdpr.host_vendor_id", 15, cfg.GDPR.HostVendorID)
	cmpStrings(t, "gdpr.default_value", "1", cfg.GDPR.DefaultValue)
	cmpStrings(t, "host_schain_node.asi", "pbshostcompany.com", cfg.HostSChainNode.ASI)
	cmpStrings(t, "host_schain_node.sid", "00001", cfg.HostSChainNode.SID)
	cmpStrings(t, "host_schain_node.rid", "BidRequest", cfg.HostSChainNode.RID)
	cmpInt8s(t, "host_schain_node.hp", &int8One, cfg.HostSChainNode.HP)
	cmpStrings(t, "datacenter", "1", cfg.DataCenter)
	cmpStrings(t, "validations.banner_creative_max_size", "skip", cfg.Validations.BannerCreativeMaxSize)
	cmpStrings(t, "validations.secure_markup", "skip", cfg.Validations.SecureMarkup)
	cmpInts(t, "validations.max_creative_width", 0, int(cfg.Validations.MaxCreativeWidth))
	cmpInts(t, "validations.max_creative_height", 0, int(cfg.Validations.MaxCreativeHeight))
	cmpBools(t, "tmax_adjustments.enabled", true, cfg.TmaxAdjustments.Enabled)
	cmpUnsignedInts(t, "tmax_adjustments.bidder_response_duration_min_ms", 700, cfg.TmaxAdjustments.BidderResponseDurationMin)
	cmpUnsignedInts(t, "tmax_adjustments.bidder_network_latency_buffer_ms", 100, cfg.TmaxAdjustments.BidderNetworkLatencyBuffer)
	cmpUnsignedInts(t, "tmax_adjustments.pbs_response_preparation_duration_ms", 100, cfg.TmaxAdjustments.PBSResponsePreparationDuration)
	cmpInts(t, "tmax_default", 600, cfg.TmaxDefault)

	//Assert the price floor values
	cmpBools(t, "price_floors.enabled", true, cfg.PriceFloors.Enabled)
	cmpInts(t, "price_floors.fetcher.worker", 20, cfg.PriceFloors.Fetcher.Worker)
	cmpInts(t, "price_floors.fetcher.capacity", 20000, cfg.PriceFloors.Fetcher.Capacity)
	cmpInts(t, "price_floors.fetcher.cache_size_mb", 8, cfg.PriceFloors.Fetcher.CacheSize)
	cmpInts(t, "price_floors.fetcher.http_client.max_connections_per_host", 5, cfg.PriceFloors.Fetcher.HttpClient.MaxConnsPerHost)
	cmpInts(t, "price_floors.fetcher.http_client.max_idle_connections", 1, cfg.PriceFloors.Fetcher.HttpClient.MaxIdleConns)
	cmpInts(t, "price_floors.fetcher.http_client.max_idle_connections_per_host", 2, cfg.PriceFloors.Fetcher.HttpClient.MaxIdleConnsPerHost)
	cmpInts(t, "price_floors.fetcher.http_client.idle_connection_timeout_seconds", 10, cfg.PriceFloors.Fetcher.HttpClient.IdleConnTimeout)
	cmpInts(t, "price_floors.fetcher.max_retries", 5, cfg.PriceFloors.Fetcher.MaxRetries)
	cmpBools(t, "account_defaults.price_floors.enabled", true, cfg.AccountDefaults.PriceFloors.Enabled)
	cmpInts(t, "account_defaults.price_floors.enforce_floors_rate", 50, cfg.AccountDefaults.PriceFloors.EnforceFloorsRate)
	cmpBools(t, "account_defaults.price_floors.adjust_for_bid_adjustment", false, cfg.AccountDefaults.PriceFloors.AdjustForBidAdjustment)
	cmpBools(t, "account_defaults.price_floors.enforce_deal_floors", true, cfg.AccountDefaults.PriceFloors.EnforceDealFloors)
	cmpBools(t, "account_defaults.price_floors.use_dynamic_data", true, cfg.AccountDefaults.PriceFloors.UseDynamicData)
	cmpInts(t, "account_defaults.price_floors.max_rules", 120, cfg.AccountDefaults.PriceFloors.MaxRule)
	cmpInts(t, "account_defaults.price_floors.max_schema_dims", 5, cfg.AccountDefaults.PriceFloors.MaxSchemaDims)
	cmpBools(t, "account_defaults.price_floors.fetch.enabled", true, cfg.AccountDefaults.PriceFloors.Fetcher.Enabled)
	cmpStrings(t, "account_defaults.price_floors.fetch.url", "http://test.com/floors", cfg.AccountDefaults.PriceFloors.Fetcher.URL)
	cmpInts(t, "account_defaults.price_floors.fetch.timeout_ms", 500, cfg.AccountDefaults.PriceFloors.Fetcher.Timeout)
	cmpInts(t, "account_defaults.price_floors.fetch.max_file_size_kb", 200, cfg.AccountDefaults.PriceFloors.Fetcher.MaxFileSizeKB)
	cmpInts(t, "account_defaults.price_floors.fetch.max_rules", 500, cfg.AccountDefaults.PriceFloors.Fetcher.MaxRules)
	cmpInts(t, "account_defaults.price_floors.fetch.period_sec", 2000, cfg.AccountDefaults.PriceFloors.Fetcher.Period)
	cmpInts(t, "account_defaults.price_floors.fetch.max_age_sec", 6000, cfg.AccountDefaults.PriceFloors.Fetcher.MaxAge)
	cmpInts(t, "account_defaults.price_floors.fetch.max_schema_dims", 10, cfg.AccountDefaults.PriceFloors.Fetcher.MaxSchemaDims)

	// Assert the DSA was correctly unmarshalled and DefaultUnpacked was built correctly
	expectedDSA := AccountDSA{
		Default: "{\"dsarequired\":3,\"pubrender\":1,\"datatopub\":2,\"transparency\":[{\"domain\":\"domain.com\",\"dsaparams\":[1]}]}",
		DefaultUnpacked: &openrtb_ext.ExtRegsDSA{
			Required:  ptrutil.ToPtr[int8](3),
			PubRender: ptrutil.ToPtr[int8](1),
			DataToPub: ptrutil.ToPtr[int8](2),
			Transparency: []openrtb_ext.ExtBidDSATransparency{
				{
					Domain: "domain.com",
					Params: []int{1},
				},
			},
		},
		GDPROnly: true,
	}
	assert.Equal(t, &expectedDSA, cfg.AccountDefaults.Privacy.DSA)

	cmpBools(t, "account_defaults.events.enabled", true, cfg.AccountDefaults.Events.Enabled)

	cmpInts(t, "account_defaults.privacy.ipv6.anon_keep_bits", 50, cfg.AccountDefaults.Privacy.IPv6Config.AnonKeepBits)
	cmpInts(t, "account_defaults.privacy.ipv4.anon_keep_bits", 20, cfg.AccountDefaults.Privacy.IPv4Config.AnonKeepBits)

	cmpStrings(t, "account_defaults.privacy.topicsdomain", "test.com", cfg.AccountDefaults.Privacy.PrivacySandbox.TopicsDomain)
	cmpBools(t, "account_defaults.privacy.cookiedeprecation.enabled", true, cfg.AccountDefaults.Privacy.PrivacySandbox.CookieDeprecation.Enabled)
	cmpInts(t, "account_defaults.privacy.cookiedeprecation.ttl_sec", 86400, cfg.AccountDefaults.Privacy.PrivacySandbox.CookieDeprecation.TTLSec)

	// Assert compression related defaults
	cmpBools(t, "compression.request.enable_gzip", true, cfg.Compression.Request.GZIP)
	cmpBools(t, "compression.response.enable_gzip", false, cfg.Compression.Response.GZIP)

	//Assert the NonStandardPublishers was correctly unmarshalled
	assert.Equal(t, []string{"pub1", "pub2"}, cfg.GDPR.NonStandardPublishers, "gdpr.non_standard_publishers")
	assert.Equal(t, map[string]struct{}{"pub1": {}, "pub2": {}}, cfg.GDPR.NonStandardPublisherMap, "gdpr.non_standard_publishers Hash Map")

	// Assert EEA Countries was correctly unmarshalled and the EEACountriesMap built correctly.
	assert.Equal(t, []string{"eea1", "eea2"}, cfg.GDPR.EEACountries, "gdpr.eea_countries")
	assert.Equal(t, map[string]struct{}{"eea1": {}, "eea2": {}}, cfg.GDPR.EEACountriesMap, "gdpr.eea_countries Hash Map")

	cmpBools(t, "ccpa.enforce", true, cfg.CCPA.Enforce)
	cmpBools(t, "lmt.enforce", true, cfg.LMT.Enforce)

	//Assert the NonStandardPublishers was correctly unmarshalled
	cmpStrings(t, "blocked_apps", "spamAppID", cfg.BlockedApps[0])
	cmpStrings(t, "blocked_apps", "sketchy-app-id", cfg.BlockedApps[1])

	//Assert the BlockedAppsLookup hash table was built correctly
	for i := 0; i < len(cfg.BlockedApps); i++ {
		cmpBools(t, "cfg.BlockedAppsLookup", true, cfg.BlockedAppsLookup[cfg.BlockedApps[i]])
	}

	//Assert purpose VendorExceptionMap hash tables were built correctly
	expectedTCF2 := TCF2{
		Enabled: true,
		Purpose1: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo1a", "foo1b"},
			VendorExceptionMap: map[string]struct{}{"foo1a": {}, "foo1b": {}},
		},
		Purpose2: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     false,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo2"},
			VendorExceptionMap: map[string]struct{}{"foo2": {}},
		},
		Purpose3: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoBasic,
			EnforceAlgoID:      TCF2BasicEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo3"},
			VendorExceptionMap: map[string]struct{}{"foo3": {}},
		},
		Purpose4: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo4"},
			VendorExceptionMap: map[string]struct{}{"foo4": {}},
		},
		Purpose5: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo5"},
			VendorExceptionMap: map[string]struct{}{"foo5": {}},
		},
		Purpose6: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo6"},
			VendorExceptionMap: map[string]struct{}{"foo6": {}},
		},
		Purpose7: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo7"},
			VendorExceptionMap: map[string]struct{}{"foo7": {}},
		},
		Purpose8: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo8"},
			VendorExceptionMap: map[string]struct{}{"foo8": {}},
		},
		Purpose9: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo9"},
			VendorExceptionMap: map[string]struct{}{"foo9": {}},
		},
		Purpose10: TCF2Purpose{
			EnforceAlgo:        TCF2EnforceAlgoFull,
			EnforceAlgoID:      TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []string{"foo10"},
			VendorExceptionMap: map[string]struct{}{"foo10": {}},
		},
		SpecialFeature1: TCF2SpecialFeature{
			Enforce:            true, // true by default
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("fooSP1")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("fooSP1"): {}},
		},
		PurposeOneTreatment: TCF2PurposeOneTreatment{
			Enabled:       true, // true by default
			AccessAllowed: true, // true by default
		},
	}
	expectedTCF2.PurposeConfigs = map[consentconstants.Purpose]*TCF2Purpose{
		1:  &expectedTCF2.Purpose1,
		2:  &expectedTCF2.Purpose2,
		3:  &expectedTCF2.Purpose3,
		4:  &expectedTCF2.Purpose4,
		5:  &expectedTCF2.Purpose5,
		6:  &expectedTCF2.Purpose6,
		7:  &expectedTCF2.Purpose7,
		8:  &expectedTCF2.Purpose8,
		9:  &expectedTCF2.Purpose9,
		10: &expectedTCF2.Purpose10,
	}

	expectedBidAdjustments := &openrtb_ext.ExtRequestPrebidBidAdjustments{
		MediaType: openrtb_ext.MediaType{
			WildCard: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"*": {
					"*": []openrtb_ext.Adjustment{{Type: "multiplier", Value: 1.01, Currency: "USD"}},
				},
			},
			VideoInstream: map[openrtb_ext.BidderName]openrtb_ext.AdjustmentsByDealID{
				"bidder": {
					"deal_id": []openrtb_ext.Adjustment{{Type: "cpm", Value: 1.02, Currency: "EUR"}},
				},
			},
		},
	}
	assert.Equal(t, expectedTCF2, cfg.GDPR.TCF2, "gdpr.tcf2")
	assert.Equal(t, expectedBidAdjustments, cfg.AccountDefaults.BidAdjustments)

	cmpStrings(t, "currency_converter.fetch_url", "https://currency.prebid.org", cfg.CurrencyConverter.FetchURL)
	cmpInts(t, "currency_converter.fetch_interval_seconds", 1800, cfg.CurrencyConverter.FetchIntervalSeconds)
	cmpStrings(t, "recaptcha_secret", "asdfasdfasdfasdf", cfg.RecaptchaSecret)
	cmpStrings(t, "metrics.influxdb.host", "upstream:8232", cfg.Metrics.Influxdb.Host)
	cmpStrings(t, "metrics.influxdb.database", "metricsdb", cfg.Metrics.Influxdb.Database)
	cmpStrings(t, "metrics.influxdb.measurement", "anyMeasurement", cfg.Metrics.Influxdb.Measurement)
	cmpStrings(t, "metrics.influxdb.username", "admin", cfg.Metrics.Influxdb.Username)
	cmpStrings(t, "metrics.influxdb.password", "admin1324", cfg.Metrics.Influxdb.Password)
	cmpBools(t, "metrics.influxdb.align_timestamps", true, cfg.Metrics.Influxdb.AlignTimestamps)
	cmpInts(t, "metrics.influxdb.metric_send_interval", 30, cfg.Metrics.Influxdb.MetricSendInterval)
	cmpStrings(t, "", "http://prebidcache.net", cfg.CacheURL.GetBaseURL())
	cmpStrings(t, "", "http://prebidcache.net/cache?uuid=a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11", cfg.GetCachedAssetURL("a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11"))
	cmpBools(t, "account_required", true, cfg.AccountRequired)
	cmpBools(t, "auto_gen_source_tid", false, cfg.AutoGenSourceTID)
	cmpBools(t, "account_adapter_details", true, cfg.Metrics.Disabled.AccountAdapterDetails)
	cmpBools(t, "account_debug", false, cfg.Metrics.Disabled.AccountDebug)
	cmpBools(t, "account_stored_responses", false, cfg.Metrics.Disabled.AccountStoredResponses)
	cmpBools(t, "adapter_connections_metrics", true, cfg.Metrics.Disabled.AdapterConnectionMetrics)
	cmpBools(t, "adapter_buyeruid_scrubbed", false, cfg.Metrics.Disabled.AdapterBuyerUIDScrubbed)
	cmpBools(t, "adapter_gdpr_request_blocked", true, cfg.Metrics.Disabled.AdapterGDPRRequestBlocked)
	cmpStrings(t, "certificates_file", "/etc/ssl/cert.pem", cfg.PemCertsFile)
	cmpStrings(t, "request_validation.ipv4_private_networks", "1.1.1.0/24", cfg.RequestValidation.IPv4PrivateNetworks[0])
	cmpStrings(t, "request_validation.ipv6_private_networks", "1111::/16", cfg.RequestValidation.IPv6PrivateNetworks[0])
	cmpStrings(t, "request_validation.ipv6_private_networks", "2222::/16", cfg.RequestValidation.IPv6PrivateNetworks[1])
	cmpBools(t, "generate_bid_id", true, cfg.GenerateBidID)
	cmpStrings(t, "debug.override_token", "", cfg.Debug.OverrideToken)
	cmpStrings(t, "experiment.adscert.mode", "inprocess", cfg.Experiment.AdCerts.Mode)
	cmpStrings(t, "experiment.adscert.inprocess.origin", "http://test.com", cfg.Experiment.AdCerts.InProcess.Origin)
	cmpStrings(t, "experiment.adscert.inprocess.key", "ABC123", cfg.Experiment.AdCerts.InProcess.PrivateKey)
	cmpInts(t, "experiment.adscert.inprocess.domain_check_interval_seconds", 40, cfg.Experiment.AdCerts.InProcess.DNSCheckIntervalInSeconds)
	cmpInts(t, "experiment.adscert.inprocess.domain_renewal_interval_seconds", 60, cfg.Experiment.AdCerts.InProcess.DNSRenewalIntervalInSeconds)
	cmpStrings(t, "experiment.adscert.remote.url", "", cfg.Experiment.AdCerts.Remote.Url)
	cmpInts(t, "experiment.adscert.remote.signing_timeout_ms", 10, cfg.Experiment.AdCerts.Remote.SigningTimeoutMs)
	cmpBools(t, "hooks.enabled", true, cfg.Hooks.Enabled)
	cmpBools(t, "account_modules_metrics", true, cfg.Metrics.Disabled.AccountModulesMetrics)
	cmpBools(t, "analytics.agma.enabled", true, cfg.Analytics.Agma.Enabled)
	cmpStrings(t, "analytics.agma.endpoint.timeout", "5s", cfg.Analytics.Agma.Endpoint.Timeout)
	cmpBools(t, "analytics.agma.endpoint.gzip", false, cfg.Analytics.Agma.Endpoint.Gzip)
	cmpStrings(t, "analytics.agma.endpoint.url", "http://test.com", cfg.Analytics.Agma.Endpoint.Url)
	cmpStrings(t, "analytics.agma.buffers.size", "10MB", cfg.Analytics.Agma.Buffers.BufferSize)
	cmpInts(t, "analytics.agma.buffers.count", 111, cfg.Analytics.Agma.Buffers.EventCount)
	cmpStrings(t, "analytics.agma.buffers.timeout", "5m", cfg.Analytics.Agma.Buffers.Timeout)
	cmpStrings(t, "analytics.agma.accounts.0.publisher_id", "publisher-id", cfg.Analytics.Agma.Accounts[0].PublisherId)
	cmpStrings(t, "analytics.agma.accounts.0.code", "agma-code", cfg.Analytics.Agma.Accounts[0].Code)
	cmpStrings(t, "analytics.agma.accounts.0.site_app_id", "site-or-app-id", cfg.Analytics.Agma.Accounts[0].SiteAppId)
}

func TestValidateConfig(t *testing.T) {
	cfg := Configuration{
		GDPR: GDPR{
			DefaultValue: "1",
			TCF2: TCF2{
				Purpose1:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoBasic},
				Purpose2:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoFull},
				Purpose3:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoBasic},
				Purpose4:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoFull},
				Purpose5:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoBasic},
				Purpose6:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoFull},
				Purpose7:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoBasic},
				Purpose8:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoFull},
				Purpose9:  TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoBasic},
				Purpose10: TCF2Purpose{EnforceAlgo: TCF2EnforceAlgoFull},
			},
		},
		StoredRequests: StoredRequests{
			Files: FileFetcherConfig{Enabled: true},
			InMemoryCache: InMemoryCache{
				Type: "none",
			},
		},
		StoredRequestsTimeout: 50,
		StoredVideo: StoredRequests{
			Files: FileFetcherConfig{Enabled: true},
			InMemoryCache: InMemoryCache{
				Type: "none",
			},
		},
		CategoryMapping: StoredRequests{
			Files: FileFetcherConfig{Enabled: true},
		},
		Accounts: StoredRequests{
			Files:         FileFetcherConfig{Enabled: true},
			InMemoryCache: InMemoryCache{Type: "none"},
		},
		AccountDefaults: Account{
			PriceFloors: AccountPriceFloors{
				Fetcher: AccountFloorFetch{
					Timeout: 100,
					Period:  300,
					MaxAge:  600,
				},
			},
		},
	}

	v := viper.New()
	v.Set("gdpr.default_value", "0")

	resolvedStoredRequestsConfig(&cfg)
	err := cfg.validate(v)
	assert.Nil(t, err, "OpenRTB filesystem config should work. %v", err)
}

func TestMigrateConfigFromEnv(t *testing.T) {
	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_ENDPOINT"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_ENDPOINT", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_ENDPOINT")
	}

	os.Setenv("PBS_ADAPTERS_BIDDER1_ENDPOINT", "http://bidder1_override.com")
	cfg, _ := newDefaultConfig(t)
	cmpStrings(t, "adapters.bidder1.endpoint", "http://bidder1_override.com", cfg.BidderInfos["bidder1"].Endpoint)
}

func TestUserSyncFromEnv(t *testing.T) {
	truePtr := true

	// setup env vars for testing
	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_USER_MACRO"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_USER_MACRO", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_USER_MACRO")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_USERSYNC_SUPPORT_CORS"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_SUPPORT_CORS", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_USERSYNC_SUPPORT_CORS")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER2_USERSYNC_IFRAME_URL"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER2_USERSYNC_IFRAME_URL", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER2_USERSYNC_IFRAME_URL")
	}

	// set new
	os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL", "http://some.url/sync?redirect={{.RedirectURL}}")
	os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_USER_MACRO", "[UID]")
	os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_SUPPORT_CORS", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER2_USERSYNC_IFRAME_URL", "http://somedifferent.url/sync?redirect={{.RedirectURL}}")

	cfg, _ := newDefaultConfig(t)

	assert.Equal(t, "http://some.url/sync?redirect={{.RedirectURL}}", cfg.BidderInfos["bidder1"].Syncer.Redirect.URL)
	assert.Equal(t, "[UID]", cfg.BidderInfos["bidder1"].Syncer.Redirect.UserMacro)
	assert.Nil(t, cfg.BidderInfos["bidder1"].Syncer.IFrame)
	assert.Equal(t, &truePtr, cfg.BidderInfos["bidder1"].Syncer.SupportCORS)

	assert.Equal(t, "http://somedifferent.url/sync?redirect={{.RedirectURL}}", cfg.BidderInfos["bidder2"].Syncer.IFrame.URL)
	assert.Nil(t, cfg.BidderInfos["bidder2"].Syncer.Redirect)
	assert.Nil(t, cfg.BidderInfos["bidder2"].Syncer.SupportCORS)
}

func TestBidderInfoFromEnv(t *testing.T) {
	// setup env vars for testing
	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_DISABLED"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_DISABLED", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_DISABLED")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_ENDPOINT"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_ENDPOINT", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_ENDPOINT")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_EXTRA_INFO"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_EXTRA_INFO", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_EXTRA_INFO")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_DEBUG_ALLOW"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_DEBUG_ALLOW", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_DEBUG_ALLOW")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_GVLVENDORID"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_GVLVENDORID", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_GVLVENDORID")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_EXPERIMENT_ADSCERT_ENABLED"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_EXPERIMENT_ADSCERT_ENABLED", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_EXPERIMENT_ADSCERT_ENABLED")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_XAPI_USERNAME"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_XAPI_USERNAME", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_XAPI_USERNAME")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL")
	}
	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_OPENRTB_VERSION"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_OPENRTB_VERSION", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_OPENRTB_VERSION")
	}

	// set new
	os.Setenv("PBS_ADAPTERS_BIDDER1_DISABLED", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER1_ENDPOINT", "http://some.url/override")
	os.Setenv("PBS_ADAPTERS_BIDDER1_EXTRA_INFO", `{"extrainfo": true}`)
	os.Setenv("PBS_ADAPTERS_BIDDER1_DEBUG_ALLOW", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER1_GVLVENDORID", "42")
	os.Setenv("PBS_ADAPTERS_BIDDER1_EXPERIMENT_ADSCERT_ENABLED", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER1_XAPI_USERNAME", "username_override")
	os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL", "http://some.url/sync?redirect={{.RedirectURL}}")
	os.Setenv("PBS_ADAPTERS_BIDDER1_OPENRTB_VERSION", "2.6")

	cfg, _ := newDefaultConfig(t)

	assert.Equal(t, true, cfg.BidderInfos["bidder1"].Disabled)
	assert.Equal(t, "http://some.url/override", cfg.BidderInfos["bidder1"].Endpoint)
	assert.Equal(t, `{"extrainfo": true}`, cfg.BidderInfos["bidder1"].ExtraAdapterInfo)

	assert.Equal(t, true, cfg.BidderInfos["bidder1"].Debug.Allow)
	assert.Equal(t, uint16(42), cfg.BidderInfos["bidder1"].GVLVendorID)

	assert.Equal(t, true, cfg.BidderInfos["bidder1"].Experiment.AdsCert.Enabled)
	assert.Equal(t, "username_override", cfg.BidderInfos["bidder1"].XAPI.Username)

	assert.Equal(t, "2.6", cfg.BidderInfos["bidder1"].OpenRTB.Version)
}

func TestIsConfigInfoPresent(t *testing.T) {
	configPrefix1Field2Only := []byte(`
      prefix1:
        field2: "value2"
    `)
	configPrefix1Field4Only := []byte(`
      prefix1:
        field4: "value4"
      `)
	configPrefix1Field2AndField3 := []byte(`
      prefix1:
        field2: "value2"
        field3: "value3"
      `)

	tests := []struct {
		description string
		config      []byte
		keyPrefix   string
		fields      []string
		wantResult  bool
	}{
		{
			description: "config is nil",
			config:      nil,
			keyPrefix:   "prefix1",
			fields:      []string{"field1", "field2", "field3"},
			wantResult:  false,
		},
		{
			description: "config is empty",
			config:      []byte{},
			keyPrefix:   "prefix1",
			fields:      []string{"field1", "field2", "field3"},
			wantResult:  false,
		},
		{
			description: "present - one field exists in config",
			config:      configPrefix1Field2Only,
			keyPrefix:   "prefix1",
			fields:      []string{"field1", "field2", "field3"},
			wantResult:  true,
		},
		{
			description: "present - many fields exist in config",
			config:      configPrefix1Field2AndField3,
			keyPrefix:   "prefix1",
			fields:      []string{"field1", "field2", "field3"},
			wantResult:  true,
		},
		{
			description: "not present - field not found",
			config:      configPrefix1Field4Only,
			keyPrefix:   "prefix1",
			fields:      []string{"field1", "field2", "field3"},
			wantResult:  false,
		},
		{
			description: "not present - field exists but with a different prefix",
			config:      configPrefix1Field2Only,
			keyPrefix:   "prefix2",
			fields:      []string{"field1", "field2", "field3"},
			wantResult:  false,
		},
		{
			description: "not present - fields is nil",
			config:      configPrefix1Field2Only,
			keyPrefix:   "prefix1",
			fields:      nil,
			wantResult:  false,
		},
		{
			description: "not present - fields is empty",
			config:      configPrefix1Field2Only,
			keyPrefix:   "prefix1",
			fields:      []string{},
			wantResult:  false,
		},
	}

	for _, tt := range tests {
		v := viper.New()
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer(tt.config))

		result := isConfigInfoPresent(v, tt.keyPrefix, tt.fields)
		assert.Equal(t, tt.wantResult, result, tt.description)
	}
}

func TestNegativeOrZeroStoredRequestsTimeout(t *testing.T) {
	cfg, v := newDefaultConfig(t)

	cfg.StoredRequestsTimeout = -1
	assertOneError(t, cfg.validate(v), "cfg.stored_requests_timeout_ms must be > 0. Got -1")

	cfg.StoredRequestsTimeout = 0
	assertOneError(t, cfg.validate(v), "cfg.stored_requests_timeout_ms must be > 0. Got 0")
}

func TestNegativeRequestSize(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.MaxRequestSize = -1
	assertOneError(t, cfg.validate(v), "cfg.max_request_size must be >= 0. Got -1")
}

func TestNegativePrometheusTimeout(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.Metrics.Prometheus.Port = 8001
	cfg.Metrics.Prometheus.TimeoutMillisRaw = 0
	assertOneError(t, cfg.validate(v), "metrics.prometheus.timeout_ms must be positive if metrics.prometheus.port is defined. Got timeout=0 and port=8001")
}

func TestInvalidHostVendorID(t *testing.T) {
	tests := []struct {
		description  string
		vendorID     int
		wantErrorMsg string
	}{
		{
			description:  "Negative GDPR.HostVendorID",
			vendorID:     -1,
			wantErrorMsg: "gdpr.host_vendor_id must be in the range [0, 65535]. Got -1",
		},
		{
			description:  "Overflowed GDPR.HostVendorID",
			vendorID:     (0xffff) + 1,
			wantErrorMsg: "gdpr.host_vendor_id must be in the range [0, 65535]. Got 65536",
		},
	}

	for _, tt := range tests {
		cfg, v := newDefaultConfig(t)
		cfg.GDPR.HostVendorID = tt.vendorID
		errs := cfg.validate(v)

		assert.Equal(t, 1, len(errs), tt.description)
		assert.EqualError(t, errs[0], tt.wantErrorMsg, tt.description)
	}
}

func TestInvalidAMPException(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.GDPR.AMPException = true
	assertOneError(t, cfg.validate(v), "gdpr.amp_exception has been discontinued and must be removed from your config. If you need to disable GDPR for AMP, you may do so per-account (gdpr.integration_enabled.amp) or at the host level for the default account (account_defaults.gdpr.integration_enabled.amp)")
}

func TestInvalidGDPRDefaultValue(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.GDPR.DefaultValue = "2"
	assertOneError(t, cfg.validate(v), "gdpr.default_value must be 0 or 1")
}

func TestMissingGDPRDefaultValue(t *testing.T) {
	v := viper.New()

	cfg, _ := newDefaultConfig(t)
	assertOneError(t, cfg.validate(v), "gdpr.default_value is required and must be specified")
}

func TestInvalidEnforceAlgo(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.GDPR.TCF2.Purpose1.EnforceAlgo = ""
	cfg.GDPR.TCF2.Purpose2.EnforceAlgo = TCF2EnforceAlgoFull
	cfg.GDPR.TCF2.Purpose3.EnforceAlgo = TCF2EnforceAlgoBasic
	cfg.GDPR.TCF2.Purpose4.EnforceAlgo = TCF2EnforceAlgoFull
	cfg.GDPR.TCF2.Purpose5.EnforceAlgo = "invalid1"
	cfg.GDPR.TCF2.Purpose6.EnforceAlgo = "invalid2"
	cfg.GDPR.TCF2.Purpose7.EnforceAlgo = TCF2EnforceAlgoFull
	cfg.GDPR.TCF2.Purpose8.EnforceAlgo = TCF2EnforceAlgoBasic
	cfg.GDPR.TCF2.Purpose9.EnforceAlgo = TCF2EnforceAlgoFull
	cfg.GDPR.TCF2.Purpose10.EnforceAlgo = "invalid3"

	errs := cfg.validate(v)

	expectedErrs := []error{
		errors.New("gdpr.tcf2.purpose1.enforce_algo must be \"basic\" or \"full\". Got "),
		errors.New("gdpr.tcf2.purpose5.enforce_algo must be \"basic\" or \"full\". Got invalid1"),
		errors.New("gdpr.tcf2.purpose6.enforce_algo must be \"basic\" or \"full\". Got invalid2"),
		errors.New("gdpr.tcf2.purpose10.enforce_algo must be \"basic\" or \"full\". Got invalid3"),
	}
	assert.ElementsMatch(t, errs, expectedErrs, "gdpr.tcf2.purposeX.enforce_algo should prevent invalid values but it doesn't")
}

func TestNegativeCurrencyConverterFetchInterval(t *testing.T) {
	v := viper.New()
	v.Set("gdpr.default_value", "0")

	cfg := Configuration{
		CurrencyConverter: CurrencyConverter{
			FetchIntervalSeconds: -1,
		},
	}
	err := cfg.validate(v)
	assert.NotNil(t, err, "cfg.currency_converter.fetch_interval_seconds should prevent negative values, but it doesn't")
}

func TestOverflowedCurrencyConverterFetchInterval(t *testing.T) {
	v := viper.New()
	v.Set("gdpr.default_value", "0")

	cfg := Configuration{
		CurrencyConverter: CurrencyConverter{
			FetchIntervalSeconds: (0xffff) + 1,
		},
	}
	err := cfg.validate(v)
	assert.NotNil(t, err, "cfg.currency_converter.fetch_interval_seconds prevent values over %d, but it doesn't", 0xffff)
}

func TestLimitTimeout(t *testing.T) {
	doTimeoutTest(t, 10, 15, 10, 0)
	doTimeoutTest(t, 10, 0, 10, 0)
	doTimeoutTest(t, 5, 5, 10, 0)
	doTimeoutTest(t, 15, 15, 0, 0)
	doTimeoutTest(t, 15, 0, 20, 15)
}

func TestCookieSizeError(t *testing.T) {
	testCases := []struct {
		description string
		cookieSize  int
		expectError bool
	}{
		{"MIN_COOKIE_SIZE_BYTES + 1", MIN_COOKIE_SIZE_BYTES + 1, false},
		{"MIN_COOKIE_SIZE_BYTES", MIN_COOKIE_SIZE_BYTES, false},
		{"MIN_COOKIE_SIZE_BYTES - 1", MIN_COOKIE_SIZE_BYTES - 1, true},
		{"Zero", 0, false},
		{"Negative", -100, true},
	}

	for _, test := range testCases {
		resultErr := isValidCookieSize(test.cookieSize)

		if test.expectError {
			assert.Error(t, resultErr, test.description)
		} else {
			assert.NoError(t, resultErr, test.description)
		}
	}
}

func TestNewCallsRequestValidation(t *testing.T) {
	testCases := []struct {
		description       string
		privateIPNetworks string
		expectedError     string
		expectedIPs       []net.IPNet
	}{
		{
			description:       "Valid",
			privateIPNetworks: `["1.1.1.0/24"]`,
			expectedIPs:       []net.IPNet{{IP: net.IP{1, 1, 1, 0}, Mask: net.CIDRMask(24, 32)}},
		},
		{
			description:       "Invalid",
			privateIPNetworks: `["1"]`,
			expectedError:     "Invalid private IPv4 networks: '1'",
		},
	}

	for _, test := range testCases {
		v := viper.New()
		SetupViper(v, "", bidderInfos)
		v.Set("gdpr.default_value", "0")
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer([]byte(
			`request_validation:
    ipv4_private_networks: ` + test.privateIPNetworks)))

		result, resultErr := New(v, bidderInfos, mockNormalizeBidderName)

		if test.expectedError == "" {
			assert.NoError(t, resultErr, test.description+":err")
			assert.ElementsMatch(t, test.expectedIPs, result.RequestValidation.IPv4PrivateNetworksParsed, test.description+":parsed")
		} else {
			assert.Error(t, resultErr, test.description+":err")
		}
	}
}

func TestValidateDebug(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.Debug.TimeoutNotification.SamplingRate = 1.1

	err := cfg.validate(v)
	assert.NotNil(t, err, "cfg.debug.timeout_notification.sampling_rate should not be allowed to be greater than 1.0, but it was allowed")
}

func TestValidateAccountsConfigRestrictions(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.Accounts.Files.Enabled = true
	cfg.Accounts.HTTP.Endpoint = "http://localhost"
	cfg.Accounts.Database.ConnectionInfo.Database = "accounts"

	errs := cfg.validate(v)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs, errors.New("accounts.database: retrieving accounts via database not available, use accounts.files"))
}

func newDefaultConfig(t *testing.T) (*Configuration, *viper.Viper) {
	v := viper.New()
	SetupViper(v, "", bidderInfos)
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	cfg, err := New(v, bidderInfos, mockNormalizeBidderName)
	assert.NoError(t, err, "Setting up config should work but it doesn't")
	return cfg, v
}

func assertOneError(t *testing.T, errs []error, message string) {
	if !assert.Len(t, errs, 1) {
		return
	}
	assert.EqualError(t, errs[0], message)
}

func doTimeoutTest(t *testing.T, expected int, requested int, max uint64, def uint64) {
	t.Helper()
	cfg := AuctionTimeouts{
		Default: def,
		Max:     max,
	}
	expectedDuration := time.Duration(expected) * time.Millisecond
	limited := cfg.LimitAuctionTimeout(time.Duration(requested) * time.Millisecond)
	assert.Equal(t, limited, expectedDuration, "Expected %dms timeout, got %dms", expectedDuration, limited/time.Millisecond)
}

func TestSpecialFeature1VendorExceptionMap(t *testing.T) {
	baseConfig := []byte(`
    gdpr:
      default_value: 0
      tcf2:
        special_feature1:
          vendor_exceptions: `)

	tests := []struct {
		description             string
		configVendorExceptions  []byte
		wantVendorExceptions    []openrtb_ext.BidderName
		wantVendorExceptionsMap map[openrtb_ext.BidderName]struct{}
	}{
		{
			description:             "nil vendor exceptions",
			configVendorExceptions:  []byte(`null`),
			wantVendorExceptions:    []openrtb_ext.BidderName{},
			wantVendorExceptionsMap: map[openrtb_ext.BidderName]struct{}{},
		},
		{
			description:             "no vendor exceptions",
			configVendorExceptions:  []byte(`[]`),
			wantVendorExceptions:    []openrtb_ext.BidderName{},
			wantVendorExceptionsMap: map[openrtb_ext.BidderName]struct{}{},
		},
		{
			description:             "one vendor exception",
			configVendorExceptions:  []byte(`["vendor1"]`),
			wantVendorExceptions:    []openrtb_ext.BidderName{openrtb_ext.BidderName("vendor1")},
			wantVendorExceptionsMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("vendor1"): {}},
		},
		{
			description:             "many exceptions",
			configVendorExceptions:  []byte(`["vendor1","vendor2"]`),
			wantVendorExceptions:    []openrtb_ext.BidderName{openrtb_ext.BidderName("vendor1"), openrtb_ext.BidderName("vendor2")},
			wantVendorExceptionsMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("vendor1"): {}, openrtb_ext.BidderName("vendor2"): {}},
		},
	}

	for _, tt := range tests {
		config := append(baseConfig, tt.configVendorExceptions...)

		v := viper.New()
		SetupViper(v, "", bidderInfos)
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer(config))
		cfg, err := New(v, bidderInfos, mockNormalizeBidderName)
		assert.NoError(t, err, "Setting up config error", tt.description)

		assert.Equal(t, tt.wantVendorExceptions, cfg.GDPR.TCF2.SpecialFeature1.VendorExceptions, tt.description)
		assert.Equal(t, tt.wantVendorExceptionsMap, cfg.GDPR.TCF2.SpecialFeature1.VendorExceptionMap, tt.description)
	}
}

func TestSetConfigBidderInfoNillableFields(t *testing.T) {
	falseValue := false
	trueValue := true

	bidder1ConfigFalses := []byte(`
    adapters:
      bidder1:
        disabled: false
        modifyingVastXmlAllowed: false`)
	bidder1ConfigTrues := []byte(`
    adapters:
      bidder1:
        disabled: true
        modifyingVastXmlAllowed: true`)
	bidder1ConfigNils := []byte(`
    adapters:
      bidder1:
        disabled: null
        modifyingVastXmlAllowed: null`)
	bidder1Bidder2ConfigMixed := []byte(`
    adapters:
      bidder1:
        disabled: true
        modifyingVastXmlAllowed: false
      bidder2:
        disabled: false
        modifyingVastXmlAllowed: true`)

	tests := []struct {
		name        string
		rawConfig   []byte
		bidderInfos BidderInfos
		expected    nillableFieldBidderInfos
		expectError bool
	}{
		{
			name:     "viper and bidder infos are nil",
			expected: nil,
		},
		{
			name:        "viper is nil",
			bidderInfos: map[string]BidderInfo{},
			expected:    nil,
		},
		{
			name:      "bidder infos is nil",
			rawConfig: []byte{},
			expected:  nil,
		},
		{
			name:        "bidder infos is empty",
			bidderInfos: map[string]BidderInfo{},
			expected:    nil,
		},
		{
			name: "one: bidder info has nillable fields as false, viper has as nil",
			bidderInfos: map[string]BidderInfo{
				"bidder1": {Disabled: false, ModifyingVastXmlAllowed: false},
			},
			rawConfig: bidder1ConfigNils,
			expected: nillableFieldBidderInfos{
				"bidder1": nillableFieldBidderInfo{
					nillableFields: bidderInfoNillableFields{
						Disabled:                nil,
						ModifyingVastXmlAllowed: nil,
					},
					bidderInfo: BidderInfo{Disabled: false, ModifyingVastXmlAllowed: false},
				},
			},
		},
		{
			name: "one: bidder info has nillable fields as false, viper has as false",
			bidderInfos: map[string]BidderInfo{
				"bidder1": {Disabled: false, ModifyingVastXmlAllowed: false},
			},
			rawConfig: bidder1ConfigFalses,
			expected: nillableFieldBidderInfos{
				"bidder1": nillableFieldBidderInfo{
					nillableFields: bidderInfoNillableFields{
						Disabled:                &falseValue,
						ModifyingVastXmlAllowed: &falseValue,
					},
					bidderInfo: BidderInfo{Disabled: false, ModifyingVastXmlAllowed: false},
				},
			},
		},
		{
			name: "one: bidder info has nillable fields as false, viper has as true",
			bidderInfos: map[string]BidderInfo{
				"bidder1": {Disabled: false, ModifyingVastXmlAllowed: false},
			},
			rawConfig: bidder1ConfigTrues,
			expected: nillableFieldBidderInfos{
				"bidder1": nillableFieldBidderInfo{
					nillableFields: bidderInfoNillableFields{
						Disabled:                &trueValue,
						ModifyingVastXmlAllowed: &trueValue,
					},
					bidderInfo: BidderInfo{Disabled: false, ModifyingVastXmlAllowed: false},
				},
			},
		},
		{
			name: "many with extra info: bidder infos have nillable fields as false and true, viper has as true and false",
			bidderInfos: map[string]BidderInfo{
				"bidder1": {Disabled: false, ModifyingVastXmlAllowed: true, Endpoint: "endpoint a"},
				"bidder2": {Disabled: true, ModifyingVastXmlAllowed: false, Endpoint: "endpoint b"},
			},
			rawConfig: bidder1Bidder2ConfigMixed,
			expected: nillableFieldBidderInfos{
				"bidder1": nillableFieldBidderInfo{
					nillableFields: bidderInfoNillableFields{
						Disabled:                &trueValue,
						ModifyingVastXmlAllowed: &falseValue,
					},
					bidderInfo: BidderInfo{Disabled: false, ModifyingVastXmlAllowed: true, Endpoint: "endpoint a"},
				},
				"bidder2": nillableFieldBidderInfo{
					nillableFields: bidderInfoNillableFields{
						Disabled:                &falseValue,
						ModifyingVastXmlAllowed: &trueValue,
					},
					bidderInfo: BidderInfo{Disabled: true, ModifyingVastXmlAllowed: false, Endpoint: "endpoint b"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := viper.New()
			v.SetConfigType("yaml")
			for bidderName := range tt.bidderInfos {
				setBidderDefaults(v, strings.ToLower(bidderName))
			}
			v.ReadConfig(bytes.NewBuffer(tt.rawConfig))

			result, err := setConfigBidderInfoNillableFields(v, tt.bidderInfos)

			assert.Equal(t, tt.expected, result)
			if tt.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}

func TestTCF2PurposeEnforced(t *testing.T) {
	tests := []struct {
		description          string
		givePurposeConfigNil bool
		givePurpose1Enforced bool
		givePurpose2Enforced bool
		givePurpose          consentconstants.Purpose
		wantEnforced         bool
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			wantEnforced:         false,
		},
		{
			description:          "Purpose 1 Enforced set to true",
			givePurpose1Enforced: true,
			givePurpose:          1,
			wantEnforced:         true,
		},
		{
			description:          "Purpose 1 Enforced set to false",
			givePurpose1Enforced: false,
			givePurpose:          1,
			wantEnforced:         false,
		},
		{
			description:          "Purpose 2 Enforced set to true",
			givePurpose2Enforced: true,
			givePurpose:          2,
			wantEnforced:         true,
		},
	}

	for _, tt := range tests {
		tcf2 := TCF2{}

		if !tt.givePurposeConfigNil {
			tcf2.PurposeConfigs = map[consentconstants.Purpose]*TCF2Purpose{
				1: {
					EnforcePurpose: tt.givePurpose1Enforced,
				},
				2: {
					EnforcePurpose: tt.givePurpose2Enforced,
				},
			}
		}

		value := tcf2.PurposeEnforced(tt.givePurpose)

		assert.Equal(t, tt.wantEnforced, value, tt.description)
	}
}

func TestTCF2PurposeEnforcementAlgo(t *testing.T) {
	tests := []struct {
		description          string
		givePurposeConfigNil bool
		givePurpose1Algo     TCF2EnforcementAlgo
		givePurpose2Algo     TCF2EnforcementAlgo
		givePurpose          consentconstants.Purpose
		wantAlgo             TCF2EnforcementAlgo
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			wantAlgo:             TCF2FullEnforcement,
		},
		{
			description:      "Purpose 1 enforcement algo set to basic",
			givePurpose1Algo: TCF2BasicEnforcement,
			givePurpose:      1,
			wantAlgo:         TCF2BasicEnforcement,
		},
		{
			description:      "Purpose 1 enforcement algo set to full",
			givePurpose1Algo: TCF2FullEnforcement,
			givePurpose:      1,
			wantAlgo:         TCF2FullEnforcement,
		},
		{
			description:      "Purpose 2 Enforcement algo set to basic",
			givePurpose2Algo: TCF2BasicEnforcement,
			givePurpose:      2,
			wantAlgo:         TCF2BasicEnforcement,
		},
	}

	for _, tt := range tests {
		tcf2 := TCF2{}

		if !tt.givePurposeConfigNil {
			tcf2.PurposeConfigs = map[consentconstants.Purpose]*TCF2Purpose{
				1: {
					EnforceAlgoID: tt.givePurpose1Algo,
				},
				2: {
					EnforceAlgoID: tt.givePurpose2Algo,
				},
			}
		}

		value := tcf2.PurposeEnforcementAlgo(tt.givePurpose)

		assert.Equal(t, tt.wantAlgo, value, tt.description)
	}
}

func TestTCF2PurposeEnforcingVendors(t *testing.T) {
	tests := []struct {
		description           string
		givePurposeConfigNil  bool
		givePurpose1Enforcing bool
		givePurpose2Enforcing bool
		givePurpose           consentconstants.Purpose
		wantEnforcing         bool
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			wantEnforcing:        false,
		},
		{
			description:           "Purpose 1 Enforcing set to true",
			givePurpose1Enforcing: true,
			givePurpose:           1,
			wantEnforcing:         true,
		},
		{
			description:           "Purpose 1 Enforcing set to false",
			givePurpose1Enforcing: false,
			givePurpose:           1,
			wantEnforcing:         false,
		},
		{
			description:           "Purpose 2 Enforcing set to true",
			givePurpose2Enforcing: true,
			givePurpose:           2,
			wantEnforcing:         true,
		},
	}

	for _, tt := range tests {
		tcf2 := TCF2{}

		if !tt.givePurposeConfigNil {
			tcf2.PurposeConfigs = map[consentconstants.Purpose]*TCF2Purpose{
				1: {
					EnforceVendors: tt.givePurpose1Enforcing,
				},
				2: {
					EnforceVendors: tt.givePurpose2Enforcing,
				},
			}
		}

		value := tcf2.PurposeEnforcingVendors(tt.givePurpose)

		assert.Equal(t, tt.wantEnforcing, value, tt.description)
	}
}

func TestTCF2PurposeVendorExceptions(t *testing.T) {
	tests := []struct {
		description              string
		givePurposeConfigNil     bool
		givePurpose1ExceptionMap map[string]struct{}
		givePurpose2ExceptionMap map[string]struct{}
		givePurpose              consentconstants.Purpose
		wantExceptionMap         map[string]struct{}
	}{
		{
			description:          "Purpose config is nil",
			givePurposeConfigNil: true,
			givePurpose:          1,
			wantExceptionMap:     map[string]struct{}{},
		},
		{
			description:      "Nil - exception map not defined for purpose",
			givePurpose:      1,
			wantExceptionMap: map[string]struct{}{},
		},
		{
			description:              "Empty - exception map empty for purpose",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[string]struct{}{},
			wantExceptionMap:         map[string]struct{}{},
		},
		{
			description:              "Nonempty - exception map with multiple entries for purpose",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[string]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			wantExceptionMap:         map[string]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
		},
		{
			description:              "Nonempty - exception map with multiple entries for different purpose",
			givePurpose:              2,
			givePurpose1ExceptionMap: map[string]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			givePurpose2ExceptionMap: map[string]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
			wantExceptionMap:         map[string]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
		},
	}

	for _, tt := range tests {
		tcf2 := TCF2{}

		if !tt.givePurposeConfigNil {
			tcf2.PurposeConfigs = map[consentconstants.Purpose]*TCF2Purpose{
				1: {
					VendorExceptionMap: tt.givePurpose1ExceptionMap,
				},
				2: {
					VendorExceptionMap: tt.givePurpose2ExceptionMap,
				},
			}
		}

		value := tcf2.PurposeVendorExceptions(tt.givePurpose)

		assert.Equal(t, tt.wantExceptionMap, value, tt.description)
	}
}

func TestTCF2FeatureOneVendorException(t *testing.T) {
	tests := []struct {
		description           string
		giveExceptionMap      map[openrtb_ext.BidderName]struct{}
		giveBidder            openrtb_ext.BidderName
		wantIsVendorException bool
	}{
		{
			description:           "Nil - exception map not defined",
			giveBidder:            "appnexus",
			wantIsVendorException: false,
		},
		{
			description:           "Empty - exception map empty",
			giveExceptionMap:      map[openrtb_ext.BidderName]struct{}{},
			giveBidder:            "appnexus",
			wantIsVendorException: false,
		},
		{
			description:           "One - bidder found in exception map containing one entry",
			giveExceptionMap:      map[openrtb_ext.BidderName]struct{}{"appnexus": {}},
			giveBidder:            "appnexus",
			wantIsVendorException: true,
		},
		{
			description:           "Many - bidder found in exception map containing multiple entries",
			giveExceptionMap:      map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:            "appnexus",
			wantIsVendorException: true,
		},
		{
			description:           "Many - bidder not found in exception map containing multiple entries",
			giveExceptionMap:      map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:            "openx",
			wantIsVendorException: false,
		},
	}

	for _, tt := range tests {
		tcf2 := TCF2{
			SpecialFeature1: TCF2SpecialFeature{
				VendorExceptionMap: tt.giveExceptionMap,
			},
		}

		value := tcf2.FeatureOneVendorException(tt.giveBidder)

		assert.Equal(t, tt.wantIsVendorException, value, tt.description)
	}
}

func TestUnpackDSADefault(t *testing.T) {
	tests := []struct {
		name      string
		giveDSA   *AccountDSA
		wantError bool
	}{
		{
			name:      "nil",
			giveDSA:   nil,
			wantError: false,
		},
		{
			name: "empty",
			giveDSA: &AccountDSA{
				Default: "",
			},
			wantError: false,
		},
		{
			name: "empty_json",
			giveDSA: &AccountDSA{
				Default: "{}",
			},
			wantError: false,
		},
		{
			name: "well_formed",
			giveDSA: &AccountDSA{
				Default: "{\"dsarequired\":3,\"pubrender\":1,\"datatopub\":2,\"transparency\":[{\"domain\":\"domain.com\",\"dsaparams\":[1]}]}",
			},
			wantError: false,
		},
		{
			name: "well_formed_with_extra_fields",
			giveDSA: &AccountDSA{
				Default: "{\"unmappedkey\":\"unmappedvalue\",\"dsarequired\":3,\"pubrender\":1,\"datatopub\":2,\"transparency\":[{\"domain\":\"domain.com\",\"dsaparams\":[1]}]}",
			},
			wantError: false,
		},
		{
			name: "invalid_type",
			giveDSA: &AccountDSA{
				Default: "{\"dsarequired\":\"invalid\",\"pubrender\":1,\"datatopub\":2,\"transparency\":[{\"domain\":\"domain.com\",\"dsaparams\":[1]}]}",
			},
			wantError: true,
		},
		{
			name: "invalid_malformed_missing_colon",
			giveDSA: &AccountDSA{
				Default: "{\"dsarequired\"3,\"pubrender\":1,\"datatopub\":2,\"transparency\":[{\"domain\":\"domain.com\",\"dsaparams\":[1]}]}",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := UnpackDSADefault(tt.giveDSA)
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
