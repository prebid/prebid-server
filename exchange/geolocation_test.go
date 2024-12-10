package exchange

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/prebid/go-gdpr/vendorconsent"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/config/countrycode"
	"github.com/prebid/prebid-server/v3/gdpr"
	"github.com/prebid/prebid-server/v3/geolocation"
	"github.com/prebid/prebid-server/v3/geolocation/geolocationtest"
	"github.com/prebid/prebid-server/v3/metrics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

type mockMetrics struct {
	metrics.MetricsEngineMock
	success int64
	fail    int64
}

func (m *mockMetrics) RecordGeoLocationRequest(success bool) {
	if success {
		atomic.AddInt64(&m.success, 1)
	} else {
		atomic.AddInt64(&m.fail, 1)
	}
}

type mockGeoLocationResolverResult struct {
	geo *geolocation.GeoInfo
	err error
}

type mockGeoLocationResolver struct {
	data map[string]mockGeoLocationResolverResult
}

func (g *mockGeoLocationResolver) Lookup(ctx context.Context, ip string, country string) (*geolocation.GeoInfo, error) {
	if ip == "" || country != "" {
		return &geolocation.GeoInfo{}, assert.AnError
	}

	if g.data == nil {
		return &geolocation.GeoInfo{}, nil
	}
	if result, ok := g.data[ip]; ok {
		return result.geo, result.err
	}
	return &geolocation.GeoInfo{}, nil
}

func makeMockGeoLocationResolver(data map[string]mockGeoLocationResolverResult) GeoLocationResolver {
	return &mockGeoLocationResolver{data: data}
}

func TestGeoLocationResolver(t *testing.T) {
	me := &mockMetrics{}
	geoservice := geolocationtest.NewMockGeoLocation(map[string]*geolocation.GeoInfo{
		"1.1.1.1": {Country: "CN"},
		"1.1.1.2": {Country: "US"},
	})
	tests := []struct {
		name       string
		geoloc     geolocation.GeoLocation
		ip         string
		country    string
		geoCountry string
		geoErr     bool
	}{
		{
			"Resolver is nil",
			nil, "1.1.1.1", "", "", true,
		},
		{
			"Lookup empty IP",
			geoservice, "", "", "", true,
		},
		{
			"Lookup valid IP, country has value",
			geoservice, "1.1.1.1", "CN", "", true,
		},
		{
			"Lookup unknown IP",
			geoservice, "2.2.2.2", "", "", true,
		},
		{
			"Lookup successful, response country is CN",
			geoservice, "1.1.1.1", "", "CN", false,
		},
		{
			"Lookup successful, response country is US",
			geoservice, "1.1.1.2", "", "US", false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver := NewGeoLocationResolver(test.geoloc, me)
			geo, err := resolver.Lookup(context.Background(), test.ip, test.country)
			if test.geoErr {
				assert.Error(t, err, "geolocation should return error")
			} else {
				assert.NoError(t, err, "geolocation should not return error. Error: %v", err)
				assert.Equal(t, test.geoCountry, geo.Country)
			}
		})
	}

	assert.Equal(t, int64(2), me.success, "metrics success count should be 2")
	assert.Equal(t, int64(1), me.fail, "metrics fail count should be 1")
}

