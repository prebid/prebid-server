package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReqCompressionCfgIsSupported(t *testing.T) {
	testCases := []struct {
		description     string
		cfg             CompressionInfo
		contentEncoding ContentEncoding
		wantSupported   bool
	}{
		{
			description: "Compression type not supported",
			cfg: CompressionInfo{
				GZIP: true,
			},
			contentEncoding: ContentEncoding("invalid"),
			wantSupported:   false,
		},
		{
			description: "Compression type supported",
			cfg: CompressionInfo{
				GZIP: true,
			},
			contentEncoding: ContentEncodingGZIP,
			wantSupported:   true,
		},
		{
			description: "Compression not enabled",
			cfg: CompressionInfo{
				GZIP: false,
			},
			contentEncoding: ContentEncodingGZIP,
			wantSupported:   false,
		},
	}

	for _, test := range testCases {
		got := test.cfg.IsSupported(test.contentEncoding)
		assert.Equal(t, got, test.wantSupported, test.description)
	}
}
