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
func (s *RedisSiteStore) Get(ctx context.Context, siteID, placementID string) (*SiteConfig, error) {
	if siteID == "" {
		return nil, fmt.Errorf("site id is required")
	}

	readCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Primary: new composite key mile:site:{siteId}|plcmt:{placementId}
	primaryKey := fmt.Sprintf("mile:site:%s|plcmt:%s", siteID, placementID)
	val, err := s.client.Get(readCtx, primaryKey).Result()
	if err == redis.Nil {
		// Fallback to legacy key mile:site:{siteId}
		legacyKey := fmt.Sprintf(s.keyTemplate, siteID)
		val, err = s.client.Get(readCtx, legacyKey).Result()
		if err != nil {
			if err == redis.Nil {
				return nil, ErrSiteNotFound
			}
			return nil, fmt.Errorf("redis get failed: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var site SiteConfig
	if err := json.Unmarshal([]byte(val), &site); err != nil {
		return nil, fmt.Errorf("failed to unmarshal site config: %w", err)
	}
	return &site, nil
}

// GetMulti retrieves site configurations for multiple placements using Redis pipeline.
func (s *RedisSiteStore) GetMulti(ctx context.Context, siteID string, placementIDs []string) (map[string]*SiteConfig, error) {
	if siteID == "" {
		return nil, fmt.Errorf("site id is required")
	}
	if len(placementIDs) == 0 {
		return nil, fmt.Errorf("placement ids are required")
	}

	readCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	// Build primary keys for all placements
	primaryKeys := make([]string, len(placementIDs))
	for i, placementID := range placementIDs {
		primaryKeys[i] = fmt.Sprintf("mile:site:%s|plcmt:%s", siteID, placementID)
	}

	// Use pipeline to fetch all keys at once
	pipe := s.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(primaryKeys))
	for i, key := range primaryKeys {
		cmds[i] = pipe.Get(readCtx, key)
	}
	_, _ = pipe.Exec(readCtx)

	result := make(map[string]*SiteConfig, len(placementIDs))
	var missingPlacements []int // indices of placements not found with primary key

	for i, cmd := range cmds {
		val, err := cmd.Result()
		if err == redis.Nil {
			missingPlacements = append(missingPlacements, i)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("redis get failed for placement %s: %w", placementIDs[i], err)
		}

		var site SiteConfig
		if err := json.Unmarshal([]byte(val), &site); err != nil {
			return nil, fmt.Errorf("failed to unmarshal site config for placement %s: %w", placementIDs[i], err)
		}
		result[placementIDs[i]] = &site
	}

	// Fallback to legacy key for missing placements
	if len(missingPlacements) > 0 {
		legacyKey := fmt.Sprintf(s.keyTemplate, siteID)
		val, err := s.client.Get(readCtx, legacyKey).Result()
		if err == redis.Nil {
			// No legacy key either - check if we have any results
			if len(result) == 0 {
				return nil, ErrSiteNotFound
			}
			// Return partial results (some placements found, some not)
			return result, nil
		}
		if err != nil {
			return nil, fmt.Errorf("redis get failed: %w", err)
		}

		var site SiteConfig
		if err := json.Unmarshal([]byte(val), &site); err != nil {
			return nil, fmt.Errorf("failed to unmarshal site config: %w", err)
		}

		// Use the same site config for all missing placements
		for _, idx := range missingPlacements {
			result[placementIDs[idx]] = &site
		}
	}

	return result, nil
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
