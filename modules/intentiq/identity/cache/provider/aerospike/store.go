package aerospike

import (
	"context"
	"errors"
	"time"

	as "github.com/aerospike/aerospike-client-go/v7"
)

const valueBin = "v"

const (
	defaultConnectionQueueSize   = 1024
	defaultMinConnectionsPerNode = 3
)

type Client interface {
	Get(policy *as.BasePolicy, key *as.Key, binNames ...string) (*as.Record, as.Error)
	Put(policy *as.WritePolicy, key *as.Key, bins as.BinMap) as.Error
	Close()
}

type Store struct {
	client    Client
	namespace string
	set       string
}

func New(cfg Config) (*Store, error) {
	client, err := as.NewClientWithPolicy(clientPolicy(cfg.Policy), cfg.Host, cfg.Port)
	if err != nil {
		return nil, err
	}
	client.DisableMetrics()

	return NewWithClient(client, cfg.Namespace, cfg.Set), nil
}

func NewWithClient(client Client, namespace, set string) *Store {
	return &Store{client: client, namespace: namespace, set: set}
}

func clientPolicy(p ClientPolicy) *as.ClientPolicy {
	policy := as.NewClientPolicy()

	policy.ConnectionQueueSize = orDefault(p.ConnectionQueueSize, defaultConnectionQueueSize)
	policy.MinConnectionsPerNode = orDefault(p.MinConnectionsPerNode, defaultMinConnectionsPerNode)
	policy.LimitConnectionsToQueueSize = true

	if p.ConnectTimeoutMs > 0 {
		policy.Timeout = time.Duration(p.ConnectTimeoutMs) * time.Millisecond
	}
	if p.IdleTimeoutMs > 0 {
		policy.IdleTimeout = time.Duration(p.IdleTimeoutMs) * time.Millisecond
	}
	return policy
}

func (s *Store) Name() string { return "aerospike" }

func (s *Store) Close() error {
	s.client.Close()
	return nil
}

func (s *Store) Get(_ context.Context, key string) (string, error) {
	k, err := as.NewKey(s.namespace, s.set, key)
	if err != nil {
		return "", err
	}
	record, aerr := s.client.Get(nil, k, valueBin)
	if aerr != nil {
		if errors.Is(aerr, as.ErrKeyNotFound) {
			return "", nil
		}
		return "", aerr
	}
	if record == nil {
		return "", nil
	}
	value, ok := record.Bins[valueBin].(string)
	if !ok {
		return "", nil
	}
	return value, nil
}

func (s *Store) Put(_ context.Context, key, value string, ttl time.Duration) error {
	k, err := as.NewKey(s.namespace, s.set, key)
	if err != nil {
		return err
	}

	policy := as.NewWritePolicy(0, expirationSeconds(ttl))
	policy.RecordExistsAction = as.REPLACE

	if aerr := s.client.Put(policy, k, as.BinMap{valueBin: value}); aerr != nil {
		return aerr
	}
	return nil
}

func orDefault(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func expirationSeconds(ttl time.Duration) uint32 {
	seconds := int64(ttl / time.Second)
	if ttl%time.Second != 0 {
		seconds++
	}
	if seconds < 1 {
		seconds = 1
	}
	return uint32(seconds)
}
