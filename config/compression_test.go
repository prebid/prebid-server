package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReqCompressionCfgIsSupported(t *testing.T) {
	testCases := []struct {
		description     string
		cfg             CompressionInfo
		CompressionType CompressionType
		wantSupported   bool
	}{
		{
			description: "Compression type not supported",
			cfg: CompressionInfo{
				GZIP: true,
			},
			CompressionType: CompressionType("invalid"),
			wantSupported:   false,
		},
		{
			description: "Compression type supported",
			cfg: CompressionInfo{
				GZIP: true,
			},
			CompressionType: CompressionGZIP,
			wantSupported:   true,
		},
		{
			description: "Compression not enabled",
			cfg: CompressionInfo{
				GZIP: false,
			},
			CompressionType: CompressionGZIP,
			wantSupported:   false,
		},
	}

	for _, test := range testCases {
		got := test.cfg.IsSupported(test.CompressionType)
		assert.Equal(t, got, test.wantSupported, test.description)
	}
}
