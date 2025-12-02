package common

import (
	"context"
	"errors"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestExtractCountry(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		expected    string
		expectError bool
	}{
		{
			name:     "valid uppercase country",
			wrapper:  createTestWrapper("US", 2, "UA"),
			expected: "US",
		},
		{
			name:     "valid lowercase country",
			wrapper:  createTestWrapper("in", 2, "UA"),
			expected: "IN",
		},
		{
			name:     "valid mixed case country",
			wrapper:  createTestWrapper("Gb", 2, "UA"),
			expected: "GB",
		},
		{
			name:        "missing geo",
			wrapper:     createTestWrapperNoGeo(2, "UA"),
			expectError: true,
		},
		{
			name:        "empty country",
			wrapper:     createTestWrapper("", 2, "UA"),
			expectError: true,
		},
		{
			name:        "invalid country code (too long)",
			wrapper:     createTestWrapper("USA", 2, "UA"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			country, err := ExtractCountry(tt.wrapper)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, country)
			}
		})
	}
}

func TestDeriveCountry(t *testing.T) {
	tests := []struct {
		name        string
		wrapper     *openrtb_ext.RequestWrapper
		geoResolver GeoResolver
		expectError bool
	}{
		{
			name: "nil_geo_resolver",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "1.1.1.1",
					},
				},
			},
			geoResolver: nil,
			expectError: true,
		},
		{
			name: "missing_device",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: nil,
				},
			},
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: true,
		},
		{
			name: "ipv6_fallback",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP:   "",
						IPv6: "2001:db8::1",
					},
				},
			},
			geoResolver: &mockGeoResolver{country: "CA"},
			expectError: false,
		},
		{
			name: "resolver_error",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "1.1.1.1",
					},
				},
			},
			geoResolver: &mockGeoResolver{err: errors.New("test error")},
			expectError: true,
		},
		{
			name: "empty_country",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "1.1.1.1",
					},
				},
			},
			geoResolver: &mockGeoResolver{country: ""},
			expectError: true,
		},
		{
			name: "success",
			wrapper: &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						IP: "1.1.1.1",
					},
				},
			},
			geoResolver: &mockGeoResolver{country: "US"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			country, err := DeriveCountry(context.Background(), tt.wrapper, tt.geoResolver)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, country)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, country)
			}
		})
	}
}

func TestDefaultResolver_Resolve(t *testing.T) {
	t.Run("success_with_all_fields", func(t *testing.T) {
		resolver := NewDefaultResolver(nil)
		wrapper := createTestWrapper("US", 2, "Mozilla/5.0 Chrome/120.0.0.0")

		info, activities, err := resolver.Resolve(context.Background(), wrapper)

		assert.NoError(t, err)
		assert.Equal(t, "US", info.Country)
		assert.Equal(t, "w", info.Device)
		assert.Equal(t, "chrome", info.Browser)
		assert.Empty(t, activities)
	})

	t.Run("country_fallback", func(t *testing.T) {
		geoResolver := &mockGeoResolver{country: "CA"}
		resolver := NewDefaultResolver(geoResolver)
		wrapper := createTestWrapperNoGeo(2, "Mozilla/5.0 Chrome/120.0.0.0")
		wrapper.Device.IP = "1.1.1.1"

		info, activities, err := resolver.Resolve(context.Background(), wrapper)

		assert.NoError(t, err)
		assert.Equal(t, "CA", info.Country)
		assert.Equal(t, "w", info.Device)
		assert.Equal(t, "chrome", info.Browser)
		assert.Len(t, activities, 1)
		assert.Equal(t, "country_derived", activities[0].Name)
	})

	t.Run("device_fallback", func(t *testing.T) {
		resolver := NewDefaultResolver(nil)
		wrapper := createTestWrapper("US", 0, "Mozilla/5.0 (Windows NT 10.0)")

		info, activities, err := resolver.Resolve(context.Background(), wrapper)

		assert.NoError(t, err)
		assert.Equal(t, "US", info.Country)
		assert.Equal(t, "w", info.Device)
		assert.Equal(t, "chrome", info.Browser)
		assert.Len(t, activities, 1)
		assert.Equal(t, "devicetype_derived", activities[0].Name)
	})

	t.Run("both_fallbacks", func(t *testing.T) {
		geoResolver := &mockGeoResolver{country: "GB"}
		resolver := NewDefaultResolver(geoResolver)
		wrapper := createTestWrapperNoGeo(0, "Mozilla/5.0 (iPhone)")
		wrapper.Device.IP = "1.1.1.1"

		info, activities, err := resolver.Resolve(context.Background(), wrapper)

		assert.NoError(t, err)
		assert.Equal(t, "GB", info.Country)
		assert.Equal(t, "m", info.Device)
		assert.Equal(t, "chrome", info.Browser)
		assert.Len(t, activities, 2)
	})

	t.Run("nil_wrapper", func(t *testing.T) {
		resolver := NewDefaultResolver(nil)
		_, _, err := resolver.Resolve(context.Background(), nil)
		assert.Error(t, err)
	})

	t.Run("missing_browser", func(t *testing.T) {
		resolver := NewDefaultResolver(nil)
		wrapper := createTestWrapper("US", 2, "")

		_, _, err := resolver.Resolve(context.Background(), wrapper)
		assert.Error(t, err)
	})
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

func createTestWrapperNoGeo(deviceType int64, ua string) *openrtb_ext.RequestWrapper {
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

