package endpoint

import (
	"encoding/json"
	"fmt"
)

// Config holds the configuration for the Mile endpoint module.
type Config struct {
	Enabled          bool        `json:"enabled"`
	Endpoint         string      `json:"endpoint"`
	RequestTimeoutMs int         `json:"request_timeout_ms"`
	RedisTimeoutMs   int         `json:"redis_timeout_ms"`
	MaxRequestSize   int64       `json:"max_request_size"`
	AuthToken        string      `json:"auth_token"`
	Redis            RedisConfig `json:"redis"`
}

// RedisConfig configures the Redis backend used to fetch site settings.
type RedisConfig struct {
	Addr     string `json:"addr"`
	DB       int    `json:"db"`
	Username string `json:"username"`
	Password string `json:"password"`
	TLS      bool   `json:"tls"`
}

func parseConfig(data json.RawMessage) (*Config, error) {
	if len(data) == 0 {
		return &Config{Enabled: false}, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse mile endpoint config: %w", err)
	}

	// Set defaults
	if cfg.MaxRequestSize == 0 {
		cfg.MaxRequestSize = 512 * 1024 // 512KB default
	}
	if cfg.RequestTimeoutMs == 0 {
		cfg.RequestTimeoutMs = 500 // 500ms default
	}
	if cfg.RedisTimeoutMs == 0 {
		cfg.RedisTimeoutMs = 200 // 200ms default
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = "/mile/v1/request"
	}

	return &cfg, nil
}
