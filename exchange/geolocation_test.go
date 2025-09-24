package exchange

import (
	"context"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/geolocation"
	"github.com/prebid/prebid-server/v3/geolocation/countrycodemapper"
	"github.com/prebid/prebid-server/v3/geolocation/geolocationtest"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

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
			resolver := NewGeoLocationResolver(test.geoloc)
			geo, err := resolver.Lookup(context.Background(), test.ip, test.country)
			if test.geoErr {
				assert.Error(t, err, "geolocation should return error")
			} else {
				assert.NoError(t, err, "geolocation should not return error. Error: %v", err)
				assert.Equal(t, test.geoCountry, geo.Country)
			}
		})
	}
}

func TestEnrichGeoLocation(t *testing.T) {
	countrycodemapper.Load(`
CN,CHN
US,USA
`)

	resolver := makeMockGeoLocationResolver(map[string]mockGeoLocationResolverResult{
		"1.1.1.1": {
			geo: &geolocation.GeoInfo{Country: "CN", Region: "CN-SH", TimeZone: "Asia/Shanghai"},
			err: nil,
		},
		"1111:2222:3333:4400::": {
			geo: &geolocation.GeoInfo{Country: "CN", Region: "CN-SC", TimeZone: "UTC"},
			err: nil,
		},
		"2.2.2.2": {
			geo: &geolocation.GeoInfo{Country: "US", Region: "US-HI", TimeZone: "Pacific/Honolulu"},
			err: nil,
		},
		"3.3.3.3": {
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
			name: "GeoLocation is disabled",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1"},
			},
			account:           config.Account{GeoLocation: config.AccountGeoLocation{Enabled: false}},
			resolver:          resolver,
			expectedCountry:   "",
			expectedRegion:    "",
			expectedUTCOffset: 0,
			errsCount:         0,
		},
		{
			name: "GeoLocation is enabled, IPv4 is used",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1", IPv6: "1111:2222:3333:4400::"},
			},
			account:           config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver:          resolver,
			expectedCountry:   "CHN",
			expectedRegion:    "CN-SH",
			expectedUTCOffset: 480,
			errsCount:         0,
		},
		{
			name: "GeoLocation is enabled, IPv6 is used",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IPv6: "1111:2222:3333:4400::"},
			},
			account:           config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver:          resolver,
			expectedCountry:   "CHN",
			expectedRegion:    "CN-SC",
			expectedUTCOffset: 0,
			errsCount:         0,
		},
		{
			name:              "Device is nil",
			req:               &openrtb2.BidRequest{},
			account:           config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver:          resolver,
			expectedCountry:   "",
			expectedRegion:    "",
			expectedUTCOffset: 0,
			errsCount:         1,
		},
		{
			name: "Country exists",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "1.1.1.1", Geo: &openrtb2.Geo{Country: "USA"}},
			},
			account:           config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver:          resolver,
			expectedCountry:   "USA",
			expectedRegion:    "",
			expectedUTCOffset: 0,
			errsCount:         1,
		},
		{
			name: "Remove 'US-' prefix if country is US",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "2.2.2.2"},
			},
			account:           config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver:          resolver,
			expectedCountry:   "USA",
			expectedRegion:    "HI",
			expectedUTCOffset: -600,
			errsCount:         0,
		},
		{
			name: "Resolver returns error",
			req: &openrtb2.BidRequest{
				Device: &openrtb2.Device{IP: "3.3.3.3"},
			},
			account:           config.Account{GeoLocation: config.AccountGeoLocation{Enabled: true}},
			resolver:          resolver,
			expectedCountry:   "",
			expectedRegion:    "",
			expectedUTCOffset: 0,
			errsCount:         1,
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

func TestCountryFromDevice(t *testing.T) {
	tests := []struct {
		device  *openrtb2.Device
		country string
	}{
		{nil, ""},
		{&openrtb2.Device{}, ""},
		{&openrtb2.Device{Geo: &openrtb2.Geo{}}, ""},
		{&openrtb2.Device{Geo: &openrtb2.Geo{Country: "USA"}}, "USA"},
	}

	for _, test := range tests {
		assert.Equal(t, test.country, countryFromDevice(test.device))
	}
}

func TestUpdateDeviceGeo(t *testing.T) {
	countrycodemapper.Load("CN,CHN\n")

	tests := []struct {
		name           string
		device         *openrtb2.Device
		geoinfo        *geolocation.GeoInfo
		expectedDevice *openrtb2.Device
	}{
		{
			name:           "Update device if device is nil",
			device:         nil,
			geoinfo:        &geolocation.GeoInfo{Country: "CN"},
			expectedDevice: nil,
		},
		{
			name:           "Update device if geoinfo is nil",
			device:         &openrtb2.Device{Geo: &openrtb2.Geo{}},
			geoinfo:        nil,
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{}},
		},
		{
			name:           "Update device with empty geo",
			device:         &openrtb2.Device{},
			geoinfo:        &geolocation.GeoInfo{Country: "CN", Region: "CN-SH", TimeZone: "Asia/Shanghai"},
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "CHN", Region: "CN-SH", UTCOffset: 480}},
		},
		{
			name:           "Update device with bad geo info",
			device:         &openrtb2.Device{Geo: &openrtb2.Geo{Country: "CHN", Region: "CN-CQ", UTCOffset: 420}},
			geoinfo:        &geolocation.GeoInfo{Country: "", Region: "", TimeZone: "UNKNOWN"},
			expectedDevice: &openrtb2.Device{Geo: &openrtb2.Geo{Country: "CHN", Region: "CN-CQ", UTCOffset: 420}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := &openrtb2.BidRequest{Device: test.device}
			updateDeviceGeo(req, test.geoinfo)
			expected, _ := jsonutil.Marshal(test.expectedDevice)
			updated, _ := jsonutil.Marshal(req.Device)
			assert.Equal(t, string(expected), string(updated), "device should be %s", string(expected))
		})
	}
}
