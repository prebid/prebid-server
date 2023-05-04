package config

import (
	"fmt"
	"strings"
)

type Compression struct {
	Request  ReqCompression  `mapstructure:"request"`
	Response RespCompression `mapstructure:"response"`
}

// CompressionConfig defines if compression is enabled and what type of compression to use
type ReqCompression struct {
	Enabled bool              `mapstructure:"enabled"`
	Kind    []CompressionKind `mapstructure:"kind,flow"`
	kindMap map[CompressionKind]struct{}
}

type RespCompression struct {
	Enabled bool            `mapstructure:"enabled"`
	Kind    CompressionKind `mapstructure:"kind"`
}

type CompressionKind string

const (
	CompressionGZIP CompressionKind = "gzip"
)

func (k CompressionKind) ToLower() CompressionKind {
	return CompressionKind(strings.ToLower(string(k)))
}

func (k CompressionKind) IsValid() bool {
	k = k.ToLower()
	switch k {
	// Case for valid types. As new compression types are added they should
	// be added here as a comma separated list.
	case CompressionGZIP:
		return true
	default:
		return false
	}
}

func (cfg *Compression) validate(errs []error) []error {
	errs = cfg.Request.Validate(errs)
	errs = cfg.Response.Validate(errs)
	return errs
}

func (cfg *ReqCompression) IsSupported(kind CompressionKind) bool {
	if cfg.Enabled {
		if _, ok := cfg.kindMap[kind.ToLower()]; ok {
			return true
		}
	}

	return false
}

func (cfg *ReqCompression) Validate(errs []error) []error {
	if cfg.Enabled {
		if len(cfg.Kind) == 0 {
			errs = append(errs, fmt.Errorf("compression is enabled but no compression types are specified"))
		}

		// This is to enabled O(1) lookups for supported compression types
		cfg.kindMap = make(map[CompressionKind]struct{}, len(cfg.Kind))
		for _, kind := range cfg.Kind {
			k := kind
			if !k.IsValid() {
				errs = append(errs, fmt.Errorf("compression type %s is not valid", kind))
			} else {
				cfg.kindMap[k] = struct{}{}
			}
		}
	}
	return errs
}

func (cfg RespCompression) Validate(errs []error) []error {
	if cfg.Enabled {
		k := cfg.Kind
		if !k.IsValid() {
			errs = append(errs, fmt.Errorf("compression type %s is not valid", cfg.Kind))
		}
	}
	return errs
}