func TestEnrichGeoLocation(t *testing.T) {
	countrycode.Load("CN,CHN\n")

	resolver := makeMockGeoLocationResolver(map[string]mockGeoLocationResolverResult{
		"1.1.1.1": {
			geo: &geolocation.GeoInfo{Country: "CN", Region: "Shanghai", TimeZone: "Asia/Shanghai"},
			err: nil,
		},
		"1111:2222:3333:4400::": {
			geo: &geolocation.GeoInfo{Country: "CN", Region: "Sichuan", TimeZone: "UTC"},
			err: nil,
		},
		"2.2.2.2": {
			geo: nil,
			err: assert.AnError,
		},
	})
	tests := []struct {
		name              string
		req               *openrtb2.BidRequest
		account           config.Account
		resolver          GeoLocationResolver
		expectedCountry   string
		expectedRegion    string
		expectedUTCOffset int64
		errsCount         int
	}{
		{
			"Enrich device. geoLocation is disabled",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1"},
			},
			config.Account{GeoLocation: config.AccountGeoLocation{Enabled: false}},
			resolver,
			"",
			"",
			0,
			0,
		},
		{
			"Enrich device. GeoLocation is enabled, IPv4 is used",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1", IPv6: "1111:2222:3333:4400::"},
			},
			config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver,
			"CHN",
			"Shanghai",
			480,
			0,
		},
		{
			"Enrich device. GeoLocation is enabled, IPv6 is used",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IPv6: "1111:2222:3333:4400::"},
			},
			config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver,
			"CHN",
			"Sichuan",
			0,
			0,
		},
		{
			"Enrich device. device is nil",
			&openrtb2.BidRequest{},
			config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver,
			"",
			"",
			0,
			1,
		},
		{
			"Enrich device. country exists",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1", Geo: &openrtb2.Geo{Country: "USA"}},
			},
			config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver,
			"USA",
			"",
			0,
			1,
		},
		{
			"Enrich device. resolver returns error",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "2.2.2.2"},
			},
			config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver,
			"",
			"",
			0,
			1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &openrtb_ext.RequestWrapper{BidRequest: test.req}
			errs := EnrichGeoLocation(context.Background(), req, test.account, test.resolver)
			assert.Equal(t, test.errsCount, len(errs), "errors count should be %d", test.errsCount)

			var (
				country   string
				region    string
				utcoffset int64
			)
			if req.BidRequest.Device != nil && req.BidRequest.Device.Geo != nil {
				country = req.BidRequest.Device.Geo.Country
				region = req.BidRequest.Device.Geo.Region
				utcoffset = req.BidRequest.Device.Geo.UTCOffset
			}
			assert.Equal(t, test.expectedCountry, country, "country should be %s", test.expectedCountry)
			assert.Equal(t, test.expectedRegion, region, "region should be %s", test.expectedRegion)
			assert.Equal(t, test.expectedUTCOffset, utcoffset, "utc offset should be %d", test.expectedUTCOffset)
		})
	}
}

func TestEnrichGeoLocationWithPrivacy(t *testing.T) {
	countrycode.Load("CN,CHN\n")

	resolver := makeMockGeoLocationResolver(map[string]mockGeoLocationResolverResult{
		"1.1.1.0": {
			geo: &geolocation.GeoInfo{Country: "CN", Region: "", TimeZone: "UTC"},
			err: nil,
		},
		"1.1.1.1": {
			geo: &geolocation.GeoInfo{Country: "CN", Region: "Shanghai", TimeZone: "Asia/Shanghai"},
			err: nil,
		},
		"1111:2222:3333:4400::": {
			geo: &geolocation.GeoInfo{Country: "CN", Region: "Sichuan", TimeZone: "UTC"},
			err: nil,
		},
		"2.2.2.2": {
			geo: nil,
			err: assert.AnError,
		},
	})
	tests := []struct {
		name              string
		req               *openrtb2.BidRequest
		account           config.Account
		resolver          GeoLocationResolver
		requestPrivacy    *RequestPrivacy
		tcf2config        gdpr.TCF2ConfigReader
		expectedCountry   string
		expectedRegion    string
		expectedUTCOffset int64
		errsCount         int
	}{
		{
			"Enrich device. geoLocation is disabled",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1"},
			},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: false},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: false, LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"",
			"",
			0,
			0,
		},
		{
			"Enrich device. GDPR not enforced, LMT not enforced, device is nil",
			&openrtb2.BidRequest{},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: true},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: false, LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"",
			"",
			0,
			1,
		},
		{
			"Enrich device. GDPR not enforced, LMT not enforced, resolver returns error",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "2.2.2.2"},
			},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: true},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: false, LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"",
			"",
			0,
			1,
		},
		{
			"Enrich device. GDPR enforced, LMT not enforced, should not enrich",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1"},
			},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: true},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: true, LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"",
			"",
			0,
			0,
		},
		{
			"Enrich device. GDPR not enforced, LMT not enforced, country exists",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1", Geo: &openrtb2.Geo{Country: "USA"}},
			},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: true},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: false, LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"USA",
			"",
			0,
			1,
		},
		{
			"Enrich device. GDPR not enforced, LMT not enforced, IPv4 is used",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1", IPv6: "1111:2222:3333:4400::"},
			},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: true},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: false, LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"CHN",
			"Shanghai",
			480,
			0,
		},
		{
			"Enrich device. GDPR not enforced, LMT enforced, IPv4 is used",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1", IPv6: "1111:2222:3333:4400::"},
			},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: true},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: false, LMTEnforced: true},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"CHN",
			"",
			0,
			0,
		},
		{
			"Enrich device. GDPR not enforced, LMT enforced, IPv6 is used",
			&openrtb2.BidRequest{
				Device: &openrtb2.Device{IPv6: "1111:2222:3333:4400::"},
			},
			config.Account{
				Privacy:     config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}, IPv6Config: config.IPv6{AnonKeepBits: 56}},
				GeoLocation: config.AccountGeoLocation{Enabled: true},
			},
			resolver,
			&RequestPrivacy{GDPREnforced: false, LMTEnforced: true},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"CHN",
			"Sichuan",
			0,
			0,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &openrtb_ext.RequestWrapper{BidRequest: test.req}
			errs := EnrichGeoLocationWithPrivacy(context.Background(), req, test.account, test.resolver, test.requestPrivacy, test.tcf2config)
			assert.Equal(t, test.errsCount, len(errs), "errors count should be %d", test.errsCount)

			var (
				country   string
				region    string
				utcoffset int64
			)
			if req.BidRequest.Device != nil && req.BidRequest.Device.Geo != nil {
				country = req.BidRequest.Device.Geo.Country
				region = req.BidRequest.Device.Geo.Region
				utcoffset = req.BidRequest.Device.Geo.UTCOffset
			}
			assert.Equal(t, test.expectedCountry, country, "country should be %s", test.expectedCountry)
			assert.Equal(t, test.expectedRegion, region, "region should be %s", test.expectedRegion)
			assert.Equal(t, test.expectedUTCOffset, utcoffset, "utc offset should be %d", test.expectedUTCOffset)
		})
	}
}

