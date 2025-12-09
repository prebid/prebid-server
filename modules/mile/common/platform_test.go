package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyDevicePlatform(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		// Mobile Android
		{
			name:     "Android Chrome mobile",
			ua:       "Mozilla/5.0 (Linux; Android 10; SM-G960F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			expected: "m-android|chrome",
		},
		{
			name:     "Android Samsung Internet",
			ua:       "Mozilla/5.0 (Linux; Android 10; SM-G960F) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/20.0 Chrome/106.0.5249.126 Mobile Safari/537.36",
			expected: "m-android|samsung internet for android",
		},
		{
			name:     "Android Edge",
			ua:       "Mozilla/5.0 (Linux; Android 10; SM-G960F) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36 Edg/120.0.0.0",
			expected: "m-android|edge",
		},
		{
			name:     "Android Firefox",
			ua:       "Mozilla/5.0 (Android 10; Mobile; rv:109.0) Gecko/109.0 Firefox/120.0",
			expected: "m-android|ff",
		},
		// Mobile iOS
		{
			name:     "iPhone Safari",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
			expected: "m-ios|safari",
		},
		{
			name:     "iPhone Chrome (CriOS)",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.6099.119 Mobile/15E148 Safari/604.1",
			expected: "m-ios|chrome",
		},
		{
			name:     "iPhone Firefox (FxiOS)",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) FxiOS/120.0 Mobile/15E148 Safari/605.1.15",
			expected: "m-ios|ff",
		},
		{
			name:     "iPhone Edge (EdgiOS)",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 EdgiOS/120.0.0.0 Mobile/15E148 Safari/604.1",
			expected: "m-ios|edge",
		},
		{
			name:     "iPhone Google Search App",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/280.0.586419231 Mobile/15E148 Safari/604.1",
			expected: "m-ios|google search",
		},
		{
			name:     "iPod Safari",
			ua:       "Mozilla/5.0 (iPod touch; CPU iPhone OS 15_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.0 Mobile/15E148 Safari/604.1",
			expected: "m-ios|safari",
		},
		// Tablet iOS
		{
			name:     "iPad Safari",
			ua:       "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
			expected: "t-ios|safari",
		},
		{
			name:     "iPad Chrome",
			ua:       "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.6099.119 Mobile/15E148 Safari/604.1",
			expected: "t-ios|chrome",
		},
		{
			name:     "iPad Google Search App",
			ua:       "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) GSA/280.0.586419231 Mobile/15E148 Safari/604.1",
			expected: "t-ios|google search",
		},
		// Tablet Android
		{
			name:     "Android tablet Chrome",
			ua:       "Mozilla/5.0 (Linux; Android 12; SM-T870) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "t-android|chrome",
		},
		{
			name:     "Android tablet with tablet keyword",
			ua:       "Mozilla/5.0 (Linux; Android 10; Tablet) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "t-android|chrome",
		},
		{
			name:     "Amazon Fire tablet (Silk)",
			ua:       "Mozilla/5.0 (Linux; Android 11; KFTRWI) AppleWebKit/537.36 (KHTML, like Gecko) Silk/120.0.0.0 like Chrome/120.0.0.0 Safari/537.36",
			expected: "t-android|amazon silk",
		},
		// Desktop/Web
		{
			name:     "Windows Chrome",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "w|chrome",
		},
		{
			name:     "Windows Edge",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			expected: "w|edge",
		},
		{
			name:     "Windows Firefox",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:109.0) Gecko/20100101 Firefox/120.0",
			expected: "w|ff",
		},
		{
			name:     "Mac Safari",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Safari/605.1.15",
			expected: "w|safari",
		},
		{
			name:     "Mac Chrome",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "w|chrome",
		},
		{
			name:     "Linux Chrome",
			ua:       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "w|chrome",
		},
		{
			name:     "Windows Opera",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0",
			expected: "w|opera",
		},
		// Edge cases
		{
			name:     "empty UA returns empty",
			ua:       "",
			expected: "",
		},
		{
			name:     "unknown browser defaults to chrome",
			ua:       "Mozilla/5.0 (Windows NT 10.0) SomeUnknownBrowser/1.0",
			expected: "w|chrome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ClassifyDevicePlatform(tt.ua)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectDeviceType(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			name:     "iPhone",
			ua:       "Mozilla/5.0 (iPhone; CPU iPhone OS 16_0 like Mac OS X)",
			expected: "m-ios",
		},
		{
			name:     "iPod",
			ua:       "Mozilla/5.0 (iPod touch; CPU iPhone OS 15_0 like Mac OS X)",
			expected: "m-ios",
		},
		{
			name:     "iPad",
			ua:       "Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X)",
			expected: "t-ios",
		},
		{
			name:     "Android mobile with Mobile keyword",
			ua:       "Mozilla/5.0 (Linux; Android 10; SM-G960F) AppleWebKit/537.36 Chrome/120.0.0.0 Mobile Safari/537.36",
			expected: "m-android",
		},
		{
			name:     "Android tablet without Mobile keyword",
			ua:       "Mozilla/5.0 (Linux; Android 12; SM-T870) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
			expected: "t-android",
		},
		{
			name:     "Android with tablet keyword",
			ua:       "Mozilla/5.0 (Linux; Android 10; Tablet) AppleWebKit/537.36 Chrome/120.0.0.0 Safari/537.36",
			expected: "t-android",
		},
		{
			name:     "Windows desktop",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expected: "w",
		},
		{
			name:     "Mac desktop",
			ua:       "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15",
			expected: "w",
		},
		{
			name:     "Linux desktop",
			ua:       "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36",
			expected: "w",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectDeviceType(tt.ua)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDetectBrowser(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			name:     "Chrome",
			ua:       "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36",
			expected: "chrome",
		},
		{
			name:     "Chrome iOS (CriOS)",
			ua:       "Mozilla/5.0 CriOS/120.0.6099.119 Mobile/15E148 Safari/604.1",
			expected: "chrome",
		},
		{
			name:     "Safari",
			ua:       "Mozilla/5.0 Version/16.0 Safari/605.1.15",
			expected: "safari",
		},
		{
			name:     "Firefox",
			ua:       "Mozilla/5.0 Firefox/120.0",
			expected: "ff",
		},
		{
			name:     "Firefox iOS (FxiOS)",
			ua:       "Mozilla/5.0 FxiOS/120.0 Mobile/15E148 Safari/605.1.15",
			expected: "ff",
		},
		{
			name:     "Edge Chromium",
			ua:       "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			expected: "edge",
		},
		{
			name:     "Edge Legacy",
			ua:       "Mozilla/5.0 Chrome/70.0.3538.102 Safari/537.36 Edge/18.19042",
			expected: "edge",
		},
		{
			name:     "Edge iOS (EdgiOS)",
			ua:       "Mozilla/5.0 EdgiOS/120.0.0.0 Mobile/15E148 Safari/604.1",
			expected: "edge",
		},
		{
			name:     "Opera",
			ua:       "Mozilla/5.0 Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0",
			expected: "opera",
		},
		{
			name:     "Opera Mobile",
			ua:       "Mozilla/5.0 Opera/12.0",
			expected: "opera",
		},
		{
			name:     "Google Search App",
			ua:       "Mozilla/5.0 GSA/280.0.586419231 Mobile/15E148 Safari/604.1",
			expected: "google search",
		},
		{
			name:     "Samsung Internet",
			ua:       "Mozilla/5.0 SamsungBrowser/20.0 Chrome/106.0.5249.126 Mobile Safari/537.36",
			expected: "samsung internet for android",
		},
		{
			name:     "Amazon Silk",
			ua:       "Mozilla/5.0 Silk/120.0.0.0 like Chrome/120.0.0.0 Safari/537.36",
			expected: "amazon silk",
		},
		{
			name:     "Unknown browser defaults to chrome",
			ua:       "Mozilla/5.0 SomeUnknownBrowser/1.0",
			expected: "chrome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectBrowser(tt.ua)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestBrowserDetectionOrder verifies that browser detection order is correct
// (e.g., Samsung Internet should be detected before Chrome since it contains "Chrome")
func TestBrowserDetectionOrder(t *testing.T) {
	tests := []struct {
		name     string
		ua       string
		expected string
	}{
		{
			name:     "Samsung Internet contains Chrome but should detect as Samsung",
			ua:       "Mozilla/5.0 (Linux; Android 10; SM-G960F) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/20.0 Chrome/106.0.5249.126 Mobile Safari/537.36",
			expected: "samsung internet for android",
		},
		{
			name:     "Amazon Silk contains Chrome but should detect as Silk",
			ua:       "Mozilla/5.0 (Linux; Android 11; KFTRWI) AppleWebKit/537.36 (KHTML, like Gecko) Silk/120.0.0.0 like Chrome/120.0.0.0 Safari/537.36",
			expected: "amazon silk",
		},
		{
			name:     "Edge contains Chrome but should detect as Edge",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			expected: "edge",
		},
		{
			name:     "Opera contains Chrome but should detect as Opera",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 OPR/106.0.0.0",
			expected: "opera",
		},
		{
			name:     "Chrome contains Safari but should detect as Chrome",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			expected: "chrome",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectBrowser(tt.ua)
			assert.Equal(t, tt.expected, result)
		})
	}
}
