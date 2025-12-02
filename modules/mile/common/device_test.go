package common

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestExtractDeviceCategory(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		expected    string
		expectError bool
	}{
		{
			name:     "device type 1 (mobile/tablet) with phone UA",
			wrapper:  createTestWrapper("US", 1, "Mozilla/5.0 (Linux; Android) Mobile"),
			expected: "m",
		},
		{
			name:     "device type 1 with iPad UA",
			wrapper:  createTestWrapper("US", 1, "Mozilla/5.0 (iPad; CPU OS 14_0) Safari/604.1"),
			expected: "t",
		},
		{
			name:     "device type 2 (PC)",
			wrapper:  createTestWrapper("US", 2, "Mozilla/5.0 (Windows NT 10.0) Chrome"),
			expected: "w",
		},
		{
			name:     "device type 3 (Connected TV)",
			wrapper:  createTestWrapper("US", 3, "Smart TV UA"),
			expected: "t",
		},
		{
			name:     "device type 4 (Phone)",
			wrapper:  createTestWrapper("US", 4, "iPhone UA"),
			expected: "m",
		},
		{
			name:     "device type 5 (Tablet)",
			wrapper:  createTestWrapper("US", 5, "Tablet UA"),
			expected: "t",
		},
		{
			name:     "device type 6 (Connected Device)",
			wrapper:  createTestWrapper("US", 6, "Connected Device UA"),
			expected: "m",
		},
		{
			name:     "device type 7 (Set Top Box)",
			wrapper:  createTestWrapper("US", 7, "STB UA"),
			expected: "t",
		},
		{
			name:        "device type 0 (unknown)",
			wrapper:     createTestWrapper("US", 0, "UA"),
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
			category, err := ExtractDeviceCategory(tt.wrapper)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, category)
			}
		})
	}
}

func TestDeriveDeviceCategory(t *testing.T) {
	tests := []struct {
		name     string
		wrapper  *openrtb_ext.RequestWrapper
		expected string
	}{
		{
			name: "sua_mobile_1",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0",
						SUA: &openrtb2.UserAgent{
							Mobile: ptrutil.ToPtr[int8](1),
						},
					},
				},
			},
			expected: "m",
		},
		{
			name: "sua_mobile_0",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0",
						SUA: &openrtb2.UserAgent{
							Mobile: ptrutil.ToPtr[int8](0),
						},
					},
				},
			},
			expected: "w",
		},
		{
			name: "sua_browsers_tablet",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0",
						SUA: &openrtb2.UserAgent{
							Browsers: []openrtb2.BrandVersion{
								{Brand: "iPad"},
							},
						},
					},
				},
			},
			expected: "t",
		},
		{
			name: "sua_browsers_surface",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0",
						SUA: &openrtb2.UserAgent{
							Browsers: []openrtb2.BrandVersion{
								{Brand: "Surface"},
							},
						},
					},
				},
			},
			expected: "t",
		},
		{
			name: "ua_fallback_ipad",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0 (iPad; CPU OS 14_0)",
					},
				},
			},
			expected: "t",
		},
		{
			name: "ua_fallback_tablet",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0 (Linux; Android 11) Tablet",
					},
				},
			},
			expected: "t",
		},
		{
			name: "ua_fallback_kindle",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0 (Linux; Android) Kindle",
					},
				},
			},
			expected: "t",
		},
		{
			name: "ua_fallback_mobile",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0)",
					},
				},
			},
			expected: "m",
		},
		{
			name: "ua_fallback_android",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0 (Linux; Android 10) Mobile",
					},
				},
			},
			expected: "m",
		},
		{
			name: "ua_fallback_smart_tv",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0 Smart-TV",
					},
				},
			},
			expected: "t",
		},
		{
			name: "ua_fallback_appletv",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "AppleTV",
					},
				},
			},
			expected: "t",
		},
		{
			name: "ua_fallback_default_desktop",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "Mozilla/5.0 (Windows NT 10.0)",
					},
				},
			},
			expected: "w",
		},
		{
			name: "empty_ua",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						UA: "",
					},
				},
			},
			expected: "",
		},
		{
			name: "nil_device",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: nil,
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DeriveDeviceCategory(tt.wrapper)
			assert.Equal(t, tt.expected, result)
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

// Helper function for creating test wrappers
func createTestWrapper(country string, deviceType int64, ua string) *openrtb_ext.RequestWrapper {
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

