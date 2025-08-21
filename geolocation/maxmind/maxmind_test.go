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
		name string
		path string
		ok   bool
	}{
		{
			name: "File exists",
			path: "./test-data/GeoLite2-City.tar.gz",
			ok:   true,
		},
		{
			name: "File not exists",
			path: "no_file",
			ok:   false,
		},
		{
			name: "File is not a tar.gz archive",
			path: "./test-data/nothing.mmdb",
			ok:   false,
		},
		{
			name: "Archive does not contain GeoLite2-City.mmdb",
			path: "./test-data/nothing.tar.gz",
			ok:   false,
		},
		{
			name: "Archive contains GeoLite2-City.mmdb, but GeoLite2-City.mmdb has bad data",
			path: "./test-data/GeoLite2-City-Bad-Data.tar.gz",
			ok:   false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := geo.SetDataPath(test.path)
			if test.ok {
				assert.NoError(t, err, "data path %s should not return error", test.path)
			} else {
				assert.Error(t, err, "data path %s should return error", test.path)
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
		ok       bool
	}{
		{
			name:     "Lookup empty IP",
			ip:       "",
			expected: nil,
			ok:       false,
		},
		{
			name:     "Lookup incorrect IP",
			ip:       "bad ip",
			expected: nil,
			ok:       false,
		},
		{
			name: "Lookup valid IPv4",
			ip:   testIP,
			expected: &geolocation.GeoInfo{
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
			ok: true,
		},
		{
			name: "Lookup valid IPv6",
			ip:   testIPv6,
			expected: &geolocation.GeoInfo{
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
			ok: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			geoInfo, err := geo.Lookup(context.Background(), test.ip)
			if test.ok {
				assert.NoError(t, err, "geolocation lookup should not return error. IP: %s", test.ip)
				assert.Equal(t, test.expected, geoInfo, "geolocation should be equal. IP: %s", test.ip)
			} else {
				assert.Error(t, err, "geolocation lookup should return error. IP: %s", test.ip)
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
