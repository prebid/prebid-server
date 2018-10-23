package config

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/prebid/prebid-server/openrtb_ext"

	"github.com/spf13/viper"
)

func TestDefaults(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	cfg, err := New(v)
	if err != nil {
		t.Error(err.Error())
	}

	cmpInts(t, "port", cfg.Port, 8000)
	cmpInts(t, "admin_port", cfg.AdminPort, 6060)
	cmpInts(t, "auction_timeouts_ms.max", int(cfg.AuctionTimeouts.Max), 0)
	cmpInts(t, "max_request_size", int(cfg.MaxRequestSize), 1024*256)
	cmpInts(t, "host_cookie.ttl_days", int(cfg.HostCookie.TTL), 90)
	cmpStrings(t, "datacache.type", cfg.DataCache.Type, "dummy")
	cmpStrings(t, "adapters.pubmatic.endpoint", cfg.Adapters[string(openrtb_ext.BidderPubmatic)].Endpoint, "http://hbopenbid.pubmatic.com/translator?source=prebid-server")
}

var fullConfig = []byte(`
gdpr:
  host_vendor_id: 15
  usersync_if_ambiguous: true
host_cookie:
  cookie_name: userid
  family: prebid
  domain: cookies.prebid.org
  opt_out_url: http://prebid.org/optout
  opt_in_url: http://prebid.org/optin
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
    usersync_url: http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr={{gdpr}}&euconsent={{gdpr_consent}}&url=%s
    endpoint: http://east-bid.ybp.yahoo.com/bid/appnexuspbs
  adkerneladn:
     usersync_url: https://tag.adkernel.com/syncr?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&r=
`)

func cmpStrings(t *testing.T, key string, a string, b string) {
	t.Helper()
	if a != b {
		t.Errorf("%s: %s != %s", key, a, b)
	}
}

func cmpInts(t *testing.T, key string, a int, b int) {
	t.Helper()
	if a != b {
		t.Errorf("%s: %d != %d", key, a, b)
	}
}

func cmpBools(t *testing.T, key string, a bool, b bool) {
	t.Helper()
	if a != b {
		t.Errorf("%s: %t != %t", key, a, b)
	}
}

func TestFullConfig(t *testing.T) {
	v := viper.New()
	SetupViper(v, "")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(fullConfig))
	cfg, err := New(v)
	if err != nil {
		t.Fatal(err.Error())
	}
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
	cmpStrings(t, "adapters.brightroll.endpoint", cfg.Adapters[string(openrtb_ext.BidderBrightroll)].Endpoint, "http://east-bid.ybp.yahoo.com/bid/appnexuspbs")
	cmpStrings(t, "adapters.brightroll.usersync_url", cfg.Adapters[string(openrtb_ext.BidderBrightroll)].UserSyncURL, "http://east-bid.ybp.yahoo.com/sync/appnexuspbs?gdpr={{gdpr}}&euconsent={{gdpr_consent}}&url=%s")
	cmpStrings(t, "adapters.adkerneladn.usersync_url", cfg.Adapters[strings.ToLower(string(openrtb_ext.BidderAdkernelAdn))].UserSyncURL, "https://tag.adkernel.com/syncr?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&r=")
	cmpStrings(t, "adapters.rhythmone.endpoint", cfg.Adapters[string(openrtb_ext.BidderRhythmone)].Endpoint, "http://tag.1rx.io/rmp")
	cmpStrings(t, "adapters.rhythmone.usersync_url", cfg.Adapters[string(openrtb_ext.BidderRhythmone)].UserSyncURL, "//sync.1rx.io/usersync2/rmphb?gdpr={{gdpr}}&gdpr_consent={{gdpr_consent}}&redir=")
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

	if err := cfg.validate(); err != nil {
		t.Errorf("OpenRTB filesystem config should work. %v", err)
	}
}

func TestNegativeRequestSize(t *testing.T) {
	cfg := Configuration{
		MaxRequestSize: -1,
	}

	if err := cfg.validate(); err == nil {
		t.Error("cfg.max_request_size should prevent negative values, but it doesn't")
	}
}

func TestNegativeVendorID(t *testing.T) {
	cfg := Configuration{
		GDPR: GDPR{
			HostVendorID: -1,
		},
	}

	if err := cfg.validate(); err == nil {
		t.Error("cfg.gdpr.host_vendor_id should prevent negative values, but it doesn't")
	}
}

func TestOverflowedVendorID(t *testing.T) {
	cfg := Configuration{
		GDPR: GDPR{
			HostVendorID: (0xffff) + 1,
		},
	}

	if err := cfg.validate(); err == nil {
		t.Errorf("cfg.gdpr.host_vendor_id should prevent values over %d, but it doesn't", 0xffff)
	}
}

func TestLimitTimeout(t *testing.T) {
	doTimeoutTest(t, 10, 15, 10, 0)
	doTimeoutTest(t, 10, 0, 10, 0)
	doTimeoutTest(t, 5, 5, 10, 0)
	doTimeoutTest(t, 15, 15, 0, 0)
	doTimeoutTest(t, 15, 0, 20, 15)

}

func doTimeoutTest(t *testing.T, expected int, requested int, max uint64, def uint64) {
	t.Helper()
	cfg := AuctionTimeouts{
		Default: def,
		Max:     max,
	}
	expectedDuration := time.Duration(expected) * time.Millisecond
	limited := cfg.LimitAuctionTimeout(time.Duration(requested) * time.Millisecond)
	if limited != expectedDuration {
		t.Errorf("Expected %dms timeout, got %dms", expectedDuration, limited/time.Millisecond)
	}
}
