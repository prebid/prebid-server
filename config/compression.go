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

type CompressionType string

const (
	CompressionGZIP CompressionType = "gzip"
)

func (k CompressionType) ToLower() CompressionType {
	return CompressionType(strings.ToLower(string(k)))
}

func (cfg *CompressionInfo) IsSupported(compressionType CompressionType) bool {
	compressionType = compressionType.ToLower()
	switch compressionType {
	case CompressionGZIP:
		return cfg.GZIP
	}
	return false
}
