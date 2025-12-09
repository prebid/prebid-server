package common

import (
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

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
			wrapper := &openrtb_ext.RequestWrapper{
				BidRequest: &openrtb2.BidRequest{
					Device: &openrtb2.Device{
						DeviceType: adcom1.DeviceType(2),
						UA:         tt.ua,
					},
				},
			}
			browser, err := ExtractBrowser(wrapper)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, browser)
			}
		})
	}
}
