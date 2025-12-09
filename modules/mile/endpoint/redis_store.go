package endpoint

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	redis "github.com/redis/go-redis/v9"
)

// RedisSiteStore fetches site configuration from Redis.
type RedisSiteStore struct {
	client      *redis.Client
	timeout     time.Duration
	keyTemplate string
}

// NewRedisSiteStore builds a Redis-backed SiteStore.
func NewRedisSiteStore(cfg *Config) (*RedisSiteStore, error) {
	if cfg.Redis.Addr == "" {
		return nil, fmt.Errorf("mile.redis.addr is required when mile endpoint is enabled")
	}

	opts := &redis.Options{
		Addr:         cfg.Redis.Addr,
		DB:           cfg.Redis.DB,
		Username:     cfg.Redis.Username,
		Password:     cfg.Redis.Password,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	}

	if cfg.Redis.TLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	timeout := time.Duration(cfg.RedisTimeoutMs) * time.Millisecond
	if timeout == 0 {
		timeout = 200 * time.Millisecond // default
	}

	return &RedisSiteStore{
		client:      redis.NewClient(opts),
		timeout:     timeout,
		keyTemplate: "mile:site:%s",
	}, nil
}

// Get retrieves the site configuration JSON and unmarshals it.
func (s *RedisSiteStore) Get(ctx context.Context, siteID string) (*SiteConfig, error) {
	if siteID == "" {
		return nil, fmt.Errorf("site id is required")
	}

	readCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	key := fmt.Sprintf(s.keyTemplate, siteID)
	val, err := s.client.Get(readCtx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrSiteNotFound
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var site SiteConfig
	if err := json.Unmarshal([]byte(val), &site); err != nil {
		return nil, fmt.Errorf("failed to unmarshal site config: %w", err)
	}
	return &site, nil
}

// Close releases Redis resources.
func (s *RedisSiteStore) Close() error {
	return s.client.Close()
}

// Ping checks if Redis is reachable.
func (s *RedisSiteStore) Ping(ctx context.Context) error {
	pingCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.client.Ping(pingCtx).Err()
}