func TestCountryFromDevice(t *testing.T) {
	tests := []struct {
		device  *openrtb2.Device
		country string
	}{
		{nil, ""},
		{&openrtb2.Device{}, ""},
		{&openrtb2.Device{Geo: &openrtb2.Geo{}}, ""},
		{&openrtb2.Device{Geo: &openrtb2.Geo{Country: "US"}}, "US"},
	}

	for _, test := range tests {
		assert.Equal(t, test.country, countryFromDevice(test.device))
	}
}

func TestMaybeMaskIP(t *testing.T) {
	tests := []struct {
		name           string
		device         *openrtb2.Device
		accountPrivacy config.AccountPrivacy
		reqPrivacy     *RequestPrivacy
		tcf2Config     gdpr.TCF2ConfigReader
		output         string
	}{
		{
			"Device is nil, ip should be empty",
			nil,
			config.AccountPrivacy{},
			&RequestPrivacy{LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"",
		},
		{
			"IPv4 and IPv6 both empty, ip should be empty",
			&openrtb2.Device{IP: "", IPv6: ""},
			config.AccountPrivacy{},
			&RequestPrivacy{LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"",
		},
		{
			"IPv4 with no privacy",
			&openrtb2.Device{IP: "1.1.1.1", IPv6: "1111:2222:3333:4444:5555:6666:7777:8888"},
			config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}},
			&RequestPrivacy{LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"1.1.1.1",
		},
		{
			"IPv6 with no privacy",
			&openrtb2.Device{IPv6: "1111:2222:3333:4444:5555:6666:7777:8888"},
			config.AccountPrivacy{IPv6Config: config.IPv6{AnonKeepBits: 56}},
			&RequestPrivacy{LMTEnforced: false},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"1111:2222:3333:4444:5555:6666:7777:8888",
		},
		{
			"IPv4 and IPv6 with privacy, IPv4 is preferred",
			&openrtb2.Device{IP: "1.1.1.1", IPv6: "1111:2222:3333:4444:5555:6666:7777:8888"},
			config.AccountPrivacy{IPv4Config: config.IPv4{AnonKeepBits: 24}},
			&RequestPrivacy{LMTEnforced: true},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"1.1.1.0",
		},
		{
			"IPv6 with privacy",
			&openrtb2.Device{IPv6: "1111:2222:3333:4444:5555:6666:7777:8888"},
			config.AccountPrivacy{IPv6Config: config.IPv6{AnonKeepBits: 56}},
			&RequestPrivacy{LMTEnforced: true},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			"1111:2222:3333:4400::",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ip := maybeMaskIP(test.device, test.accountPrivacy, test.reqPrivacy, test.tcf2Config)
			assert.Equal(t, test.output, ip)
		})
	}
}

