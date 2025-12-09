package trafficshaping

import (
	"context"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/modules/mile/common"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestBuildConfigURL(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		wrapper     *openrtb_ext.RequestWrapper
		expected    string
		expectError bool
	}{
		{
			name:     "valid desktop chrome US",
			baseURL:  "http://localhost:8080/ts-server/",
			wrapper:  createWrapper("US", 2, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
			expected: "http://localhost:8080/ts-server/US/w/chrome/ts.json",
		},
		{
			name:     "valid mobile chrome IN",
			baseURL:  "http://localhost:8080/ts-server/",
			wrapper:  createWrapper("IN", 4, "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"),
			expected: "http://localhost:8080/ts-server/IN/m/chrome/ts.json",
		},
		{
			name:     "valid tablet safari",
			baseURL:  "http://localhost:8080/ts-server/",
			wrapper:  createWrapper("US", 1, "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1"),
			expected: "http://localhost:8080/ts-server/US/t/safari/ts.json",
		},
		{
			name:     "valid desktop firefox",
			baseURL:  "http://localhost:8080/ts-server/",
			wrapper:  createWrapper("CA", 2, "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/120.0"),
			expected: "http://localhost:8080/ts-server/CA/w/ff/ts.json",
		},
		{
			name:     "valid desktop edge",
			baseURL:  "http://localhost:8080/ts-server/",
			wrapper:  createWrapper("GB", 2, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"),
			expected: "http://localhost:8080/ts-server/GB/w/edge/ts.json",
		},
		{
			name:     "valid mobile opera",
			baseURL:  "http://localhost:8080/ts-server/",
			wrapper:  createWrapper("FR", 4, "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 OPR/76.2.4027.73374"),
			expected: "http://localhost:8080/ts-server/FR/m/opera/ts.json",
		},
		{
			name:        "missing country",
			baseURL:     "http://localhost:8080/ts-server/",
			wrapper:     createWrapperNoGeo(2, "Chrome UA"),
			expectError: true,
		},
		{
			name:        "missing device",
			baseURL:     "http://localhost:8080/ts-server/",
			wrapper:     &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
			expectError: true,
		},
		{
			name:        "missing UA",
			baseURL:     "http://localhost:8080/ts-server/",
			wrapper:     createWrapper("US", 2, ""),
			expectError: true,
		},
		{
			name:        "nil wrapper",
			baseURL:     "http://localhost:8080/ts-server/",
			wrapper:     nil,
			expectError: true,
		},
		{
			name:     "lowercase country normalized to uppercase",
			baseURL:  "http://localhost:8080/ts-server/",
			wrapper:  createWrapper("us", 2, "Chrome UA"),
			expected: "http://localhost:8080/ts-server/US/w/chrome/ts.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := buildConfigURL(tt.baseURL, tt.wrapper)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, url)
			}
		})
	}
}

func TestBuildConfigURLWithFallback_DeviceFallback(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		expectedURL string
		expectError bool
	}{
		{
			name: "desktop fallback",
			wrapper: func() *openrtb_ext.RequestWrapper {
				w := createWrapper("US", 0, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0 Safari/537.36")
				w.Device.DeviceType = 0
				w.Device.Geo = nil
				w.Device.IP = "1.1.1.1" // IP required for geo fallback
				return w
			}(),
			expectedURL: "http://localhost:8080/ts-server/ts-server/US/w/chrome/ts.json",
		},
		{
			name: "mobile fallback",
			wrapper: func() *openrtb_ext.RequestWrapper {
				w := createWrapper("US", 0, "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Safari/604.1")
				w.Device.DeviceType = 0
				w.Device.Geo = nil
				w.Device.IP = "1.1.1.1" // IP required for geo fallback
				return w
			}(),
			expectedURL: "http://localhost:8080/ts-server/ts-server/US/m/safari/ts.json",
		},
		{
			name: "tablet fallback",
			wrapper: func() *openrtb_ext.RequestWrapper {
				w := createWrapper("US", 0, "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1")
				w.Device.DeviceType = 0
				w.Device.Geo = nil
				w.Device.IP = "1.1.1.1" // IP required for geo fallback
				return w
			}(),
			expectedURL: "http://localhost:8080/ts-server/ts-server/US/t/safari/ts.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			geoResolver := &mockGeoResolver{country: "US"}
			url, activities, err := buildConfigURLWithFallback(context.Background(), "http://localhost:8080/ts-server/", tt.wrapper, geoResolver)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedURL, url)

			hasDeviceDerived := false
			for _, activity := range activities {
				if activity.Name == "devicetype_derived" {
					hasDeviceDerived = true
					break
				}
			}
			assert.True(t, hasDeviceDerived, "expected devicetype_derived activity")
		})
	}
}

