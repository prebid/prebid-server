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

type CompressionKind string

const (
	CompressionGZIP CompressionKind = "gzip"
)

func (k CompressionKind) ToLower() CompressionKind {
	return CompressionKind(strings.ToLower(string(k)))
}

func (cfg *CompressionInfo) IsSupported(kind CompressionKind) bool {
	kind = kind.ToLower()
	switch kind {
	case CompressionGZIP:
		return cfg.GZIP
	}
	return false
}