func TestShouldMaskIP(t *testing.T) {
	tests := []struct {
		desc       string
		reqPrivacy *RequestPrivacy
		tcf2Config gdpr.TCF2ConfigReader
		output     bool
	}{
		{
			"Nothing enforced",
			&RequestPrivacy{
				COPPAEnforced: false,
				LMTEnforced:   false,
				Consent:       "",
			},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			false,
		},
		{
			"COPPA enforced",
			&RequestPrivacy{
				COPPAEnforced: true,
				LMTEnforced:   false,
				Consent:       "",
			},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			true,
		},
		{
			"LMT enforced",
			&RequestPrivacy{
				COPPAEnforced: false,
				LMTEnforced:   true,
				Consent:       "",
			},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			true,
		},
		{
			"TCF2 without SP1 consent enforced",
			&RequestPrivacy{
				COPPAEnforced: false,
				LMTEnforced:   false,
				Consent:       "CPuKGCPPuKGCPNEAAAENCZCAAAAAAAAAAAAAAAAAAAAA",
			},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			true,
		},
		{
			"TCF2 with SP1 consent enforced",
			&RequestPrivacy{
				COPPAEnforced: false,
				LMTEnforced:   false,
				Consent:       "CQDkxqbQDkxqbHcAAAENCZCIAAAAAAAAAAAAAAAAAAAA.II7Nd_X__bX9n-_7_6ft0eY1f9_r37uQzDhfNs-8F3L_W_LwX32E7NF36tq4KmR4ku1bBIQNtHMnUDUmxaolVrzHsak2cpyNKJ_JkknsZe2dYGF9Pn9lD-YKZ7_5_9_f52T_9_9_-39z3_9f___dv_-__-vjf_599n_v9fV_78_Kf9______-____________8A",
			},
			gdpr.NewTCF2Config(config.TCF2{}, config.AccountGDPR{}),
			false,
		},
		{
			"TCF2 with SP1 host enforced",
			&RequestPrivacy{
				COPPAEnforced: false,
				LMTEnforced:   false,
				Consent:       "",
			},
			gdpr.NewTCF2Config(
				config.TCF2{SpecialFeature1: config.TCF2SpecialFeature{Enforce: true}},
				config.AccountGDPR{},
			),
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			if test.reqPrivacy.Consent != "" {
				parsedConsent, err := vendorconsent.ParseString(test.reqPrivacy.Consent)
				assert.NoError(t, err, "Failed to parse consent string")
				test.reqPrivacy.ParsedConsent = parsedConsent
			}
			assert.Equal(t, test.output, shouldMaskIP(test.reqPrivacy, test.tcf2Config))
		})
	}
}

func TestUpdateDeviceGeo(t *testing.T) {
	countrycode.Load("CN,CHN\n")

	tests := []struct {
		device         *openrtb2.Device
		geoinfo        *geolocation.GeoInfo
		expectedDevice *openrtb2.Device
	}{
		{
			nil,
			&geolocation.GeoInfo{Country: "CN"},
			nil,
		},
		{
			&openrtb2.Device{},
			nil,
			&openrtb2.Device{},
		},
		{
			&openrtb2.Device{Geo: &openrtb2.Geo{}},
			nil,
			&openrtb2.Device{Geo: &openrtb2.Geo{}},
		},
		{
			&openrtb2.Device{},
			&geolocation.GeoInfo{Country: "CN", Region: "Shanghai", TimeZone: "Asia/Shanghai"},
			&openrtb2.Device{Geo: &openrtb2.Geo{Country: "CHN", Region: "Shanghai", UTCOffset: 480}},
		},
		// bad geo info
		{
			&openrtb2.Device{Geo: &openrtb2.Geo{Country: "CN", Region: "Chongqing", UTCOffset: 420}},
			&geolocation.GeoInfo{Country: "", Region: "", TimeZone: "UNKNOWN"},
			&openrtb2.Device{Geo: &openrtb2.Geo{Country: "CN", Region: "Chongqing", UTCOffset: 420}},
		},
	}

	for _, test := range tests {
		req := &openrtb2.BidRequest{Device: test.device}
		updateDeviceGeo(req, test.geoinfo)
		expected, _ := jsonutil.Marshal(test.expectedDevice)
		updated, _ := jsonutil.Marshal(req.Device)
		assert.Equal(t, string(expected), string(updated), "device should be %s", string(expected))
	}
}
