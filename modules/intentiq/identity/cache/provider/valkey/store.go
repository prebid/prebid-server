package valkey

import (
	"context"
	"net"
	"strconv"
	"time"

	govalkey "github.com/valkey-io/valkey-go"
)

type Store struct {
	client govalkey.Client
}

func New(cfg Config) (*Store, error) {
	client, err := govalkey.NewClient(govalkey.ClientOption{
		InitAddress: []string{net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port))},
		Password:    cfg.Password,
		Dialer:      net.Dialer{Timeout: cfg.ConnectTimeout()},
		// The module already has its own L1 cache.
		DisableCache: true,
		// Let cache misses fall back to the API without retry delays.
		DisableRetry: true,
	})
	if err != nil {
		return nil, err
	}

	return &Store{client: client}, nil
}

func (s *Store) Name() string { return "valkey" }

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	value, err := s.client.Do(ctx, s.client.B().Get().Key(key).Build()).ToString()
	if govalkey.IsValkeyNil(err) {
		// Absent is not an error: fall through to a live API call.
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Store) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	set := s.client.B().Set().Key(key).Value(value)
	if ttl <= 0 {
		// PX rejects a non-positive value; no expiry matches what the Redis backend stores.
		return s.client.Do(ctx, set.Build()).Error()
	}
	return s.client.Do(ctx, set.Px(ttl).Build()).Error()
}

func (s *Store) Close() error {
	s.client.Close()
	return nil
}