type mockGeoResolver struct {
	country string
	err     error
}

func (m *mockGeoResolver) Resolve(ctx context.Context, ip string) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.country, nil
}

// Ensure mockGeoResolver implements common.GeoResolver
var _ common.GeoResolver = (*mockGeoResolver)(nil)

// Helper functions for creating test wrappers

func createWrapper(country string, deviceType int64, ua string) *openrtb_ext.RequestWrapper {
	return &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Site: &openrtb2.Site{ID: "ts-server"},
			Device: &openrtb2.Device{
				DeviceType: adcom1.DeviceType(deviceType),
				UA:         ua,
				Geo: &openrtb2.Geo{
					Country: country,
				},
			},
		},
	}
}

func createWrapperNoGeo(deviceType int64, ua string) *openrtb_ext.RequestWrapper {
	return &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			Site: &openrtb2.Site{ID: "ts-server"},
			Device: &openrtb2.Device{
				DeviceType: adcom1.DeviceType(deviceType),
				UA:         ua,
			},
		},
	}
}

func TestExtractSiteID(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		expected    string
		expectError bool
	}{
		{
			name: "valid_site_id",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{ID: "ts-server"},
				},
			},
			expected:    "ts-server",
			expectError: false,
		},
		{
			name: "missing_site",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: nil,
				},
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "empty_site_id",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{ID: ""},
				},
			},
			expected:    "",
			expectError: true,
		},
		{
			name: "nil_bid_request",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: nil,
			},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			siteID, err := extractSiteID(tt.wrapper)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, siteID)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, siteID)
			}
		})
	}
}

func TestBuildConfigURLWithFallback_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		geoResolver common.GeoResolver
		expectError bool
	}{
		{
			name:        "nil_wrapper",
			wrapper:     nil,
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: true,
		},
		{
			name: "nil_bid_request",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: nil,
			},
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: true,
		},
		{
			name: "missing_site_id",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: nil,
				},
			},
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: true,
		},
		{
			name: "missing_country_and_geo_resolver_error",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{ID: "ts-server"},
					Device: &openrtb2.Device{
						IP: "1.1.1.1",
					},
				},
			},
			geoResolver: &mockGeoResolver{err: assert.AnError},
			expectError: true,
		},
		{
			name: "missing_country_but_geo_resolver_succeeds",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{ID: "ts-server"},
					Device: &openrtb2.Device{
						IP:         "1.1.1.1",
						UA:         "Mozilla/5.0 Chrome/120.0.0.0",
						DeviceType: adcom1.DeviceType(2),
					},
				},
			},
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: false,
		},
		{
			name: "missing_device_category_and_ua",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{ID: "ts-server"},
					Device: &openrtb2.Device{
						UA: "",
					},
				},
			},
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: true,
		},
		{
			name: "missing_browser",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Site: &openrtb2.Site{ID: "ts-server"},
					Device: &openrtb2.Device{
						UA: "",
					},
				},
			},
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, activities, err := buildConfigURLWithFallback(context.Background(), "http://localhost:8080/ts-server/", tt.wrapper, tt.geoResolver)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, url)
				_ = activities // Activities may or may not be populated
			}
		})
	}
}
