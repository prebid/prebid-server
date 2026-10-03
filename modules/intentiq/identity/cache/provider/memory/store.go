package memory

import (
	"context"
	"sync"
	"time"
)

type item struct {
	value string
	exp   time.Time
}

type Store struct {
	mu    sync.Mutex
	items map[string]item
	err   error
}

func New() *Store {
	return &Store{items: make(map[string]item)}
}

func (s *Store) Name() string { return "memory" }

func (s *Store) Close() error { return nil }

func (s *Store) SetErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *Store) Advance(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, it := range s.items {
		it.exp = it.exp.Add(-d)
		s.items[key] = it
	}
}

// TTL is the remaining lifetime of key, or 0 when absent or expired.
func (s *Store) TTL(key string) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[key]
	if !ok {
		return 0
	}
	remaining := time.Until(it.exp)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Len is the number of live entries.
func (s *Store) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	now := time.Now()
	for _, it := range s.items {
		if it.exp.After(now) {
			n++
		}
	}
	return n
}

func (s *Store) Get(ctx context.Context, key string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return "", s.err
	}
	it, ok := s.items[key]
	if !ok || !it.exp.After(time.Now()) {
		return "", nil
	}
	return it.value, nil
}

func (s *Store) Put(ctx context.Context, key, value string, ttl time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.items[key] = item{value: value, exp: time.Now().Add(ttl)}
	return nil
}
