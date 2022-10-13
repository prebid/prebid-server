package config

import (
	"bytes"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/prebid/go-gdpr/consentconstants"
	"github.com/prebid/prebid-server/openrtb_ext"
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
		UserSyncURL: "http://bidder2.com/usersync",
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

	cmpInts(t, "port", cfg.Port, 8000)
	cmpInts(t, "admin_port", cfg.AdminPort, 6060)
	cmpInts(t, "auction_timeouts_ms.max", int(cfg.AuctionTimeouts.Max), 0)
	cmpInts(t, "max_request_size", int(cfg.MaxRequestSize), 1024*256)
	cmpInts(t, "host_cookie.ttl_days", int(cfg.HostCookie.TTL), 90)
	cmpInts(t, "host_cookie.max_cookie_size_bytes", cfg.HostCookie.MaxCookieSizeBytes, 0)
	cmpInts(t, "currency_converter.fetch_interval_seconds", cfg.CurrencyConverter.FetchIntervalSeconds, 1800)
	cmpStrings(t, "currency_converter.fetch_url", cfg.CurrencyConverter.FetchURL, "https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json")
	cmpBools(t, "account_required", cfg.AccountRequired, false)
	cmpInts(t, "metrics.influxdb.collection_rate_seconds", cfg.Metrics.Influxdb.MetricSendInterval, 20)
	cmpBools(t, "account_adapter_details", cfg.Metrics.Disabled.AccountAdapterDetails, false)
	cmpBools(t, "account_debug", cfg.Metrics.Disabled.AccountDebug, true)
	cmpBools(t, "account_stored_responses", cfg.Metrics.Disabled.AccountStoredResponses, true)
	cmpBools(t, "adapter_connections_metrics", cfg.Metrics.Disabled.AdapterConnectionMetrics, true)
	cmpBools(t, "adapter_gdpr_request_blocked", cfg.Metrics.Disabled.AdapterGDPRRequestBlocked, false)
	cmpStrings(t, "certificates_file", cfg.PemCertsFile, "")
	cmpBools(t, "stored_requests.filesystem.enabled", false, cfg.StoredRequests.Files.Enabled)
	cmpStrings(t, "stored_requests.filesystem.directorypath", "./stored_requests/data/by_id", cfg.StoredRequests.Files.Path)
	cmpBools(t, "auto_gen_source_tid", cfg.AutoGenSourceTID, true)
	cmpBools(t, "generate_bid_id", cfg.GenerateBidID, false)
	cmpStrings(t, "experiment.adscert.mode", cfg.Experiment.AdCerts.Mode, "off")
	cmpStrings(t, "experiment.adscert.inprocess.origin", cfg.Experiment.AdCerts.InProcess.Origin, "")
	cmpStrings(t, "experiment.adscert.inprocess.key", cfg.Experiment.AdCerts.InProcess.PrivateKey, "")
	cmpInts(t, "experiment.adscert.inprocess.domain_check_interval_seconds", cfg.Experiment.AdCerts.InProcess.DNSCheckIntervalInSeconds, 30)
	cmpInts(t, "experiment.adscert.inprocess.domain_renewal_interval_seconds", cfg.Experiment.AdCerts.InProcess.DNSRenewalIntervalInSeconds, 30)
	cmpStrings(t, "experiment.adscert.remote.url", cfg.Experiment.AdCerts.Remote.Url, "")
	cmpInts(t, "experiment.adscert.remote.signing_timeout_ms", cfg.Experiment.AdCerts.Remote.SigningTimeoutMs, 5)
	cmpNils(t, "host_schain_node", cfg.HostSChainNode)
	cmpStrings(t, "datacenter", cfg.DataCenter, "")

	//Assert purpose VendorExceptionMap hash tables were built correctly
	expectedTCF2 := TCF2{
		Enabled: true,
		Purpose1: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose2: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose3: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose4: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose5: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose6: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose7: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose8: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose9: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
		},
		Purpose10: TCF2Purpose{
			Enabled:            true,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     true,
			VendorExceptions:   []openrtb_ext.BidderName{},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{},
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
      enforce_algo: "full"
      enforce_purpose: false
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
    adapter_gdpr_request_blocked: true
blacklisted_apps: ["spamAppID","sketchy-app-id"]
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

func cmpInt8s(t *testing.T, key string, a *int8, b *int8) {
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
	int8One := int8(1)

	v := viper.New()
	SetupViper(v, "", bidderInfos)
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(fullConfig))
	cfg, err := New(v, bidderInfos, mockNormalizeBidderName)
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
	cmpStrings(t, "host_schain_node.asi", cfg.HostSChainNode.ASI, "pbshostcompany.com")
	cmpStrings(t, "host_schain_node.sid", cfg.HostSChainNode.SID, "00001")
	cmpStrings(t, "host_schain_node.rid", cfg.HostSChainNode.RID, "BidRequest")
	cmpInt8s(t, "host_schain_node.hp", cfg.HostSChainNode.HP, &int8One)
	cmpStrings(t, "datacenter", cfg.DataCenter, "1")

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
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo1a"), openrtb_ext.BidderName("foo1b")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo1a"): {}, openrtb_ext.BidderName("foo1b"): {}},
		},
		Purpose2: TCF2Purpose{
			Enabled:            false,
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     false,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo2")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo2"): {}},
		},
		Purpose3: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo3")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo3"): {}},
		},
		Purpose4: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo4")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo4"): {}},
		},
		Purpose5: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo5")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo5"): {}},
		},
		Purpose6: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo6")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo6"): {}},
		},
		Purpose7: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo7")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo7"): {}},
		},
		Purpose8: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo8")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo8"): {}},
		},
		Purpose9: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo9")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo9"): {}},
		},
		Purpose10: TCF2Purpose{
			Enabled:            true, // true by default
			EnforceAlgo:        TCF2FullEnforcement,
			EnforcePurpose:     true,
			EnforceVendors:     false,
			VendorExceptions:   []openrtb_ext.BidderName{openrtb_ext.BidderName("foo10")},
			VendorExceptionMap: map[openrtb_ext.BidderName]struct{}{openrtb_ext.BidderName("foo10"): {}},
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
	assert.Equal(t, expectedTCF2, cfg.GDPR.TCF2, "gdpr.tcf2")

	cmpStrings(t, "currency_converter.fetch_url", cfg.CurrencyConverter.FetchURL, "https://currency.prebid.org")
	cmpInts(t, "currency_converter.fetch_interval_seconds", cfg.CurrencyConverter.FetchIntervalSeconds, 1800)
	cmpStrings(t, "recaptcha_secret", cfg.RecaptchaSecret, "asdfasdfasdfasdf")
	cmpStrings(t, "metrics.influxdb.host", cfg.Metrics.Influxdb.Host, "upstream:8232")
	cmpStrings(t, "metrics.influxdb.database", cfg.Metrics.Influxdb.Database, "metricsdb")
	cmpStrings(t, "metrics.influxdb.measurement", cfg.Metrics.Influxdb.Measurement, "anyMeasurement")
	cmpStrings(t, "metrics.influxdb.username", cfg.Metrics.Influxdb.Username, "admin")
	cmpStrings(t, "metrics.influxdb.password", cfg.Metrics.Influxdb.Password, "admin1324")
	cmpBools(t, "metrics.influxdb.align_timestamps", cfg.Metrics.Influxdb.AlignTimestamps, true)
	cmpInts(t, "metrics.influxdb.metric_send_interval", cfg.Metrics.Influxdb.MetricSendInterval, 30)
	cmpStrings(t, "", cfg.CacheURL.GetBaseURL(), "http://prebidcache.net")
	cmpStrings(t, "", cfg.GetCachedAssetURL("a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11"), "http://prebidcache.net/cache?uuid=a0eebc99-9c0b-4ef8-bb00-6bb9bd380a11")
	cmpBools(t, "account_required", cfg.AccountRequired, true)
	cmpBools(t, "auto_gen_source_tid", cfg.AutoGenSourceTID, false)
	cmpBools(t, "account_adapter_details", cfg.Metrics.Disabled.AccountAdapterDetails, true)
	cmpBools(t, "account_debug", cfg.Metrics.Disabled.AccountDebug, false)
	cmpBools(t, "account_stored_responses", cfg.Metrics.Disabled.AccountStoredResponses, false)
	cmpBools(t, "adapter_connections_metrics", cfg.Metrics.Disabled.AdapterConnectionMetrics, true)
	cmpBools(t, "adapter_gdpr_request_blocked", cfg.Metrics.Disabled.AdapterGDPRRequestBlocked, true)
	cmpStrings(t, "certificates_file", cfg.PemCertsFile, "/etc/ssl/cert.pem")
	cmpStrings(t, "request_validation.ipv4_private_networks", cfg.RequestValidation.IPv4PrivateNetworks[0], "1.1.1.0/24")
	cmpStrings(t, "request_validation.ipv6_private_networks", cfg.RequestValidation.IPv6PrivateNetworks[0], "1111::/16")
	cmpStrings(t, "request_validation.ipv6_private_networks", cfg.RequestValidation.IPv6PrivateNetworks[1], "2222::/16")
	cmpBools(t, "generate_bid_id", cfg.GenerateBidID, true)
	cmpStrings(t, "debug.override_token", cfg.Debug.OverrideToken, "")
	cmpStrings(t, "experiment.adscert.mode", cfg.Experiment.AdCerts.Mode, "inprocess")
	cmpStrings(t, "experiment.adscert.inprocess.origin", cfg.Experiment.AdCerts.InProcess.Origin, "http://test.com")
	cmpStrings(t, "experiment.adscert.inprocess.key", cfg.Experiment.AdCerts.InProcess.PrivateKey, "ABC123")
	cmpInts(t, "experiment.adscert.inprocess.domain_check_interval_seconds", cfg.Experiment.AdCerts.InProcess.DNSCheckIntervalInSeconds, 40)
	cmpInts(t, "experiment.adscert.inprocess.domain_renewal_interval_seconds", cfg.Experiment.AdCerts.InProcess.DNSRenewalIntervalInSeconds, 60)
	cmpStrings(t, "experiment.adscert.remote.url", cfg.Experiment.AdCerts.Remote.Url, "")
	cmpInts(t, "experiment.adscert.remote.signing_timeout_ms", cfg.Experiment.AdCerts.Remote.SigningTimeoutMs, 10)
}

