package wurfl_devicedetection

import (
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/stretchr/testify/assert"
)

func TestMakeHeaders(t *testing.T) {
	tests := []struct {
		name       string
		device     openrtb2.Device
		rawHeaders map[string]string
		expected   map[string]string
	}{
		{
			name:       "No SUA and no UA",
			device:     openrtb2.Device{},
			rawHeaders: map[string]string{"Custom-Header": "Value"},
			expected:   map[string]string{"Custom-Header": "Value"},
		},
		{
			name: "Only UA",
			device: openrtb2.Device{
				UA: "Mozilla/5.0",
			},
			rawHeaders: map[string]string{},
			expected:   map[string]string{"User-Agent": "Mozilla/5.0"},
		},
		{
			name: "UA and SUA without Browsers",
			device: openrtb2.Device{
				UA: "Mozilla/5.0",
				SUA: &openrtb2.UserAgent{
					Platform: &openrtb2.BrandVersion{
						Brand:   "Android",
						Version: []string{"12"},
					},
				},
			},
			rawHeaders: map[string]string{},
			expected:   map[string]string{"User-Agent": "Mozilla/5.0"},
		},
		{
			name: "No UA and SUA without Browsers",
			device: openrtb2.Device{
				SUA: &openrtb2.UserAgent{
					Platform: &openrtb2.BrandVersion{
						Brand:   "Android",
						Version: []string{"12"},
					},
				},
			},
			rawHeaders: map[string]string{"User-Agent": "Mozilla/5.0"},
			expected:   map[string]string{"User-Agent": "Mozilla/5.0"},
		},
		{
			name: "SUA with browsers and platform",
			device: openrtb2.Device{
				SUA: &openrtb2.UserAgent{
					Browsers: []openrtb2.BrandVersion{
						{Brand: "Google Chrome", Version: []string{"114", "0", "5735"}},
					},
					Platform: &openrtb2.BrandVersion{
						Brand:   "Android",
						Version: []string{"12"},
					},
				},
			},
			rawHeaders: map[string]string{},
			expected: map[string]string{
				"Sec-CH-UA":                   `"Google Chrome";v="114.0.5735"`,
				"Sec-CH-UA-Full-Version-List": `"Google Chrome";v="114.0.5735"`,
				"Sec-CH-UA-Platform":          `"Android"`,
				"Sec-CH-UA-Platform-Version":  `"12"`,
			},
		},
		{
			name: "SUA with mobile and model",
			device: openrtb2.Device{
				SUA: &openrtb2.UserAgent{
					Browsers: []openrtb2.BrandVersion{
						{Brand: "Google Chrome", Version: []string{"114", "0", "5735"}},
					},
					Mobile: func(i int8) *int8 { return &i }(1),
					Model:  "Pixel 6",
				},
			},
			rawHeaders: map[string]string{},
			expected: map[string]string{
				"Sec-CH-UA":                   `"Google Chrome";v="114.0.5735"`,
				"Sec-CH-UA-Full-Version-List": `"Google Chrome";v="114.0.5735"`,
				"Sec-CH-UA-Mobile":            `?1`,
				"Sec-CH-UA-Model":             `"Pixel 6"`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := makeHeaders(test.device, test.rawHeaders)
			assert.Equal(t, test.expected, result)
		})
	}
}
