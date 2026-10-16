package redis

import (
	"context"
	"errors"
	"net"
	"strconv"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type Store struct {
	client *goredis.Client
}

func New(cfg Config) (*Store, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:        net.JoinHostPort(cfg.Host, strconv.Itoa(cfg.Port)),
		Password:    cfg.Password,
		DialTimeout: cfg.ConnectTimeout(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout())
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}
	return &Store{client: client}, nil
}

// NewWithClient builds a Store over an existing client. Used by tests.
func NewWithClient(client *goredis.Client) *Store {
	return &Store{client: client}
}

func (s *Store) Name() string { return "redis" }

func (s *Store) Close() error { return s.client.Close() }

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	value, err := s.client.Get(ctx, key).Result()
	if errors.Is(err, goredis.Nil) {
		// Absent is not an error: fall through to a live API call.
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

func (s *Store) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	return s.client.Set(ctx, key, value, ttl).Err()
}
