package config

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestDefaults(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	cfg, err := New(v)
	assert.NoError(t, err, "Setting up config should work but it doesn't")

	cmpInts(t, "port", cfg.Port, 8000)
	cmpInts(t, "admin_port", cfg.AdminPort, 6060)
	cmpInts(t, "auction_timeouts_ms.max", int(cfg.AuctionTimeouts.Max), 0)
	cmpInts(t, "max_request_size", int(cfg.MaxRequestSize), 1024*256)
	cmpInts(t, "host_cookie.ttl_days", int(cfg.HostCookie.TTL), 90)
	cmpInts(t, "host_cookie.max_cookie_size_bytes", cfg.HostCookie.MaxCookieSizeBytes, 0)
	cmpStrings(t, "datacache.type", cfg.DataCache.Type, "dummy")
	cmpStrings(t, "adapters.pubmatic.endpoint", cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint, "http://hbopenbid.pubmatic.com/translator?source=prebid-server")
	cmpInts(t, "currency_converter.fetch_interval_seconds", cfg.CurrencyConverter.FetchIntervalSeconds, 1800)
	cmpStrings(t, "currency_converter.fetch_url", cfg.CurrencyConverter.FetchURL, "https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json")
}

var fullConfig = []byte(`
gdpr:
  host_vendor_id: 15
  usersync_if_ambiguous: true
  non_standard_publishers: ["siteID","fake-site-id","appID","agltb3B1Yi1pbmNyDAsSA0FwcBiJkfIUDA"]
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
http_client:
  max_idle_connections: 500
  max_idle_connections_per_host: 20
  idle_connection_timeout_seconds: 30
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
datacache:
  type: postgres
  filename: /usr/db/db.db
  cache_size: 10000000
  ttl_seconds: 3600
adapters:
  appnexus:
    endpoint: http://ib.adnxs.com/some/endpoint
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
	cmpInts(t, "http_client.max_idle_connections", cfg.Client.MaxIdleConns, 500)
	cmpInts(t, "http_client.max_idle_connections_per_host", cfg.Client.MaxIdleConnsPerHost, 20)
	cmpInts(t, "http_client.idle_connection_timeout_seconds", cfg.Client.IdleConnTimeout, 30)
	cmpInts(t, "gdpr.host_vendor_id", cfg.GDPR.HostVendorID, 15)
	cmpBools(t, "gdpr.usersync_if_ambiguous", cfg.GDPR.UsersyncIfAmbiguous, true)

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

	//Assert the NonStandardPublishers was correctly unmarshalled
	cmpStrings(t, "blacklisted_apps", cfg.BlacklistedApps[0], "spamAppID")
	cmpStrings(t, "blacklisted_apps", cfg.BlacklistedApps[1], "sketchy-app-id")

	//Assert the BlacklistedAppMap hash table was built correctly
	for i := 0; i < len(cfg.BlacklistedApps); i++ {
		cmpBools(t, "cfg.BlacklistedAppMap", cfg.BlacklistedAppMap[cfg.BlacklistedApps[i]], true)
	}

	cmpStrings(t, "currency_converter.fetch_url", cfg.CurrencyConverter.FetchURL, "https://currency.prebid.org")
	cmpInts(t, "currency_converter.fetch_interval_seconds", cfg.CurrencyConverter.FetchIntervalSeconds, 1800)
	cmpStrings(t, "recaptcha_secret", cfg.RecaptchaSecret, "asdfasdfasdfasdf")
	cmpStrings(t, "metrics.influxdb.host", cfg.Metrics.Influxdb.Host, "upstream:8232")
	cmpStrings(t, "metrics.influxdb.database", cfg.Metrics.Influxdb.Database, "metricsdb")
	cmpStrings(t, "metrics.influxdb.username", cfg.Metrics.Influxdb.Username, "admin")
	cmpStrings(t, "metrics.influxdb.password", cfg.Metrics.Influxdb.Password, "admin1324")
	cmpStrings(t, "datacache.type", cfg.DataCache.Type, "postgres")
	cmpStrings(t, "datacache.filename", cfg.DataCache.Filename, "/usr/db/db.db")
	cmpInts(t, "datacache.cache_size", cfg.DataCache.CacheSize, 10000000)
	cmpInts(t, "datacache.ttl_seconds", cfg.DataCache.TTLSeconds, 3600)
	cmpStrings(t, "", cfg.CacheURL.GetBaseURL(), "http://prebidcache.net")
	cmpStrings(t, "", cfg.GetCachedAssetURL("a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11"), "http://prebidcache.net/cache?uuid=a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11")
	cmpStrings(t, "adapters.appnexus.endpoint", cfg.Adapters[string(openrtb_ext.BidderAppnexus)].Endpoint, "http://ib.adnxs.com/some/endpoint")
	cmpStrings(t, "adapters.audiencenetwork.endpoint", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].Endpoint, "http://facebook.com/pbs")
	cmpStrings(t, "adapters.audiencenetwork.usersync_url", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].UserSyncURL, "http://facebook.com/ortb/prebid-s2s")
	cmpStrings(t, "adapters.audiencenetwork.platform_id", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderFacebook))].PlatformID, "abcdefgh1234")
	cmpStrings(t, "adapters.ix.endpoint", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderIx))].Endpoint, "http://ixtest.com/api")
	cmpStrings(t, "adapters.rubicon.endpoint", cfg.Adapters[string(openrtb_ext.BidderRubicon)].Endpoint, "http://rubitest.com/api")
	cmpStrings(t, "adapters.rubicon.usersync_url", cfg.Adapters[string(openrtb_ext.BidderRubicon)].UserSyncURL, "http://pixel.rubiconproject.com/sync.php?p=prebid")
	cmpStrings(t, "adapters.rubicon.xapi.username", cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Username, "rubiuser")
	cmpStrings(t, "adapters.rubicon.xapi.password", cfg.Adapters[string(openrtb_ext.BidderRubicon)].XAPI.Password, "rubipw23")
	cmpStrings(t, "adapters.brightroll.endpoint", cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint, "http://test-bid.ybp.yahoo.com/bid/appnexuspbs")
	cmpStrings(t, "adapters.brightroll.usersync_url", cfg.Adapters[string(openrtb_ext.BidderBrightroll)].UserSyncURL, "http://test-bh.ybp.yahoo.com/sync/appnexuspbs?gdpr={{.GDPR}}&euconsent={{.GDPRConsent}}&url=%s")
	cmpStrings(t, "adapters.adkerneladn.usersync_url", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].UserSyncURL, "https://tag.adkernel.com/syncr?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&r=")
	cmpStrings(t, "adapters.rhythmone.endpoint", cfg.Adapters[string(openrtb_ext.BidderRhythmone)].Endpoint, "http://tag.1rx.io/rmp")
	cmpStrings(t, "adapters.rhythmone.usersync_url", cfg.Adapters[string(openrtb_ext.BidderRhythmone)].UserSyncURL, "https://sync.1rx.io/usersync2/rmphb?gdpr={{.GDPR}}&gdpr_consent={{.GDPRConsent}}&redir=http%3A%2F%2Fprebid-server.prebid.org%2F%2Fsetuid%3Fbidder%3Drhythmone%26gdpr%3D{{.GDPR}}%26gdpr_consent%3D{{.GDPRConsent}}%26uid%3D%5BRX_UUID%5D")
}

func TestValidConfig(t *testing.T) {
	cfg := Configuration{
		StoredRequests: StoredRequests{
			Files: true,
			InMemoryCache: InMemoryCache{
				Type: "none",
			},
		},
	}

	err := cfg.validate()
	assert.Nil(t, err, "OpenRTB filesystem config should work. %v", err)
}

func TestInvalidAdapterEndpointConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(invalidAdapterEndpointConfig))
	_, err := New(v)
	assert.Error(t, err, "invalid endpoint in config should return an error")
}

func TestInvalidAdapterUserSyncURLConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(invalidUserSyncURLConfig))
	_, err := New(v)
	assert.Error(t, err, "invalid user_sync URL in config should return an error")
}

func TestNegativeRequestSize(t *testing.T) {
	cfg := newDefaultConfig(t)
	cfg.MaxRequestSize = -1
	assertOneError(t, cfg.validate(), "cfg.max_request_size must be >= 0. Got -1")
}

func TestNegativeVendorID(t *testing.T) {
	cfg := newDefaultConfig(t)
	cfg.GDPR.HostVendorID = -1
	assertOneError(t, cfg.validate(), "gdpr.host_vendor_id must be in the range [0, 65535]. Got -1")
}

func TestNegativePrometheusTimeout(t *testing.T) {
	cfg := newDefaultConfig(t)
	cfg.Metrics.Prometheus.Port = 8001
	cfg.Metrics.Prometheus.TimeoutMillisRaw = 0
	assertOneError(t, cfg.validate(), "metrics.prometheus.timeout_ms must be positive if metrics.prometheus.port is defined. Got timeout=0 and port=8001")
}

func TestOverflowedVendorID(t *testing.T) {
	cfg := newDefaultConfig(t)
	cfg.GDPR.HostVendorID = (0xffff) + 1
	assertOneError(t, cfg.validate(), "gdpr.host_vendor_id must be in the range [0, 65535]. Got 65536")
}

func TestNegativeCurrencyConverterFetchInterval(t *testing.T) {
	cfg := Configuration{
		CurrencyConverter: CurrencyConverter{
			FetchIntervalSeconds: -1,
		},
	}
	err := cfg.validate()
	assert.NotNil(t, err, "cfg.currency_converter.fetch_interval_seconds should prevent negative values, but it doesn't")
}

func TestOverflowedCurrencyConverterFetchInterval(t *testing.T) {
	cfg := Configuration{
		CurrencyConverter: CurrencyConverter{
			FetchIntervalSeconds: (0xffff) + 1,
		},
	}
	err := cfg.validate()
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
	type aTest struct {
		cookieHost  *HostCookie
		expectError bool
	}
	testCases := []aTest{
		{cookieHost: &HostCookie{MaxCookieSizeBytes: 1 << 15}, expectError: false}, //32 KB, no error
		{cookieHost: &HostCookie{MaxCookieSizeBytes: 800}, expectError: false},
		{cookieHost: &HostCookie{MaxCookieSizeBytes: 500}, expectError: false},
		{cookieHost: &HostCookie{MaxCookieSizeBytes: 0}, expectError: false},
		{cookieHost: &HostCookie{MaxCookieSizeBytes: 200}, expectError: true},
		{cookieHost: &HostCookie{MaxCookieSizeBytes: -100}, expectError: true},
	}
	for i := range testCases {
		if testCases[i].expectError {
			assert.Error(t, isValidCookieSize(testCases[i].cookieHost.MaxCookieSizeBytes), fmt.Sprintf("Configuration.HostCooki.MaxCookieSizeBytes less than MIN_COOKIE_SIZE_BYTES = %d and not equal to zero should return an error", MIN_COOKIE_SIZE_BYTES))
		} else {
			assert.NoError(t, isValidCookieSize(testCases[i].cookieHost.MaxCookieSizeBytes), fmt.Sprintf("Configuration.HostCooki.MaxCookieSizeBytes greater than MIN_COOKIE_SIZE_BYTES = %d or equal to zero should not return an error", MIN_COOKIE_SIZE_BYTES))
		}
	}
}

func newDefaultConfig(t *testing.T) *Configuration {
	v := viper.New()
	SetupViper(v, "")
	v.SetConfigType("yaml")
	cfg, err := New(v)
	assert.NoError(t, err)
	return cfg
}

func assertOneError(t *testing.T, errs configErrors, message string) {
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
