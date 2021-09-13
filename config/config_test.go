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
	cmpStrings(t, "datacache.type", cfg.DataCache.Type, "dummy")
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
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose2: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose3: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose4: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose5: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose6: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose7: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose8: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose9: TCF2Purpose{
			Enabled:            true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose10: TCF2Purpose{
			Enabled:            true,
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
}

var fullConfig = []byte(`
gdpr:
  host_vendor_id: 15
  default_value: "1"
  non_standard_publishers: ["siteID","fake-site-id","appID","agltb3B1Yi1pbmNyDAsSA0FwcBiJkfIUDA"]
  tcf2:
    purpose1:
      enforce_vendors: false
      vendor_exceptions: ["foo1a", "foo1b"]
    purpose2:
      enabled: false
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
datacache:
  type: postgres
  filename: /usr/db/db.db
  cache_size: 10000000
  ttl_seconds: 3600
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
    usersync_url: http://pixel.rubiconproject.com/sync.php?p=prebid
    xapi:
      username: rubiuser
      password: rubipw23
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
  audienceNetwork:
    endpoint: http://facebook.com/pbs
    usersync_url: http://facebook.com/ortb/prebid-s2s
    platform_id: abcdefgh1234
  ix:
    endpoint: http://ixtest.com/api
  rubicon:
    endpoint: http://rubitest.com/api
    usersync_url: http://pixel.rubiconproject.com/sync.php?p=prebid
    xapi:
      username: rubiuser
      password: rubipw23
  brightroll:
    usersync_url: http://test-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&url=%s
    endpoint: http://test-bid.ybp.yahoo.com/bid/appnexuspbs
  adkerneladn:
     usersync_url: https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=
blacklisted_apps: ["spamAppID","sketchy-app-id"]
`)

var invalidAdapterEndpointConfig = []byte(`
adapters:
  appnexus:
    endpoint: ib.adnxs.com/some/endpoint
  audienceNetwork:
    endpoint: http://facebook.com/pbs
    usersync_url: http://facebook.com/ortb/prebid-s2s
    platform_id: abcdefgh1234
  brightroll:
    usersync_url: http://http://test-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&url=%s
  adkerneladn:
     usersync_url: https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=
`)

var invalidUserSyncURLConfig = []byte(`
adapters:
  appnexus:
    endpoint: http://ib.adnxs.com/some/endpoint
  audienceNetwork:
    endpoint: http://facebook.com/pbs
    usersync_url: http://facebook.com/ortb/prebid-s2s
    platform_id: abcdefgh1234
  brightroll:
    usersync_url: http//test-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&url=%s
  adkerneladn:
     usersync_url: http:\\tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=
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
	cmpStrings(t, "gdpr.non_standard_publishers", cfg.GDPR.NonStandardPublishers[0], "siteID")
	cmpStrings(t, "gdpr.non_standard_publishers", cfg.GDPR.NonStandardPublishers[1], "fake-site-id")
	cmpStrings(t, "gdpr.non_standard_publishers", cfg.GDPR.NonStandardPublishers[2], "appID")
	cmpStrings(t, "gdpr.non_standard_publishers", cfg.GDPR.NonStandardPublishers[3], "agltb3B1Yi1pbmNyDAsSA0FwcBiJkfIUDA")

	//Assert the NonStandardPublisherMap hash table was built correctly
	var found bool
	for i := 0; i < len(cfg.GDPR.NonStandardPublishers); i++ {
		_, found = cfg.GDPR.NonStandardPublisherMap[cfg.GDPR.NonStandardPublishers[i]]
		cmpBools(t, "cfg.GDPR.NonStandardPublisherMap", found, true)
	}
	_, found = cfg.GDPR.NonStandardPublisherMap["appnexus"]
	cmpBools(t, "cfg.GDPR.NonStandardPublisherMap", found, false)

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
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo1a"), openrtb_ext.BidderName("foo1b")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo1a"): {}, openrtb_ext.BidderName("foo1b"): {}},
		},
		Purpose2: TCF2Purpose{
			Enabled:            false,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo2")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo2"): {}},
		},
		Purpose3: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo3")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo3"): {}},
		},
		Purpose4: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo4")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo4"): {}},
		},
		Purpose5: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo5")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo5"): {}},
		},
		Purpose6: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo6")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo6"): {}},
		},
		Purpose7: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo7")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo7"): {}},
		},
		Purpose8: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo8")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo8"): {}},
		},
		Purpose9: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo9")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo9"): {}},
		},
		Purpose10: TCF2Purpose{
			Enabled:            true, // true by default
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
	cmpStrings(t, "datacache.type", cfg.DataCache.Type, "postgres")
	cmpStrings(t, "datacache.filename", cfg.DataCache.Filename, "/usr/db/db.db")
	cmpInts(t, "datacache.cache_size", cfg.DataCache.CacheSize, 10000000)
	cmpInts(t, "datacache.ttl_seconds", cfg.DataCache.TTLSeconds, 3600)
	cmpStrings(t, "", cfg.CacheURL.GetBaseURL(), "http://prebidcache.net")
	cmpStrings(t, "", cfg.GetCachedAssetURL("a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11"), "http://prebidcache.net/cache?uuid=a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11")
	cmpStrings(t, "adapters.appnexus.endpoint", cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, "http://ib.adnxs.com/some/endpoint")
	cmpStrings(t, "adapters.appnexus.extra_info", cfg.Adapters[string(openrtb_ext.BidderAppnexus)].ExtraAdapterInfo, "{\"native\":\"http://www.native.org/endpoint\",\"video\":\"http://www.video.org/endpoint\"}")
	cmpStrings(t, "adapters.audiencenetwork.endpoint", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].Endpoint, "http://facebook.com/pbs")
	cmpStrings(t, "adapters.audiencenetwork.usersync_url", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].UserSyncURL, "http://facebook.com/ortb/prebid-s2s")
	cmpStrings(t, "adapters.audiencenetwork.platform_id", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].PlatformID, "abcdefgh1234")
	cmpStrings(t, "adapters.audiencenetwork.app_secret", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAudienceNetwork))].AppSecret, "987abc")
	cmpStrings(t, "adapters.beachfront.endpoint", cfg.Adapters[string(openrtb_ext.BidderBeachfront)].Endpoint, "https://display.bfmio.com/prebid_display")
	cmpStrings(t, "adapters.beachfront.extra_info", cfg.Adapters[string(openrtb_ext.BidderBeachfront)].ExtraAdapterInfo, "{\"video_endpoint\":\"https://reachms.bfmio.com/bid.json?exchange_id\"}")
	cmpStrings(t, "adapters.ix.endpoint", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIx))].Endpoint, "http://ixtest.com/api")
	cmpStrings(t, "adapters.rubicon.endpoint", cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint, "http://rubitest.com/api")
	cmpStrings(t, "adapters.rubicon.usersync_url", cfg.Adapters[string(openrtb_ext.BidderRubicon)].UserSyncURL, "http://pixel.rubiconproject.com/sync.php?p=prebid")
	cmpStrings(t, "adapters.rubicon.xapi.username", cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username, "rubiuser")
	cmpStrings(t, "adapters.rubicon.xapi.password", cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password, "rubipw23")
	cmpStrings(t, "adapters.brightroll.endpoint", cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint, "http://test-bid.ybp.yahoo.com/bid/appnexuspbs")
	cmpStrings(t, "adapters.brightroll.usersync_url", cfg.Adapters[string(openrtb_ext.BidderBrightroll)].UserSyncURL, "http://test-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&url=%s")
	cmpStrings(t, "adapters.adkerneladn.usersync_url", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].UserSyncURL, "https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=")
	cmpStrings(t, "adapters.rhythmone.endpoint", cfg.Adapters[string(openrtb_ext.BidderRhythmone)].Endpoint, "http://tag.1rx.io/rmp")
	cmpStrings(t, "adapters.rhythmone.usersync_url", cfg.Adapters[string(openrtb_ext.BidderRhythmone)].UserSyncURL, "https://sync.1rx.io/usersync2/rmphb?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&us_privacy={{.USPrivacy}}&redir=http%3A%2F%2Fprebid-server.prebid.org%2F%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5BRX_UUID%5D")
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

	// Unescape quotes of JSON-formatted string
	strings.Replace(cfg.Adapters[string(openrtb_ext.BidderAppnexus)].ExtraAdapterInfo, "\\\"", "\"", -1)

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

func TestValidConfig(t *testing.T) {
	cfg := Configuration{
		GDPR: GDPR{
			DefaultValue: "1",
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

func TestInvalidAdapterEndpointConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(invalidAdapterEndpointConfig))
	_, err := New(v)
	assert.Error(t, err, "invalid endpoint in config should return an error")
}

func TestInvalidAdapterUserSyncURLConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(invalidUserSyncURLConfig))
	_, err := New(v)
	assert.Error(t, err, "invalid user_sync URL in config should return an error")
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
