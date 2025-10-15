package trafficshaping

import (
	"context"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
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

func TestExtractCountry(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		expected    string
		expectError bool
	}{
		{
			name:     "valid uppercase country",
			wrapper:  createWrapper("US", 2, "UA"),
			expected: "US",
		},
		{
			name:     "valid lowercase country",
			wrapper:  createWrapper("in", 2, "UA"),
			expected: "IN",
		},
		{
			name:     "valid mixed case country",
			wrapper:  createWrapper("Gb", 2, "UA"),
			expected: "GB",
		},
		{
			name:        "missing geo",
			wrapper:     createWrapperNoGeo(2, "UA"),
			expectError: true,
		},
		{
			name:        "empty country",
			wrapper:     createWrapper("", 2, "UA"),
			expectError: true,
		},
		{
			name:        "invalid country code (too long)",
			wrapper:     createWrapper("USA", 2, "UA"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			country, err := extractCountry(tt.wrapper)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, country)
			}
		})
	}
}

func TestExtractDeviceCategory(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		expected    string
		expectError bool
	}{
		{
			name:     "device type 1 (mobile/tablet) with phone UA",
			wrapper:  createWrapper("US", 1, "Mozilla/5.0 (Linux; Android) Mobile"),
			expected: "m",
		},
		{
			name:     "device type 1 with iPad UA",
			wrapper:  createWrapper("US", 1, "Mozilla/5.0 (iPad; CPU OS 14_0) Safari/604.1"),
			expected: "t",
		},
		{
			name:     "device type 2 (PC)",
			wrapper:  createWrapper("US", 2, "Mozilla/5.0 (Windows NT 10.0) Chrome"),
			expected: "w",
		},
		{
			name:     "device type 3 (Connected TV)",
			wrapper:  createWrapper("US", 3, "Smart TV UA"),
			expected: "t",
		},
		{
			name:     "device type 4 (Phone)",
			wrapper:  createWrapper("US", 4, "iPhone UA"),
			expected: "m",
		},
		{
			name:     "device type 5 (Tablet)",
			wrapper:  createWrapper("US", 5, "Tablet UA"),
			expected: "t",
		},
		{
			name:     "device type 6 (Connected Device)",
			wrapper:  createWrapper("US", 6, "Connected Device UA"),
			expected: "m",
		},
		{
			name:     "device type 7 (Set Top Box)",
			wrapper:  createWrapper("US", 7, "STB UA"),
			expected: "t",
		},
		{
			name:        "device type 0 (unknown)",
			wrapper:     createWrapper("US", 0, "UA"),
			expectError: true,
		},
		{
			name:        "missing device",
			wrapper:     &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			category, err := extractDeviceCategory(tt.wrapper)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, category)
			}
		})
	}
}

func TestExtractBrowser(t *testing.T) {
	tests := []struct {
		name        string
		ua          string
		expected    string
		expectError bool
	}{
		{
			name:     "Chrome desktop",
			ua:       "Mozilla/5.0 (Windows NT 10.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "chrome",
		},
		{
			name:     "Chrome iOS (CriOS)",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.6099.119 Mobile/15E148 Safari/604.1",
			expected: "chrome",
		},
		{
			name:     "Safari desktop",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Safari/605.1.15",
			expected: "safari",
		},
		{
			name:     "Safari mobile",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
			expected: "safari",
		},
		{
			name:     "Firefox desktop",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/120.0",
			expected: "ff",
		},
		{
			name:     "Firefox iOS (FxiOS)",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/120.0 Mobile/15E148 Safari/605.1.15",
			expected: "ff",
		},
		{
			name:     "Edge Chromium",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			expected: "edge",
		},
		{
			name:     "Edge Legacy",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/70.0.3538.102 Safari/537.36 Edge/18.19042",
			expected: "edge",
		},
		{
			name:     "Opera desktop",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0",
			expected: "opera",
		},
		{
			name:     "Opera mobile",
			ua:       "Mozilla/5.0 (Linux; Android 10) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 OPR/76.2.4027.73374",
			expected: "opera",
		},
		{
			name:     "Unknown browser defaults to chrome",
			ua:       "Mozilla/5.0 SomeBrowser/1.0",
			expected: "chrome",
		},
		{
			name:        "empty UA",
			ua:          "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapper := createWrapper("US", 2, tt.ua)
			browser, err := extractBrowser(wrapper)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, browser)
			}
		})
	}
}

func TestIsTablet(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected bool
	}{
		{
			name:     "iPad",
			ua:       "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X)",
			expected: true,
		},
		{
			name:     "Android Tablet",
			ua:       "Mozilla/5.0 (Linux; Android 11) Tablet",
			expected: true,
		},
		{
			name:     "Kindle",
			ua:       "Mozilla/5.0 (Linux; Android) Kindle",
			expected: true,
		},
		{
			name:     "iPhone (not tablet)",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0)",
			expected: false,
		},
		{
			name:     "Desktop (not tablet)",
			ua:       "Mozilla/5.0 (Windows NT 10.0)",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTablet(tt.ua)
			assert.Equal(t, tt.expected, result)
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
				return w
			}(),
			expectedURL: "http://localhost:8080/ts-server/US/w/chrome/ts.json",
		},
		{
			name: "mobile fallback",
			wrapper: func() *openrtb_ext.RequestWrapper {
				w := createWrapper("US", 0, "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Safari/604.1")
				w.Device.DeviceType = 0
				w.Device.Geo = nil
				return w
			}(),
			expectedURL: "http://localhost:8080/ts-server/US/m/safari/ts.json",
		},
		{
			name: "tablet fallback",
			wrapper: func() *openrtb_ext.RequestWrapper {
				w := createWrapper("US", 0, "Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.0 Mobile/15E148 Safari/604.1")
				w.Device.DeviceType = 0
				w.Device.Geo = nil
				return w
			}(),
			expectedURL: "http://localhost:8080/ts-server/US/t/safari/ts.json",
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

// Helper functions for creating test wrappers

func createWrapper(country string, deviceType int64, ua string) *openrtb_ext.RequestWrapper {
	return &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
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
			Device: &openrtb2.Device{
				DeviceType: adcom1.DeviceType(deviceType),
				UA:         ua,
			},
		},
	}
}
