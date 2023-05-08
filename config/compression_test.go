package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReqCompressionCfgIsSupported(t *testing.T) {
	testCases := []struct {
		description     string
		cfg             CompressionInfo
		compressionKind CompressionKind
		wantSupported   bool
	}{
		{
			description: "Compression type not supported",
			cfg: CompressionInfo{
				GZIP: true,
			},
			compressionKind: CompressionKind("invalid"),
			wantSupported:   false,
		},
		{
			description: "Compression type supported",
			cfg: CompressionInfo{
				GZIP: true,
			},
			compressionKind: CompressionGZIP,
			wantSupported:   true,
		},
		{
			description: "Compression not enabled",
			cfg: CompressionInfo{
				GZIP: false,
			},
			compressionKind: CompressionGZIP,
			wantSupported:   false,
		},
	}

	for _, test := range testCases {
		got := test.cfg.IsSupported(test.compressionKind)
		assert.Equal(t, got, test.wantSupported, test.description)
	}
}
