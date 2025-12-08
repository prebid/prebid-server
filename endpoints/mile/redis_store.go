package mile

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"time"

	"github.com/prebid/prebid-server/v3/config"

	redis "github.com/redis/go-redis/v9"
)

// RedisSiteStore fetches site configuration from Redis.
type RedisSiteStore struct {
	client      *redis.Client
	timeout     time.Duration
	keyTemplate string
}

// NewRedisSiteStore builds a Redis-backed SiteStore.
func NewRedisSiteStore(cfg config.Mile) (*RedisSiteStore, error) {
	if cfg.Redis.Addr == "" {
		return nil, fmt.Errorf("mile.redis.addr is required when mile is enabled")
	}

	opts := &redis.Options{
		Addr:     cfg.Redis.Addr,
		DB:       cfg.Redis.DB,
		Username: cfg.Redis.Username,
		Password: cfg.Redis.Password,
	}

	if cfg.Redis.TLS {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	return &RedisSiteStore{
		client:      redis.NewClient(opts),
		timeout:     time.Duration(cfg.RedisTimeoutMs) * time.Millisecond,
		keyTemplate: "mile:site:%s",
	}, nil
}

// Get retrieves the site configuration JSON and unmarshals it.
func (s *RedisSiteStore) Get(ctx context.Context, siteID string) (*SiteConfig, error) {
	if siteID == "" {
		return nil, fmt.Errorf("site id is required")
	}

	readCtx := ctx
	cancel := func() {}
	if s.timeout > 0 {
		readCtx, cancel = context.WithTimeout(ctx, s.timeout)
	}
	defer cancel()

	key := fmt.Sprintf(s.keyTemplate, siteID)
	val, err := s.client.Get(readCtx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, ErrSiteNotFound
		}
		return nil, err
	}

	var site SiteConfig
	if err := json.Unmarshal([]byte(val), &site); err != nil {
		return nil, err
	}
	return &site, nil
}

// Close releases Redis resources.
func (s *RedisSiteStore) Close() error {
	return s.client.Close()
}