func TestValidateConfig(t *testing.T) {
	cfg := Configuration{
		GDPR: GDPR{
			DefaultValue: "1",
			TCF2: TCF2{
				Purpose1:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose2:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose3:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose4:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose5:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose6:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose7:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose8:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose9:  TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
				Purpose10: TCF2Purpose{EnforceAlgo: TCF2FullEnforcement},
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
	SetupViper(v, "", bidderInfos)
	v.Set("gdpr.default_value", "0")
	v.SetConfigType("yaml")
	v.ReadConfig(bytes.NewBuffer(oldStoredRequestsConfig))
	migrateConfig(v)
	cfg, err := New(v, bidderInfos, mockNormalizeBidderName)
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

	if oldval, ok := os.LookupEnv("PBS_ADAPTERS_BIDDER1_ENDPOINT"); ok {
		defer os.Setenv("PBS_ADAPTERS_BIDDER1_ENDPOINT", oldval)
	} else {
		defer os.Unsetenv("PBS_ADAPTERS_BIDDER1_ENDPOINT")
	}

	os.Setenv("PBS_STORED_REQUESTS_FILESYSTEM", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER1_ENDPOINT", "http://bidder1_override.com")
	cfg, _ := newDefaultConfig(t)
	cmpBools(t, "stored_requests.filesystem.enabled", true, cfg.StoredRequests.Files.Enabled)
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

	assert.Nil(t, cfg.BidderInfos["brightroll"].Syncer)
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

	// set new
	os.Setenv("PBS_ADAPTERS_BIDDER1_DISABLED", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER1_ENDPOINT", "http://some.url/override")
	os.Setenv("PBS_ADAPTERS_BIDDER1_EXTRA_INFO", `{"extrainfo": true}`)
	os.Setenv("PBS_ADAPTERS_BIDDER1_DEBUG_ALLOW", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER1_GVLVENDORID", "42")
	os.Setenv("PBS_ADAPTERS_BIDDER1_EXPERIMENT_ADSCERT_ENABLED", "true")
	os.Setenv("PBS_ADAPTERS_BIDDER1_XAPI_USERNAME", "username_override")
	os.Setenv("PBS_ADAPTERS_BIDDER1_USERSYNC_REDIRECT_URL", "http://some.url/sync?redirect={{.RedirectURL}}")

	cfg, _ := newDefaultConfig(t)

	assert.Equal(t, true, cfg.BidderInfos["bidder1"].Disabled)
	assert.Equal(t, "http://some.url/override", cfg.BidderInfos["bidder1"].Endpoint)
	assert.Equal(t, `{"extrainfo": true}`, cfg.BidderInfos["bidder1"].ExtraAdapterInfo)

	assert.Equal(t, true, cfg.BidderInfos["bidder1"].Debug.Allow)
	assert.Equal(t, uint16(42), cfg.BidderInfos["bidder1"].GVLVendorID)

	assert.Equal(t, true, cfg.BidderInfos["bidder1"].Experiment.AdsCert.Enabled)
	assert.Equal(t, "username_override", cfg.BidderInfos["bidder1"].XAPI.Username)
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

func TestMigrateConfigSpecialFeature1(t *testing.T) {
	oldSpecialFeature1Config := []byte(`
      gdpr:
        tcf2:
          special_purpose1:
            enabled: true
            vendor_exceptions: ["appnexus"]
    `)
	newSpecialFeature1Config := []byte(`
      gdpr:
        tcf2:
          special_feature1:
            enforce: true
            vendor_exceptions: ["appnexus"]
    `)
	oldAndNewSpecialFeature1Config := []byte(`
      gdpr:
        tcf2:
          special_purpose1:
            enabled: false
            vendor_exceptions: ["appnexus"]
          special_feature1:
            enforce: true
            vendor_exceptions: ["rubicon"]
    `)

	tests := []struct {
		description                         string
		config                              []byte
		wantSpecialFeature1Enforce          bool
		wantSpecialFeature1VendorExceptions []string
	}{
		{
			description: "New config and old config not set",
			config:      []byte{},
		},
		{
			description:                         "New config not set, old config set",
			config:                              oldSpecialFeature1Config,
			wantSpecialFeature1Enforce:          true,
			wantSpecialFeature1VendorExceptions: []string{"appnexus"},
		},
		{
			description:                         "New config set, old config not set",
			config:                              newSpecialFeature1Config,
			wantSpecialFeature1Enforce:          true,
			wantSpecialFeature1VendorExceptions: []string{"appnexus"},
		},
		{
			description:                         "New config and old config set",
			config:                              oldAndNewSpecialFeature1Config,
			wantSpecialFeature1Enforce:          true,
			wantSpecialFeature1VendorExceptions: []string{"rubicon"},
		},
	}

	for _, tt := range tests {
		v := viper.New()
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer(tt.config))

		migrateConfigSpecialFeature1(v)

		if len(tt.config) > 0 {
			assert.Equal(t, tt.wantSpecialFeature1Enforce, v.Get("gdpr.tcf2.special_feature1.enforce").(bool), tt.description)
			assert.Equal(t, tt.wantSpecialFeature1VendorExceptions, v.GetStringSlice("gdpr.tcf2.special_feature1.vendor_exceptions"), tt.description)
		} else {
			assert.Nil(t, v.Get("gdpr.tcf2.special_feature1.enforce"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.special_feature1.vendor_exceptions"), tt.description)
		}

		var c Configuration
		err := v.Unmarshal(&c)
		assert.NoError(t, err, tt.description)
		assert.Equal(t, tt.wantSpecialFeature1Enforce, c.GDPR.TCF2.SpecialFeature1.Enforce, tt.description)

		// convert expected vendor exceptions to type BidderName
		expectedVendorExceptions := make([]openrtb_ext.BidderName, 0, 0)
		for _, ve := range tt.wantSpecialFeature1VendorExceptions {
			expectedVendorExceptions = append(expectedVendorExceptions, openrtb_ext.BidderName(ve))
		}
		assert.ElementsMatch(t, expectedVendorExceptions, c.GDPR.TCF2.SpecialFeature1.VendorExceptions, tt.description)
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
			wantPurpose1EnforcePurpose:  falseStr,
			wantPurpose2EnforcePurpose:  trueStr,
			wantPurpose3EnforcePurpose:  falseStr,
			wantPurpose4EnforcePurpose:  trueStr,
			wantPurpose5EnforcePurpose:  falseStr,
			wantPurpose6EnforcePurpose:  trueStr,
			wantPurpose7EnforcePurpose:  falseStr,
			wantPurpose8EnforcePurpose:  trueStr,
			wantPurpose9EnforcePurpose:  falseStr,
			wantPurpose10EnforcePurpose: trueStr,
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
                    enforce_purpose: true
                  purpose2:
                    enforce_purpose: false
                  purpose3:
                    enforce_purpose: true
                  purpose4:
                    enforce_purpose: false
                  purpose5:
                    enforce_purpose: true
                  purpose6:
                    enforce_purpose: false
                  purpose7:
                    enforce_purpose: true
                  purpose8:
                    enforce_purpose: false
                  purpose9:
                    enforce_purpose: true
                  purpose10:
                    enforce_purpose: false
            `),
			wantPurpose1EnforcePurpose:  trueStr,
			wantPurpose2EnforcePurpose:  falseStr,
			wantPurpose3EnforcePurpose:  trueStr,
			wantPurpose4EnforcePurpose:  falseStr,
			wantPurpose5EnforcePurpose:  trueStr,
			wantPurpose6EnforcePurpose:  falseStr,
			wantPurpose7EnforcePurpose:  trueStr,
			wantPurpose8EnforcePurpose:  falseStr,
			wantPurpose9EnforcePurpose:  trueStr,
			wantPurpose10EnforcePurpose: falseStr,
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
                    enforce_purpose: true
                  purpose2:
                    enabled: false
                    enforce_purpose: true
                  purpose3:
                    enabled: false
                    enforce_purpose: true
                  purpose4:
                    enabled: false
                    enforce_purpose: true
                  purpose5:
                    enabled: false
                    enforce_purpose: true
                  purpose6:
                    enabled: false
                    enforce_purpose: true
                  purpose7:
                    enabled: false
                    enforce_purpose: true
                  purpose8:
                    enabled: false
                    enforce_purpose: true
                  purpose9:
                    enabled: false
                    enforce_purpose: true
                  purpose10:
                    enabled: false
                    enforce_purpose: true
              `),
			wantPurpose1EnforcePurpose:  trueStr,
			wantPurpose2EnforcePurpose:  trueStr,
			wantPurpose3EnforcePurpose:  trueStr,
			wantPurpose4EnforcePurpose:  trueStr,
			wantPurpose5EnforcePurpose:  trueStr,
			wantPurpose6EnforcePurpose:  trueStr,
			wantPurpose7EnforcePurpose:  trueStr,
			wantPurpose8EnforcePurpose:  trueStr,
			wantPurpose9EnforcePurpose:  trueStr,
			wantPurpose10EnforcePurpose: trueStr,
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

func TestMigrateConfigTCF2PurposeFlags(t *testing.T) {
	tests := []struct {
		description                string
		config                     []byte
		wantPurpose1EnforceAlgo    string
		wantPurpose1EnforcePurpose bool
		wantPurpose1Enabled        bool
	}{
		{
			description: "enforce_purpose does not set enforce_algo but sets enabled",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enforce_algo: "off"
                    enforce_purpose: "full"
                    enabled: false
                  purpose2:
                    enforce_purpose: "full"
                    enabled: false
                  purpose3:
                    enabled: false
            `),
			wantPurpose1EnforceAlgo:    "off",
			wantPurpose1EnforcePurpose: true,
			wantPurpose1Enabled:        true,
		},
		{
			description: "enforce_purpose sets enforce_algo and enabled",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enforce_purpose: "full"
                    enabled: false
            `),
			wantPurpose1EnforceAlgo:    "full",
			wantPurpose1EnforcePurpose: true,
			wantPurpose1Enabled:        true,
		},
		{
			description: "enforce_purpose does not set enforce_algo or enabled",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enabled: false
            `),
			wantPurpose1EnforceAlgo:    "",
			wantPurpose1EnforcePurpose: false,
			wantPurpose1Enabled:        false,
		},
	}

	for _, tt := range tests {
		v := viper.New()
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer(tt.config))

		migrateConfigTCF2PurposeFlags(v)

		assert.Equal(t, tt.wantPurpose1EnforceAlgo, v.GetString("gdpr.tcf2.purpose1.enforce_algo"), tt.description)
		assert.Equal(t, tt.wantPurpose1EnforcePurpose, v.GetBool("gdpr.tcf2.purpose1.enforce_purpose"), tt.description)
		assert.Equal(t, tt.wantPurpose1Enabled, v.GetBool("gdpr.tcf2.purpose1.enabled"), tt.description)
	}

}

func TestMigrateConfigTCF2EnforcePurposeFlags(t *testing.T) {
	trueStr := "true"
	falseStr := "false"

	tests := []struct {
		description                 string
		config                      []byte
		wantEnforceAlgosSet         bool
		wantPurpose1EnforceAlgo     string
		wantPurpose2EnforceAlgo     string
		wantPurpose3EnforceAlgo     string
		wantPurpose4EnforceAlgo     string
		wantPurpose5EnforceAlgo     string
		wantPurpose6EnforceAlgo     string
		wantPurpose7EnforceAlgo     string
		wantPurpose8EnforceAlgo     string
		wantPurpose9EnforceAlgo     string
		wantPurpose10EnforceAlgo    string
		wantEnforcePurposesSet      bool
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
	}{
		{
			description:            "enforce_algo and enforce_purpose are not set",
			config:                 []byte{},
			wantEnforceAlgosSet:    false,
			wantEnforcePurposesSet: false,
		},
		{
			description: "enforce_algo not set; set it based on enforce_purpose string value",
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
			wantEnforceAlgosSet:         true,
			wantPurpose1EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose2EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose3EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose4EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose5EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose6EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose7EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose8EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose9EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose10EnforceAlgo:    TCF2FullEnforcement,
			wantEnforcePurposesSet:      true,
			wantPurpose1EnforcePurpose:  trueStr,
			wantPurpose2EnforcePurpose:  falseStr,
			wantPurpose3EnforcePurpose:  trueStr,
			wantPurpose4EnforcePurpose:  falseStr,
			wantPurpose5EnforcePurpose:  trueStr,
			wantPurpose6EnforcePurpose:  falseStr,
			wantPurpose7EnforcePurpose:  trueStr,
			wantPurpose8EnforcePurpose:  falseStr,
			wantPurpose9EnforcePurpose:  trueStr,
			wantPurpose10EnforcePurpose: falseStr,
		},
		{
			description: "enforce_algo not set; don't set it based on enforce_purpose bool value",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enforce_purpose: true
                  purpose2:
                    enforce_purpose: false
                  purpose3:
                    enforce_purpose: true
                  purpose4:
                    enforce_purpose: false
                  purpose5:
                    enforce_purpose: true
                  purpose6:
                    enforce_purpose: false
                  purpose7:
                    enforce_purpose: true
                  purpose8:
                    enforce_purpose: false
                  purpose9:
                    enforce_purpose: true
                  purpose10:
                    enforce_purpose: false
            `),
			wantEnforceAlgosSet:         false,
			wantEnforcePurposesSet:      true,
			wantPurpose1EnforcePurpose:  trueStr,
			wantPurpose2EnforcePurpose:  falseStr,
			wantPurpose3EnforcePurpose:  trueStr,
			wantPurpose4EnforcePurpose:  falseStr,
			wantPurpose5EnforcePurpose:  trueStr,
			wantPurpose6EnforcePurpose:  falseStr,
			wantPurpose7EnforcePurpose:  trueStr,
			wantPurpose8EnforcePurpose:  falseStr,
			wantPurpose9EnforcePurpose:  trueStr,
			wantPurpose10EnforcePurpose: falseStr,
		},
		{
			description: "enforce_algo is set and enforce_purpose is not; enforce_algo is unchanged",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enforce_algo: "full"
                  purpose2:
                    enforce_algo: "full"
                  purpose3:
                    enforce_algo: "full"
                  purpose4:
                    enforce_algo: "full"
                  purpose5:
                    enforce_algo: "full"
                  purpose6:
                    enforce_algo: "full"
                  purpose7:
                    enforce_algo: "full"
                  purpose8:
                    enforce_algo: "full"
                  purpose9:
                    enforce_algo: "full"
                  purpose10:
                    enforce_algo: "full"
            `),
			wantEnforceAlgosSet:      true,
			wantPurpose1EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose2EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose3EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose4EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose5EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose6EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose7EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose8EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose9EnforceAlgo:  TCF2FullEnforcement,
			wantPurpose10EnforceAlgo: TCF2FullEnforcement,
			wantEnforcePurposesSet:   false,
		},
		{
			description: "enforce_algo and enforce_purpose are set; enforce_algo is unchanged",
			config: []byte(`
              gdpr:
                tcf2:
                  purpose1:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose2:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose3:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose4:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose5:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose6:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose7:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose8:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose9:
                    enforce_algo: "full"
                    enforce_purpose: "no"
                  purpose10:
                    enforce_algo: "full"
                    enforce_purpose: "no"
            `),
			wantEnforceAlgosSet:         true,
			wantPurpose1EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose2EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose3EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose4EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose5EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose6EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose7EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose8EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose9EnforceAlgo:     TCF2FullEnforcement,
			wantPurpose10EnforceAlgo:    TCF2FullEnforcement,
			wantEnforcePurposesSet:      true,
			wantPurpose1EnforcePurpose:  falseStr,
			wantPurpose2EnforcePurpose:  falseStr,
			wantPurpose3EnforcePurpose:  falseStr,
			wantPurpose4EnforcePurpose:  falseStr,
			wantPurpose5EnforcePurpose:  falseStr,
			wantPurpose6EnforcePurpose:  falseStr,
			wantPurpose7EnforcePurpose:  falseStr,
			wantPurpose8EnforcePurpose:  falseStr,
			wantPurpose9EnforcePurpose:  falseStr,
			wantPurpose10EnforcePurpose: falseStr,
		},
	}

	for _, tt := range tests {
		v := viper.New()
		v.SetConfigType("yaml")
		v.ReadConfig(bytes.NewBuffer(tt.config))

		migrateConfigTCF2EnforcePurposeFlags(v)

		if tt.wantEnforceAlgosSet {
			assert.Equal(t, tt.wantPurpose1EnforceAlgo, v.GetString("gdpr.tcf2.purpose1.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose2EnforceAlgo, v.GetString("gdpr.tcf2.purpose2.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose3EnforceAlgo, v.GetString("gdpr.tcf2.purpose3.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose4EnforceAlgo, v.GetString("gdpr.tcf2.purpose4.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose5EnforceAlgo, v.GetString("gdpr.tcf2.purpose5.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose6EnforceAlgo, v.GetString("gdpr.tcf2.purpose6.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose7EnforceAlgo, v.GetString("gdpr.tcf2.purpose7.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose8EnforceAlgo, v.GetString("gdpr.tcf2.purpose8.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose9EnforceAlgo, v.GetString("gdpr.tcf2.purpose9.enforce_algo"), tt.description)
			assert.Equal(t, tt.wantPurpose10EnforceAlgo, v.GetString("gdpr.tcf2.purpose10.enforce_algo"), tt.description)
		} else {
			assert.Nil(t, v.Get("gdpr.tcf2.purpose1.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose2.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose3.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose4.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose5.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose6.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose7.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose8.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose9.enforce_algo"), tt.description)
			assert.Nil(t, v.Get("gdpr.tcf2.purpose10.enforce_algo"), tt.description)
		}

		if tt.wantEnforcePurposesSet {
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
		}
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
	cfg.GDPR.TCF2.Purpose1.EnforceAlgo = ""
	cfg.GDPR.TCF2.Purpose2.EnforceAlgo = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose3.EnforceAlgo = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose4.EnforceAlgo = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose5.EnforceAlgo = "invalid1"
	cfg.GDPR.TCF2.Purpose6.EnforceAlgo = "invalid2"
	cfg.GDPR.TCF2.Purpose7.EnforceAlgo = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose8.EnforceAlgo = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose9.EnforceAlgo = TCF2FullEnforcement
	cfg.GDPR.TCF2.Purpose10.EnforceAlgo = "invalid3"

	errs := cfg.validate(v)

	expectedErrs := []error{
		errors.New("gdpr.tcf2.purpose1.enforce_algo must be \"full\". Got "),
		errors.New("gdpr.tcf2.purpose5.enforce_algo must be \"full\". Got invalid1"),
		errors.New("gdpr.tcf2.purpose6.enforce_algo must be \"full\". Got invalid2"),
		errors.New("gdpr.tcf2.purpose10.enforce_algo must be \"full\". Got invalid3"),
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
	cfg.Accounts.Postgres.ConnectionInfo.Database = "accounts"

	errs := cfg.validate(v)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs, errors.New("accounts.postgres: retrieving accounts via postgres not available, use accounts.files"))
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

func TestTCF2PurposeVendorException(t *testing.T) {
	tests := []struct {
		description              string
		givePurposeConfigNil     bool
		givePurpose1ExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose2ExceptionMap map[openrtb_ext.BidderName]struct{}
		givePurpose              consentconstants.Purpose
		giveBidder               openrtb_ext.BidderName
		wantIsVendorException    bool
	}{
		{
			description:           "Purpose config is nil",
			givePurposeConfigNil:  true,
			givePurpose:           1,
			giveBidder:            "appnexus",
			wantIsVendorException: false,
		},
		{
			description:           "Nil - exception map not defined for purpose",
			givePurpose:           1,
			giveBidder:            "appnexus",
			wantIsVendorException: false,
		},
		{
			description:              "Empty - exception map empty for purpose",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{},
			giveBidder:               "appnexus",
			wantIsVendorException:    false,
		},
		{
			description:              "One - bidder found in purpose exception map containing one entry",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"appnexus": {}},
			giveBidder:               "appnexus",
			wantIsVendorException:    true,
		},
		{
			description:              "Many - bidder found in purpose exception map containing multiple entries",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			giveBidder:               "appnexus",
			wantIsVendorException:    true,
		},
		{
			description:              "Many - bidder not found in purpose exception map containing multiple entries",
			givePurpose:              1,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			givePurpose2ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
			giveBidder:               "openx",
			wantIsVendorException:    false,
		},
		{
			description:              "Many - bidder found in different purpose exception map containing multiple entries",
			givePurpose:              2,
			givePurpose1ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "index": {}},
			givePurpose2ExceptionMap: map[openrtb_ext.BidderName]struct{}{"rubicon": {}, "appnexus": {}, "openx": {}},
			giveBidder:               "openx",
			wantIsVendorException:    true,
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

		value := tcf2.PurposeVendorException(tt.givePurpose, tt.giveBidder)

		assert.Equal(t, tt.wantIsVendorException, value, tt.description)
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
