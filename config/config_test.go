package config

import (
	"bytes"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"encoding/json"

	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

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

	cmpInts(t, "port", cfg.Port, 8000)
	cmpInts(t, "admin_port", cfg.AdminPort, 6060)
	cmpInts(t, "auction_timeouts_ms.max", int(cfg.AuctionTimeouts.Max), 0)
	cmpInts(t, "max_request_size", int(cfg.MaxRequestSize), 1024*256)
	cmpInts(t, "host_cookie.ttl_days", int(cfg.HostCookie.TTL), 90)
	cmpInts(t, "host_cookie.max_cookie_size_bytes", cfg.HostCookie.MaxCookieSizeBytes, 0)
	cmpStrings(t, "adapters.pubmatic.endpoint", cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint, "https://hbopenbid.pubmatic.com/translator?source=prebid-server")
	cmpInts(t, "currency_converter.fetch_interval_seconds", cfg.CurrencyConverter.FetchIntervalSeconds, 1800)
	cmpStrings(t, "currency_converter.fetch_url", cfg.CurrencyConverter.FetchURL, "https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json")
	cmpBools(t, "account_required", cfg.AccountRequired, false)
	cmpInts(t, "metrics.influxdb.collection_rate_seconds", cfg.Metrics.Influxdb.MetricSendInterval, 20)
	cmpBools(t, "account_adapter_details", cfg.Metrics.Disabled.AccountAdapterDetails, false)
	cmpBools(t, "adapter_connections_metrics", cfg.Metrics.Disabled.AdapterConnectionMetrics, true)
	cmpBools(t, "adapter_gdpr_request_blocked", cfg.Metrics.Disabled.AdapterGDPRRequestBlocked, false)
	cmpStrings(t, "certificates_file", cfg.PemCertsFile, "")
	cmpBools(t, "stored_requests.filesystem.enabled", false, cfg.StoredRequests.Files.Enabled)
	cmpStrings(t, "stored_requests.filesystem.directorypath", "./stored_requests/data/by_id", cfg.StoredRequests.Files.Path)
	cmpBools(t, "auto_gen_source_tid", cfg.AutoGenSourceTID, true)
	cmpBools(t, "generate_bid_id", cfg.GenerateBidID, false)

	//Assert purpose VendorExceptionMap hash tables were built correctly
	expectedTCF2 := TCF2{
		Enabled: true,
		Purpose1: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose2: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose3: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose4: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose5: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose6: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose7: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose8: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose9: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose10: TCF2Purpose{
			Enabled:            true,
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		SpecialPurpose1: TCF2Purpose{
			Enabled:            true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		PurposeOneTreatment: TCF2PurposeOneTreatment{
			Enabled:       true,
			AccessAllowed: true,
		},
	}
	assert.Equal(t, expectedTCF2, cfg.GDPR.TCF2, "gdpr.tcf2")

	// Assert User Sync Override Defaults To Nil
	assert.Nil(t, cfg.Adapters["appnexus"].Syncer, "User Sync")
}

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
      enabled: false
      enforce_purpose: "no"
      enforce_vendors: false
      vendor_exceptions: ["foo2"]
    purpose3:
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
    special_purpose1:
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
garbage_collector_threshold: 1
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
    username: admin
    password: admin1324
    metric_send_interval: 30
  disabled_metrics:
    account_adapter_details: true
    adapter_connections_metrics: true
    adapter_gdpr_request_blocked: true
adapters:
  appnexus:
    endpoint: http://ib.adnxs.com/some/endpoint
    extra_info: "{\"native\":\"http://www.native.org/endpoint\",\"video\":\"http://www.video.org/endpoint\"}"
  audienceNetwork:
    endpoint: http://facebook.com/pbs
    usersync_url: http://facebook.com/ortb/prebid-s2s
    platform_id: abcdefgh1234
    app_secret: 987abc
  ix:
    endpoint: http://ixtest.com/api
  rubicon:
    endpoint: http://rubitest.com/api
    xapi:
      username: rubiuser
      password: rubipw23
    usersync:
      redirect:
        url: http://rubitest.com/sync
        user_macro: "{UID}"
  brightroll:
    usersync_url: http://test-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&url=%s
    endpoint: http://test-bid.ybp.yahoo.com/bid/appnexuspbs
  adkerneladn:
     usersync_url: https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=
blacklisted_apps: ["spamAppID","sketchy-app-id"]
account_required: true
auto_gen_source_tid: false
certificates_file: /etc/ssl/cert.pem
request_validation:
    ipv4_private_networks: ["1.1.1.0/24"]
    ipv6_private_networks: ["1111::/16", "2222::/16"]
generate_bid_id: true
`)

var adapterExtraInfoConfig = []byte(`
adapters:
  appnexus:
    endpoint: http://ib.adnxs.com/some/endpoint
    usersync_url: http://adnxs.com/sync.php?p=prebid
    platform_id: appNexus
    xapi:
      username: appuser
      password: 123456
      tracker: anxsTrack
    disabled: true
    extra_info: "{\"native\":\"http://www.native.org/endpoint\",\"video\":\"http://www.video.org/endpoint\"}"
`)

var invalidAdapterEndpointConfig = []byte(`
adapters:
  appnexus:
    endpoint: ib.adnxs.com/some/endpoint
  brightroll:
    usersync:
      redirect:
      url: http://http://test-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&url=%s
`)

var oldStoredRequestsConfig = []byte(`
stored_requests:
  filesystem: true
  directorypath: "/somepath"
`)

func cmpStrings(t *testing.T, key string, a string, b string) {
	t.Helper()
	assert.Equal(t, a, b, "%s: %s != %s", key, a, b)
}

func cmpInts(t *testing.T, key string, a int, b int) {
	t.Helper()
	assert.Equal(t, a, b, "%s: %d != %d", key, a, b)
}

func cmpBools(t *testing.T, key string, a bool, b bool) {
	t.Helper()
	assert.Equal(t, a, b, "%s: %t != %t", key, a, b)
}

func cmpNils(t *testing.T, key string, a interface{}) {
	t.Helper()
	assert.Nilf(t, a, "%s: %t != nil", key, a)
}

func TestFullConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(fullConfig))
	cfg, err := New(v)
	assert.NoError(t, err, "Setting up config should work but it doesn't")
	cmpStrings(t, "cookie domain", cfg.HostCookie.Domain, "cookies.prebid.org")
	cmpStrings(t, "cookie name", cfg.HostCookie.CookieName, "userid")
	cmpStrings(t, "cookie family", cfg.HostCookie.Family, "prebid")
	cmpStrings(t, "opt out", cfg.HostCookie.OptOutURL, "http://prebid.org/optout")
	cmpStrings(t, "opt in", cfg.HostCookie.OptInURL, "http://prebid.org/optin")
	cmpStrings(t, "external url", cfg.ExternalURL, "http://prebid-server.prebid.org/")
	cmpStrings(t, "host", cfg.Host, "prebid-server.prebid.org")
	cmpInts(t, "port", cfg.Port, 1234)
	cmpInts(t, "admin_port", cfg.AdminPort, 5678)
	cmpInts(t, "garbage_collector_threshold", cfg.GarbageCollectorThreshold, 1)
	cmpInts(t, "auction_timeouts_ms.default", int(cfg.AuctionTimeouts.Default), 50)
	cmpInts(t, "auction_timeouts_ms.max", int(cfg.AuctionTimeouts.Max), 123)
	cmpStrings(t, "cache.scheme", cfg.CacheURL.Scheme, "http")
	cmpStrings(t, "cache.host", cfg.CacheURL.Host, "prebidcache.net")
	cmpStrings(t, "cache.query", cfg.CacheURL.Query, "uuid=%PBS_CACHE_UUID%")
	cmpStrings(t, "external_cache.scheme", cfg.ExtCacheURL.Scheme, "https")
	cmpStrings(t, "external_cache.host", cfg.ExtCacheURL.Host, "www.externalprebidcache.net")
	cmpStrings(t, "external_cache.path", cfg.ExtCacheURL.Path, "/endpoints/cache")
	cmpInts(t, "http_client.max_connections_per_host", cfg.Client.MaxConnsPerHost, 10)
	cmpInts(t, "http_client.max_idle_connections", cfg.Client.MaxIdleConns, 500)
	cmpInts(t, "http_client.max_idle_connections_per_host", cfg.Client.MaxIdleConnsPerHost, 20)
	cmpInts(t, "http_client.idle_connection_timeout_seconds", cfg.Client.IdleConnTimeout, 30)
	cmpInts(t, "http_client_cache.max_connections_per_host", cfg.CacheClient.MaxConnsPerHost, 5)
	cmpInts(t, "http_client_cache.max_idle_connections", cfg.CacheClient.MaxIdleConns, 1)
	cmpInts(t, "http_client_cache.max_idle_connections_per_host", cfg.CacheClient.MaxIdleConnsPerHost, 2)
	cmpInts(t, "http_client_cache.idle_connection_timeout_seconds", cfg.CacheClient.IdleConnTimeout, 3)
	cmpInts(t, "gdpr.host_vendor_id", cfg.GDPR.HostVendorID, 15)
	cmpStrings(t, "gdpr.default_value", cfg.GDPR.DefaultValue, "1")

	//Assert the NonStandardPublishers was correctly unmarshalled
	assert.Equal(t, []string{"pub1", "pub2"}, cfg.GDPR.NonStandardPublishers, "gdpr.non_standard_publishers")
	assert.Equal(t, map[string]struct{}{"pub1": {}, "pub2": {}}, cfg.GDPR.NonStandardPublisherMap, "gdpr.non_standard_publishers Hash Map")

	// Assert EEA Countries was correctly unmarshalled and the EEACountriesMap built correctly.
	assert.Equal(t, []string{"eea1", "eea2"}, cfg.GDPR.EEACountries, "gdpr.eea_countries")
	assert.Equal(t, map[string]struct{}{"eea1": {}, "eea2": {}}, cfg.GDPR.EEACountriesMap, "gdpr.eea_countries Hash Map")

	cmpBools(t, "ccpa.enforce", cfg.CCPA.Enforce, true)
	cmpBools(t, "lmt.enforce", cfg.LMT.Enforce, true)

	//Assert the NonStandardPublishers was correctly unmarshalled
	cmpStrings(t, "blacklisted_apps", cfg.BlacklistedApps[0], "spamAppID")
	cmpStrings(t, "blacklisted_apps", cfg.BlacklistedApps[1], "sketchy-app-id")

	//Assert the BlacklistedAppMap hash table was built correctly
	for i := 0; i < len(cfg.BlacklistedApps); i++ {
		cmpBools(t, "cfg.BlacklistedAppMap", cfg.BlacklistedAppMap[cfg.BlacklistedApps[i]], true)
	}

	//Assert purpose VendorExceptionMap hash tables were built correctly
	expectedTCF2 := TCF2{
		Enabled: true,
		Purpose1: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo1a"), openrtb_ext.BidderName("foo1b")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo1a"): {}, openrtb_ext.BidderName("foo1b"): {}},
		},
		Purpose2: TCF2Purpose{
			Enabled:            false,
			EnforcePurpose:     TCF2NoEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo2")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo2"): {}},
		},
		Purpose3: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo3")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo3"): {}},
		},
		Purpose4: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo4")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo4"): {}},
		},
		Purpose5: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo5")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo5"): {}},
		},
		Purpose6: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo6")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo6"): {}},
		},
		Purpose7: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo7")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo7"): {}},
		},
		Purpose8: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo8")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo8"): {}},
		},
		Purpose9: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo9")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo9"): {}},
		},
		Purpose10: TCF2Purpose{
			Enabled:            true, // true by default
			EnforcePurpose:     TCF2FullEnforcement,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo10")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo10"): {}},
		},
		SpecialPurpose1: TCF2Purpose{
			Enabled:            true, // true by default
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("fooSP1")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("fooSP1"): {}},
		},
		PurposeOneTreatment: TCF2PurposeOneTreatment{
			Enabled:       true, // true by default
			AccessAllowed: true, // true by default
		},
	}
	assert.Equal(t, expectedTCF2, cfg.GDPR.TCF2, "gdpr.tcf2")

	cmpStrings(t, "currency_converter.fetch_url", cfg.CurrencyConverter.FetchURL, "https://currency.prebid.org")
	cmpInts(t, "currency_converter.fetch_interval_seconds", cfg.CurrencyConverter.FetchIntervalSeconds, 1800)
	cmpStrings(t, "recaptcha_secret", cfg.RecaptchaSecret, "asdfasdfasdfasdf")
	cmpStrings(t, "metrics.influxdb.host", cfg.Metrics.Influxdb.Host, "upstream:8232")
	cmpStrings(t, "metrics.influxdb.database", cfg.Metrics.Influxdb.Database, "metricsdb")
	cmpStrings(t, "metrics.influxdb.username", cfg.Metrics.Influxdb.Username, "admin")
	cmpStrings(t, "metrics.influxdb.password", cfg.Metrics.Influxdb.Password, "admin1324")
	cmpInts(t, "metrics.influxdb.metric_send_interval", cfg.Metrics.Influxdb.MetricSendInterval, 30)
	cmpStrings(t, "", cfg.CacheURL.GetBaseURL(), "http://prebidcache.net")
	cmpStrings(t, "", cfg.GetCachedAssetURL("a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11"), "http://prebidcache.net/cache?uuid=a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11")
	cmpStrings(t, "adapters.appnexus.endpoint", cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, "http://ib.adnxs.com/some/endpoint")
	cmpStrings(t, "adapters.appnexus.extra_info", cfg.Adapters[string(openrtb_ext.BidderAppnexus)].ExtraAdapterInfo, "{\"native\":\"http://www.native.org/endpoint\",\"video\":\"http://www.video.org/endpoint\"}")
	cmpStrings(t, "adapters.audiencenetwork.endpoint", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].Endpoint, "http://facebook.com/pbs")
	cmpStrings(t, "adapters.audiencenetwork.platform_id", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].PlatformID, "abcdefgh1234")
	cmpStrings(t, "adapters.audiencenetwork.app_secret", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].AppSecret, "987abc")
	cmpStrings(t, "adapters.audiencenetwork.usersync_url", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].UserSyncURL, "http://facebook.com/ortb/prebid-s2s")
	cmpStrings(t, "adapters.beachfront.endpoint", cfg.Adapters[string(openrtb_ext.BidderBeachfront)].Endpoint, "https://display.bfmio.com/prebid_display")
	cmpStrings(t, "adapters.beachfront.extra_info", cfg.Adapters[string(openrtb_ext.BidderBeachfront)].ExtraAdapterInfo, "{\"video_endpoint\":\"https://reachms.bfmio.com/bid.json?exchange_id\"}")
	cmpStrings(t, "adapters.ix.endpoint", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIx))].Endpoint, "http://ixtest.com/api")
	cmpStrings(t, "adapters.rubicon.endpoint", cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint, "http://rubitest.com/api")
	cmpStrings(t, "adapters.rubicon.xapi.username", cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username, "rubiuser")
	cmpStrings(t, "adapters.rubicon.xapi.password", cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password, "rubipw23")
	cmpStrings(t, "adapters.rubicon.usersync.redirect.url", cfg.Adapters[string(openrtb_ext.BidderRubicon)].Syncer.Redirect.URL, "http://rubitest.com/sync")
	cmpNils(t, "adapters.rubicon.usersync.iframe", cfg.Adapters[string(openrtb_ext.BidderRubicon)].Syncer.IFrame)
	cmpStrings(t, "adapters.rubicon.usersync.redirect.user_macro", cfg.Adapters[string(openrtb_ext.BidderRubicon)].Syncer.Redirect.UserMacro, "{UID}")
	cmpStrings(t, "adapters.brightroll.endpoint", cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint, "http://test-bid.ybp.yahoo.com/bid/appnexuspbs")
	cmpStrings(t, "adapters.rhythmone.endpoint", cfg.Adapters[string(openrtb_ext.BidderRhythmone)].Endpoint, "http://tag.1rx.io/rmp")
	cmpBools(t, "account_required", cfg.AccountRequired, true)
	cmpBools(t, "auto_gen_source_tid", cfg.AutoGenSourceTID, false)
	cmpBools(t, "account_adapter_details", cfg.Metrics.Disabled.AccountAdapterDetails, true)
	cmpBools(t, "adapter_connections_metrics", cfg.Metrics.Disabled.AdapterConnectionMetrics, true)
	cmpBools(t, "adapter_gdpr_request_blocked", cfg.Metrics.Disabled.AdapterGDPRRequestBlocked, true)
	cmpStrings(t, "certificates_file", cfg.PemCertsFile, "/etc/ssl/cert.pem")
	cmpStrings(t, "request_validation.ipv4_private_networks", cfg.RequestValidation.IPv4PrivateNetworks[0], "1.1.1.0/24")
	cmpStrings(t, "request_validation.ipv6_private_networks", cfg.RequestValidation.IPv6PrivateNetworks[0], "1111::/16")
	cmpStrings(t, "request_validation.ipv6_private_networks", cfg.RequestValidation.IPv6PrivateNetworks[1], "2222::/16")
	cmpBools(t, "generate_bid_id", cfg.GenerateBidID, true)
	cmpStrings(t, "debug.override_token", cfg.Debug.OverrideToken, "")
}

func TestUnmarshalAdapterExtraInfo(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(adapterExtraInfoConfig))
	cfg, err := New(v)

	// Assert correctly unmarshaled
	assert.NoError(t, err, "invalid endpoint in config should return an error")

	// Assert JSON-formatted string
	assert.JSONEqf(t, `{"native":"http://www.native.org/endpoint","video":"http://www.video.org/endpoint"}`, cfg.Adapters[string(openrtb_ext.BidderAppnexus)].ExtraAdapterInfo, "Unexpected value of the ExtraAdapterInfo String \n")

	// Data type where we'll unmarshal endpoint values and adapter custom extra information
	type AppNexusAdapterEndpoints struct {
		NativeEndpoint string `json:"native,omitempty"`
		VideoEndpoint  string `json:"video,omitempty"`
	}
	var AppNexusAdapterExtraInfo AppNexusAdapterEndpoints
	err = json.Unmarshal([]byte(cfg.Adapters[string(openrtb_ext.BidderAppnexus)].ExtraAdapterInfo), &AppNexusAdapterExtraInfo)

	// Assert correctly unmarshaled
	assert.NoErrorf(t, err, "Error. Could not unmarshal cfg.Adapters[string(openrtb_ext.BidderAppnexus)].ExtraAdapterInfo. Value: %s. Error: %v \n", cfg.Adapters[string(openrtb_ext.BidderAppnexus)].ExtraAdapterInfo, err)

	// Assert endpoint values
	assert.Equal(t, "http://www.native.org/endpoint", AppNexusAdapterExtraInfo.NativeEndpoint)
	assert.Equal(t, "http://www.video.org/endpoint", AppNexusAdapterExtraInfo.VideoEndpoint)
}

func TestValidateConfig(t *testing.T) {
	cfg := Configuration{
		GDPR: GDPR{
			DefaultValue: "1",
			TCF2: TCF2{
				Purpose1:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose2:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose3:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose4:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose5:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose6:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose7:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose8:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose9:  TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
				Purpose10: TCF2Purpose{EnforcePurpose: TCF2FullEnforcement},
			},
		},
		StoredRequests: StoredRequests{
			Files: FileFetcherConfig{Enabled: true},
			InMemoryCache: InMemoryCache{
				Type: "none",
			},
		},
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
	}

	v := viper.New()
	v.Set("gdpr.default_value", "0")

	resolvedStoredRequestsConfig(&cfg)
	err := cfg.validate(v)
	assert.Nil(t, err, "OpenRTB filesystem config should work. %v", err)
}

func TestMigrateConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(oldStoredRequestsConfig))
	migrateConfig(v)
	cfg, err := New(v)
	assert.NoError(t, err, "Setting up config should work but it doesn't")
	cmpBools(t, "stored_requests.filesystem.enabled", true, cfg.StoredRequests.Files.Enabled)
	cmpStrings(t, "stored_requests.filesystem.path", "/somepath", cfg.StoredRequests.Files.Path)
}

func TestMigrateConfigFromEnv(t *testing.T) {
	if oldval, ok := os.LookupEnv("PBS_STORED_REQUESTS_FILESYSTEM"); ok {
		defer os.Setenv("PBS_STORED_REQUESTS_FILESYSTEM", oldval)
	} else {
		defer os.Unsetenv("PBS_STORED_REQUESTS_FILESYSTEM")
	}
	os.Setenv("PBS_STORED_REQUESTS_FILESYSTEM", "true")
	cfg, _ := newDefaultConfig(t)
	cmpBools(t, "stored_requests.filesystem.enabled", true, cfg.StoredRequests.Files.Enabled)
}

func TestMigrateConfigPurposeOneTreatment(t *testing.T) {
	oldPurposeOneTreatmentConfig := []byte(`
      gdpr:
        tcf2:
          purpose_one_treatement:
            enabled: true
            access_allowed: true
    `)
	newPurposeOneTreatmentConfig := []byte(`
      gdpr:
        tcf2:
          purpose_one_treatment:
            enabled: true
            access_allowed: true
    `)
	oldAndNewPurposeOneTreatmentConfig := []byte(`
      gdpr:
        tcf2:
          purpose_one_treatement:
            enabled: false
            access_allowed: true
          purpose_one_treatment:
            enabled: true
            access_allowed: false
    `)

	tests := []struct {
		description                        string
		config                             []byte
		wantPurpose1TreatmentEnabled       bool
		wantPurpose1TreatmentAccessAllowed bool
	}{
		{
			description: "New config and old config not set",
			config:      []byte{},
		},
		{
			description:                        "New config not set, old config set",
			config:                             oldPurposeOneTreatmentConfig,
			wantPurpose1TreatmentEnabled:       true,
			wantPurpose1TreatmentAccessAllowed: true,
		},
		{
			description:                        "New config set, old config not set",
			config:                             newPurposeOneTreatmentConfig,
			wantPurpose1TreatmentEnabled:       true,
			wantPurpose1TreatmentAccessAllowed: true,
		},
		{
			description:                        "New config and old config set",
			config:                             oldAndNewPurposeOneTreatmentConfig,
			wantPurpose1TreatmentEnabled:       true,
			wantPurpose1TreatmentAccessAllowed: false,
		},
	}

	for _, tt := range tests {
		v := viper.New()
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer(tt.config))

		migrateConfigPurposeOneTreatment(v)

		if len(tt.config) > 0 {
			assert.Equal(t, tt.wantPurpose1TreatmentEnabled, v.Get("gdpr.tcf2.purpose_one_treatment.enabled").(bool), tt.description)
			assert.Equal(t, tt.wantPurpose1TreatmentAccessAllowed, v.Get("gdpr.tcf2.purpose_one_treatment.access_allowed").(bool), tt.description)
		} else {
			assert.Nil(t, v.Get("gdpr.tcf2.purpose_one_treatment.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose_one_treatment.access_allowed"), tt.description)
		}
	}
}

func TestMigrateConfigTCF2PurposeEnabledFlags(t *testing.T) {
	trueStr := "true"
	falseStr := "false"

	tests := []struct {
		description                 string
		config                      []byte
		wantPurpose1EnforcePurpose  string
		wantPurpose2EnforcePurpose  string
		wantPurpose3EnforcePurpose  string
		wantPurpose4EnforcePurpose  string
		wantPurpose5EnforcePurpose  string
		wantPurpose6EnforcePurpose  string
		wantPurpose7EnforcePurpose  string
		wantPurpose8EnforcePurpose  string
		wantPurpose9EnforcePurpose  string
		wantPurpose10EnforcePurpose string
		wantPurpose1Enabled         string
		wantPurpose2Enabled         string
		wantPurpose3Enabled         string
		wantPurpose4Enabled         string
		wantPurpose5Enabled         string
		wantPurpose6Enabled         string
		wantPurpose7Enabled         string
		wantPurpose8Enabled         string
		wantPurpose9Enabled         string
		wantPurpose10Enabled        string
	}{
		{
			description: "New config and old config flags not set",
			config:      []byte{},
		},
		{
			description: "New config not set, old config set - use old flags",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enabled: false
                  purpose2:
                    enabled: true
                  purpose3:
                    enabled: false
                  purpose4:
                    enabled: true
                  purpose5:
                    enabled: false
                  purpose6:
                    enabled: true
                  purpose7:
                    enabled: false
                  purpose8:
                    enabled: true
                  purpose9:
                    enabled: false
                  purpose10:
                    enabled: true
            `),
			wantPurpose1EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose2EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose3EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose4EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose5EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose6EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose7EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose8EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose9EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose10EnforcePurpose: TCF2FullEnforcement,
			wantPurpose1Enabled:         falseStr,
			wantPurpose2Enabled:         trueStr,
			wantPurpose3Enabled:         falseStr,
			wantPurpose4Enabled:         trueStr,
			wantPurpose5Enabled:         falseStr,
			wantPurpose6Enabled:         trueStr,
			wantPurpose7Enabled:         falseStr,
			wantPurpose8Enabled:         trueStr,
			wantPurpose9Enabled:         falseStr,
			wantPurpose10Enabled:        trueStr,
		},
		{
			description: "New config flags set, old config flags not set - use new flags",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enforce_purpose: "full"
                  purpose2:
                    enforce_purpose: "no"
                  purpose3:
                    enforce_purpose: "full"
                  purpose4:
                    enforce_purpose: "no"
                  purpose5:
                    enforce_purpose: "full"
                  purpose6:
                    enforce_purpose: "no"
                  purpose7:
                    enforce_purpose: "full"
                  purpose8:
                    enforce_purpose: "no"
                  purpose9:
                    enforce_purpose: "full"
                  purpose10:
                    enforce_purpose: "no"
            `),
			wantPurpose1EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose2EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose3EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose4EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose5EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose6EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose7EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose8EnforcePurpose:  TCF2NoEnforcement,
			wantPurpose9EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose10EnforcePurpose: TCF2NoEnforcement,
			wantPurpose1Enabled:         trueStr,
			wantPurpose2Enabled:         falseStr,
			wantPurpose3Enabled:         trueStr,
			wantPurpose4Enabled:         falseStr,
			wantPurpose5Enabled:         trueStr,
			wantPurpose6Enabled:         falseStr,
			wantPurpose7Enabled:         trueStr,
			wantPurpose8Enabled:         falseStr,
			wantPurpose9Enabled:         trueStr,
			wantPurpose10Enabled:        falseStr,
		},
		{
			description: "New config flags and old config flags set - use new flags",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enabled: false
                    enforce_purpose: "full"
                  purpose2:
                    enabled: false
                    enforce_purpose: "full"
                  purpose3:
                    enabled: false
                    enforce_purpose: "full"
                  purpose4:
                    enabled: false
                    enforce_purpose: "full"
                  purpose5:
                    enabled: false
                    enforce_purpose: "full"
                  purpose6:
                    enabled: false
                    enforce_purpose: "full"
                  purpose7:
                    enabled: false
                    enforce_purpose: "full"
                  purpose8:
                    enabled: false
                    enforce_purpose: "full"
                  purpose9:
                    enabled: false
                    enforce_purpose: "full"
                  purpose10:
                    enabled: false
                    enforce_purpose: "full"
              `),
			wantPurpose1EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose2EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose3EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose4EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose5EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose6EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose7EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose8EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose9EnforcePurpose:  TCF2FullEnforcement,
			wantPurpose10EnforcePurpose: TCF2FullEnforcement,
			wantPurpose1Enabled:         trueStr,
			wantPurpose2Enabled:         trueStr,
			wantPurpose3Enabled:         trueStr,
			wantPurpose4Enabled:         trueStr,
			wantPurpose5Enabled:         trueStr,
			wantPurpose6Enabled:         trueStr,
			wantPurpose7Enabled:         trueStr,
			wantPurpose8Enabled:         trueStr,
			wantPurpose9Enabled:         trueStr,
			wantPurpose10Enabled:        trueStr,
		},
	}

	for _, tt := range tests {
		v := viper.New()
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer(tt.config))

		migrateConfigTCF2PurposeEnabledFlags(v)

		if len(tt.config) > 0 {
			assert.Equal(t, tt.wantPurpose1EnforcePurpose, v.GetString("gdpr.tcf2.purpose1.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose2EnforcePurpose, v.GetString("gdpr.tcf2.purpose2.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose3EnforcePurpose, v.GetString("gdpr.tcf2.purpose3.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose4EnforcePurpose, v.GetString("gdpr.tcf2.purpose4.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose5EnforcePurpose, v.GetString("gdpr.tcf2.purpose5.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose6EnforcePurpose, v.GetString("gdpr.tcf2.purpose6.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose7EnforcePurpose, v.GetString("gdpr.tcf2.purpose7.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose8EnforcePurpose, v.GetString("gdpr.tcf2.purpose8.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose9EnforcePurpose, v.GetString("gdpr.tcf2.purpose9.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose10EnforcePurpose, v.GetString("gdpr.tcf2.purpose10.enforce_purpose"), tt.description)
			assert.Equal(t, tt.wantPurpose1Enabled, v.GetString("gdpr.tcf2.purpose1.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose2Enabled, v.GetString("gdpr.tcf2.purpose2.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose3Enabled, v.GetString("gdpr.tcf2.purpose3.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose4Enabled, v.GetString("gdpr.tcf2.purpose4.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose5Enabled, v.GetString("gdpr.tcf2.purpose5.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose6Enabled, v.GetString("gdpr.tcf2.purpose6.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose7Enabled, v.GetString("gdpr.tcf2.purpose7.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose8Enabled, v.GetString("gdpr.tcf2.purpose8.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose9Enabled, v.GetString("gdpr.tcf2.purpose9.enabled"), tt.description)
			assert.Equal(t, tt.wantPurpose10Enabled, v.GetString("gdpr.tcf2.purpose10.enabled"), tt.description)
		} else {
			assert.Nil(t, v.Get("gdpr.tcf2.purpose1.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose2.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose3.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose4.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose5.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose6.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose7.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose8.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose9.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose10.enforce_purpose"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose1.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose2.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose3.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose4.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose5.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose6.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose7.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose8.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose9.enabled"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose10.enabled"), tt.description)
		}
	}
}

func TestInvalidAdapterEndpointConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(invalidAdapterEndpointConfig))
	_, err := New(v)

	if assert.IsType(t, errortypes.AggregateError{}, err) {
		aggErr := err.(errortypes.AggregateError)
		assert.ElementsMatch(t, []error{errors.New("The endpoint: ib.adnxs.com/some/endpoint for appnexus is not a valid URL")}, aggErr.Errors)
	}
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

func TestInvalidEnforcePurpose(t *testing.T) {
	cfg, v := newDefaultConfig(t)
	cfg.GDPR.TCF2.Purpose1.EnforcePurpose = ""
	cfg.GDPR.TCF2.Purpose2.EnforcePurpose = TCF2NoEnforcement
	cfg.GDPR.TCF2.Purpose3.EnforcePurpose = TCF2NoEnforcement
	cfg.GDPR.TCF2.Purpose4.EnforcePurpose = TCF2NoEnforcement
	cfg.GDPR.TCF2.Purpose5.EnforcePurpose = "invalid1"
	cfg.GDPR.TCF2.Purpose6.EnforcePurpose = "invalid2"
	cfg.GDPR.TCF2.Purpose7.EnforcePurpose = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose8.EnforcePurpose = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose9.EnforcePurpose = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose10.EnforcePurpose = "invalid3"

	errs := cfg.validate(v)

	expectedErrs := []error{
		errors.New("gdpr.tcf2.purpose1.enforce_purpose must be \"no\" or \"full\". Got "),
		errors.New("gdpr.tcf2.purpose5.enforce_purpose must be \"no\" or \"full\". Got invalid1"),
		errors.New("gdpr.tcf2.purpose6.enforce_purpose must be \"no\" or \"full\". Got invalid2"),
		errors.New("gdpr.tcf2.purpose10.enforce_purpose must be \"no\" or \"full\". Got invalid3"),
	}
	assert.ElementsMatch(t, errs, expectedErrs, "gdpr.tcf2.purposeX.enforce_purpose should prevent invalid values but it doesn't")
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
		SetupViper(v, "")
		v.Set("gdpr.default_value", "0")
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer([]byte(
			`request_validation:
    ipv4_private_networks: ` + test.privateIPNetworks)))

		result, resultErr := New(v)

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
	cfg.Accounts.Postgres.ConnectionInfo.Database = "accounts"

	errs := cfg.validate(v)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs, errors.New("accounts.postgres: retrieving accounts via postgres not available, use accounts.files"))
}

func TestUserSyncFromEnv(t *testing.T) {
	truePtr := true

	// setup env vars for testing
	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_URL"); ok {
		defer os.Setenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_URL", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_URL")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_USER_MACRO"); ok {
		defer os.Setenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_USER_MACRO", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_USER_MACRO")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_APPNEXUS_USERSYNC_SUPPORT_CORS"); ok {
		defer os.Setenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_SUPPORT_CORS", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_SUPPORT_CORS")
	}

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_RUBICON_USERSYNC_IFRAME_URL"); ok {
		defer os.Setenv("PBS_ADAPTERS_RUBICON_USERSYNC_IFRAME_URL", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_RUBICON_USERSYNC_IFRAME_URL")
	}

	// set new
	os.Setenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_URL", "http://some.url/sync?redirect={{.RedirectURL}}")
	os.Setenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_REDIRECT_USER_MACRO", "[UID]")
	os.Setenv("PBS_ADAPTERS_APPNEXUS_USERSYNC_SUPPORT_CORS", "true")
	os.Setenv("PBS_ADAPTERS_RUBICON_USERSYNC_IFRAME_URL", "http://somedifferent.url/sync?redirect={{.RedirectURL}}")

	cfg, _ := newDefaultConfig(t)
	assert.Equal(t, cfg.Adapters["appnexus"].Syncer.Redirect.URL, "http://some.url/sync?redirect={{.RedirectURL}}")
	assert.Equal(t, cfg.Adapters["appnexus"].Syncer.Redirect.UserMacro, "[UID]")
	assert.Nil(t, cfg.Adapters["appnexus"].Syncer.IFrame)
	assert.Equal(t, cfg.Adapters["appnexus"].Syncer.SupportCORS, &truePtr)

	assert.Equal(t, cfg.Adapters["rubicon"].Syncer.IFrame.URL, "http://somedifferent.url/sync?redirect={{.RedirectURL}}")
	assert.Nil(t, cfg.Adapters["rubicon"].Syncer.Redirect)
	assert.Nil(t, cfg.Adapters["rubicon"].Syncer.SupportCORS)

	assert.Nil(t, cfg.Adapters["brightroll"].Syncer)
}

func newDefaultConfig(t *testing.T) (*Configuration, *viper.Viper) {
	v := viper.New()
	SetupViper(v, "")
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	cfg, err := New(v)
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
