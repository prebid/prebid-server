package maxmind

import (
	"context"
	"testing"

	"github.com/prebid/prebid-server/v3/geolocation"

	"github.com/stretchr/testify/assert"
)

// File is only for testing purposes, never used in the production environment.
// File is taken from the official MaxMind repository.
// https://github.com/maxmind/MaxMind-DB/blob/main/test-data/GeoLite2-City-Test.mmdb
const testDataPath = "./test-data/GeoLite2-City.tar.gz"

const (
	testIP   = "2.125.160.216"
	testIPv6 = "2001:480::"
)

func TestGeoLocationNoReader(t *testing.T) {
	geo := &GeoLocation{}
	_, err := geo.Lookup(context.Background(), testIP)
	assert.Error(t, err, "should return error if data path is not set")
}

func TestGeoLocationSetDataPath(t *testing.T) {
	geo := &GeoLocation{}
	tests := []struct {
		name   string
		path   string
		failed bool
	}{
		{
			"File not exists",
			"no_file",
			true,
		},
		{
			"File is not a tar.gz archive",
			"./test-data/nothing.mmdb",
			true,
		},
		{
			"Archive does not contain GeoLite2-City.mmdb",
			"./test-data/nothing.tar.gz",
			true,
		},
		{
			"Archive contains GeoLite2-City.mmdb, but GeoLite2-City.mmdb has bad data",
			"./test-data/GeoLite2-City-Bad-Data.tar.gz",
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := geo.SetDataPath(test.path)
			if test.failed {
				assert.Error(t, err, "data path %s should return error", test.path)
			} else {
				assert.NoError(t, err, "data path %s should not return error", test.path)
			}
		})
	}
}

func TestGeoLocationLookup(t *testing.T) {
	geo := &GeoLocation{}
	err := geo.SetDataPath(testDataPath)
	assert.NoError(t, err, "geolocation should load data from %s", testDataPath)

	tests := []struct {
		name     string
		ip       string
		expected *geolocation.GeoInfo
		failed   bool
	}{
		{
			"Lookup empty IP",
			"",
			nil,
			true,
		},
		{
			"Lookup incorrect IP",
			"bad ip",
			nil,
			true,
		},
		{
			"Lookup valid IPv4",
			testIP,
			&geolocation.GeoInfo{
				Vendor:     Vendor,
				Continent:  "EU",
				Country:    "GB",
				Region:     "ENG",
				RegionCode: 0,
				City:       "Boxford",
				Zip:        "OX1",
				Lat:        51.75,
				Lon:        -1.25,
				TimeZone:   "Europe/London",
			},
			false,
		},
		{
			"Lookup valid IPv6",
			testIPv6,
			&geolocation.GeoInfo{
				Vendor:     Vendor,
				Continent:  "NA",
				Country:    "US",
				Region:     "CA",
				RegionCode: 0,
				City:       "San Diego",
				Zip:        "92101",
				Lat:        32.7203,
				Lon:        -117.1552,
				TimeZone:   "America/Los_Angeles",
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			geoInfo, err := geo.Lookup(context.Background(), test.ip)
			if test.failed {
				assert.Error(t, err, "geolocation lookup should return error. IP: %s", test.ip)
			} else {
				assert.NoError(t, err, "geolocation lookup should not return error. IP: %s", test.ip)
				assert.Equal(t, test.expected, geoInfo, "geolocation should be equal. IP: %s", test.ip)
			}
		})
	}
}

func TestGeoLocationReaderClosed(t *testing.T) {
	geo := &GeoLocation{}
	geo.SetDataPath(testDataPath)
	geo.reader.Load().Close()
	_, err := geo.Lookup(context.Background(), testIP)
	assert.Error(t, err, "should return error if reader is closed")
}
