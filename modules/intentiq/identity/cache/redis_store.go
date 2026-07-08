package cache

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// evictedKeysField is the INFO stats line prefix carrying the cumulative evicted_keys counter.
const evictedKeysField = "evicted_keys:"

// RedisStore is the Redis-backed Store (the default L2 backend). Values are stored with a per-entry
// TTL via SET key value PX <ttl>.
type RedisStore struct {
	client *redis.Client
}

// NewRedisStore builds a RedisStore from a go-redis client.
func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Get returns the value for key, or "" when absent.
func (s *RedisStore) Get(ctx context.Context, key string) (string, error) {
	value, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		// Absent is not an error: fall through to a live API call.
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// Put stores value under key with the given TTL.
func (s *RedisStore) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}

// DBSize is the current key count of the selected DB (DBSIZE). Instance-wide, not module-scoped.
func (s *RedisStore) DBSize(ctx context.Context) (int64, error) {
	return s.client.DBSize(ctx).Result()
}

// EvictedKeys is the cumulative evicted_keys from INFO stats. Instance-wide.
func (s *RedisStore) EvictedKeys(ctx context.Context) (int64, error) {
	info, err := s.client.Info(ctx, "stats").Result()
	if err != nil {
		return 0, err
	}
	return parseEvictedKeys(info), nil
}

// parseEvictedKeys extracts the evicted_keys counter from an INFO stats payload, returning 0 on a
// missing or unparsable line (faithful to Java parseEvictedKeys).
func parseEvictedKeys(info string) int64 {
	for _, line := range strings.Split(info, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.HasPrefix(line, evictedKeysField) {
			raw := strings.TrimSpace(line[len(evictedKeysField):])
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return 0
			}
			return n
		}
	}
	return 0
}
