package config

import "strings"

type Compression struct {
	Request  CompressionInfo `mapstructure:"request"`
	Response CompressionInfo `mapstructure:"response"`
}

// CompressionInfo defines what types of compressions are supported
type CompressionInfo struct {
	GZIP bool `mapstructure:"enable_gzip"`
}

type ContentEncoding string

const (
	ContentEncodingGZIP ContentEncoding = "gzip"
)

func (k ContentEncoding) ToLower() ContentEncoding {
	return ContentEncoding(strings.ToLower(string(k)))
}

func (cfg *CompressionInfo) IsSupported(contentEncoding ContentEncoding) bool {
	contentEncoding = contentEncoding.ToLower()
	switch contentEncoding {
	case ContentEncodingGZIP:
		return cfg.GZIP
	}
	return false
}
