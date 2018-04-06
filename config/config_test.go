package config

import (
	"bytes"
	"testing"

	"github.com/spf13/viper"
)

func TestDefaults(t *testing.T) {

	cfg, err := New(newViperWithDefaults())
	if err != nil {
		t.Error(err.Error())
	}

	if cfg.Port != 8000 {
		t.Error("Expected Port 8000")
	}

	if cfg.AdminPort != 6060 {
		t.Error("Expected Admin Port 6060")
	}

	if cfg.DefaultTimeout != uint64(250) {
		t.Error("Expected DefaultTimeout of 250ms")
	}

	if cfg.DataCache.Type != "dummy" {
		t.Error("Expected DataCache Type of 'dummy'")
	}

	if cfg.Adapters["pubmatic"].Endpoint != "http://openbid-useast.pubmatic.com/translator?" {
		t.Errorf("Expected Pubmatic Endpoint of http://openbid-useast.pubmatic.com/translator?")
	}

}

var fullConfig = []byte(`
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
default_timeout_ms: 123
cache:
  scheme: http
  host: prebidcache.net
  query: uuid=%PBS_CACHE_UUID%
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
  indexExchange:
    endpoint: http://ixtest.com/api
  rubicon:
    endpoint: http://rubitest.com/api
    usersync_url: http://pixel.rubiconproject.com/sync.php?p=prebid
    xapi:
      username: rubiuser
      password: rubipw23
  facebook:
    endpoint: http://facebook.com/pbs
    usersync_url: http://facebook.com/ortb/prebid-s2s
    platform_id: abcdefgh1234
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

func TestFullConfig(t *testing.T) {
	v := newViperWithDefaults()
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
	if cfg.DefaultTimeout != 123 {
		t.Errorf("DefaultTimeout was %d not 123", cfg.DefaultTimeout)
	}
	cmpStrings(t, "cache.scheme", cfg.CacheURL.Scheme, "http")
	cmpStrings(t, "cache.host", cfg.CacheURL.Host, "prebidcache.net")
	cmpStrings(t, "cache.query", cfg.CacheURL.Query, "uuid=%PBS_CACHE_UUID%")
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
	cmpStrings(t, "adapters.appnexus.endpoint", cfg.Adapters["appnexus"].Endpoint, "http://ib.adnxs.com/some/endpoint")
	cmpStrings(t, "adapters.indexExchange.endpoint", cfg.Adapters["indexexchange"].Endpoint, "http://ixtest.com/api")
	cmpStrings(t, "adapters.rubicon.endpoint", cfg.Adapters["rubicon"].Endpoint, "http://rubitest.com/api")
	cmpStrings(t, "adapters.rubicon.usersync_url", cfg.Adapters["rubicon"].UserSyncURL, "http://pixel.rubiconproject.com/sync.php?p=prebid")
	cmpStrings(t, "adapters.rubicon.xapi.username", cfg.Adapters["rubicon"].XAPI.Username, "rubiuser")
	cmpStrings(t, "adapters.rubicon.xapi.password", cfg.Adapters["rubicon"].XAPI.Password, "rubipw23")
	cmpStrings(t, "adapters.facebook.endpoint", cfg.Adapters["facebook"].Endpoint, "http://facebook.com/pbs")
	cmpStrings(t, "adapters.facebook.usersync_url", cfg.Adapters["facebook"].UserSyncURL, "http://facebook.com/ortb/prebid-s2s")
	cmpStrings(t, "adapters.facebook.platform_id", cfg.Adapters["facebook"].PlatformID, "abcdefgh1234")
}

func newViperWithDefaults() *viper.Viper {
	v := viper.New()
	v.SetConfigName("pbs")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/config")

	v.SetDefault("external_url", "http://localhost:8000")
	v.SetDefault("port", 8000)
	v.SetDefault("admin_port", 6060)
	v.SetDefault("default_timeout_ms", 250)
	v.SetDefault("datacache.type", "dummy")

	v.SetDefault("adapters.pubmatic.endpoint", "http://openbid-useast.pubmatic.com/translator?")
	v.SetDefault("adapters.rubicon.endpoint", "http://staged-by.rubiconproject.com/a/api/exchange.json")
	v.SetDefault("adapters.rubicon.usersync_url", "https://pixel.rubiconproject.com/exchange/sync.php?p=prebid")
	v.SetDefault("adapters.pulsepoint.endpoint", "http://bid.contextweb.com/header/s/ortb/prebid-s2s")
	return v
}

func TestValidConfig(t *testing.T) {
	cfg := Configuration{
		StoredRequests: StoredRequests{
			Files: true,
		},
	}

	if err := cfg.validate(); err != nil {
		t.Errorf("OpenRTB filesystem config should work. %v", err)
	}
}

func TestInvalidStoredRequestsConfig(t *testing.T) {
	cfg := Configuration{
		StoredRequests: StoredRequests{
			Files:    true,
			Postgres: &PostgresConfig{},
		},
	}

	if err := cfg.validate(); err == nil {
		t.Error("OpenRTB Configs should not be allowed from both files and postgres.")
	}
}
