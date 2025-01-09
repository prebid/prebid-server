package config

import "github.com/prebid/prebid-server/v3/util/httputil"

type Compression struct {
	Request  CompressionInfo `mapstructure:"request"`
	Response CompressionInfo `mapstructure:"response"`
}

// CompressionInfo defines what types of compression algorithms are supported.
type CompressionInfo struct {
	GZIP bool `mapstructure:"enable_gzip"`
}

func (cfg *CompressionInfo) IsSupported(contentEncoding httputil.ContentEncoding) bool {
	switch contentEncoding.Normalize() {
	case httputil.ContentEncodingGZIP:
		return cfg.GZIP
	}
	return false
}
