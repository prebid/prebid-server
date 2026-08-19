package identity

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeUA(t *testing.T) {
	tests := []struct {
		name string
		ua   string
		want string
	}{
		{"blank", "", ""},
		{"whitespace only", "   ", ""},
		{"unrecognized", "SomeRandomBot/1.0", ""},
		{
			name: "iphone safari mobile",
			ua:   "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
			want: "iOS17_MobileSafari17_iPhone",
		},
		{
			name: "android chrome mobile",
			ua:   "Mozilla/5.0 (Linux; Android 14; Pixel 8) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			want: "Android14_MobileChrome120_Pixel8",
		},
		{
			name: "windows desktop chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			want: "Windows10_Chrome120",
		},
		{
			name: "mac safari desktop",
			ua:   "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1 (KHTML, like Gecko) Version/17.0 Safari/605.1",
			want: "MacOSX10_Safari17_Mac",
		},
		{
			name: "edge precedes chrome",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			want: "Windows10_Edge120",
		},
		{
			name: "firefox",
			ua:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
			want: "Windows10_Firefox121",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeUA(tt.ua))
		})
	}
}

// TestNormalizeUADeterministic is the one hard requirement: identical input always yields identical
// output.
func TestNormalizeUADeterministic(t *testing.T) {
	uas := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) Version/17.0 Mobile Safari/604.1",
		"Mozilla/5.0 (Linux; Android 14; Pixel 8) Chrome/120.0.0.0 Mobile Safari/537.36",
		"garbage",
		"",
	}
	for _, ua := range uas {
		first := normalizeUA(ua)
		for i := 0; i < 5; i++ {
			assert.Equal(t, first, normalizeUA(ua), "normalizeUA must be deterministic for %q", ua)
		}
	}
}
