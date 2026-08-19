package aerospike

import (
	"context"
	"errors"
	"testing"
	"time"

	as "github.com/aerospike/aerospike-client-go/v7"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A port nothing listens on, for exercising the eager-dial failure path.
const (
	unreachableHost = "127.0.0.1"
	unreachablePort = 1
)

// stubClient is an in-memory Client. Records carry no expiry: TTL handling is asserted through
// lastExpiration, and the cache treats Entry.Exp as the authoritative read-time expiry anyway.
type stubClient struct {
	records        map[string]as.BinMap
	lastExpiration uint32
	lastReadPolicy *as.BasePolicy
	getCalled      bool
	nilRecord      bool
	getErr         as.Error
	putErr         as.Error
	closed         bool
}

func newStubClient() *stubClient {
	return &stubClient{records: map[string]as.BinMap{}}
}

func (c *stubClient) Get(policy *as.BasePolicy, key *as.Key, _ ...string) (*as.Record, as.Error) {
	c.lastReadPolicy, c.getCalled = policy, true
	if c.getErr != nil {
		return nil, c.getErr
	}
	if c.nilRecord {
		return nil, nil
	}
	bins, ok := c.records[key.String()]
	if !ok {
		// Model the real client: a missing record comes back as ErrKeyNotFound, not a nil record.
		return nil, as.ErrKeyNotFound
	}
	return &as.Record{Bins: bins}, nil
}

func (c *stubClient) Put(policy *as.WritePolicy, key *as.Key, bins as.BinMap) as.Error {
	if c.putErr != nil {
		return c.putErr
	}
	c.lastExpiration = policy.Expiration
	c.records[key.String()] = bins
	return nil
}

func (c *stubClient) Close() { c.closed = true }

// asError returns a genuine as.Error. The interface has unexported methods, so it cannot be
// implemented or constructed outside the client package; a failed dial is the cheapest real one.
func asError(t *testing.T) as.Error {
	t.Helper()
	_, err := as.NewClientWithPolicy(fastFailPolicy(), unreachableHost, unreachablePort)
	require.Error(t, err)
	return err
}

func fastFailPolicy() *as.ClientPolicy {
	policy := as.NewClientPolicy()
	policy.Timeout = 50 * time.Millisecond
	return policy
}

func newTestStore(t *testing.T) (*Store, *stubClient) {
	t.Helper()
	stub := newStubClient()
	store := NewWithClient(stub, "ns", "identity")
	t.Cleanup(func() { _ = store.Close() })
	return store, stub
}

func TestStoreName(t *testing.T) {
	store, _ := newTestStore(t)
	assert.Equal(t, "aerospike", store.Name())
}

func TestStorePutGetRoundTrip(t *testing.T) {
	store, stub := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.Put(ctx, "k", "hello", 42*time.Second))
	assert.Equal(t, uint32(42), stub.lastExpiration, "TTL maps to record expiration seconds")

	value, err := store.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "hello", value)
}

func TestStorePutReplacesExistingRecord(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.Put(ctx, "k", "first", time.Minute))
	require.NoError(t, store.Put(ctx, "k", "second", time.Minute))

	value, err := store.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "second", value)
}

func TestStoreGetAbsentIsNotAnError(t *testing.T) {
	store, _ := newTestStore(t)
	value, err := store.Get(context.Background(), "absent")
	require.NoError(t, err)
	assert.Equal(t, "", value, "a missing record must read as a miss, not a backend failure")
}

// A nil record with no error cannot happen with the real client, but the branch exists; keep it live.
func TestStoreGetNilRecordIsTreatedAsAbsent(t *testing.T) {
	store, stub := newTestStore(t)
	stub.nilRecord = true

	value, err := store.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, "", value)
}

// Everything that is not a miss must still surface, or real outages look like cold keys.
func TestStoreGetNonMissErrorPropagates(t *testing.T) {
	store, stub := newTestStore(t)
	stub.getErr = asError(t)

	_, err := store.Get(context.Background(), "k")
	require.Error(t, err)
	assert.False(t, errors.Is(err, as.ErrKeyNotFound))
}

// A bin written by something other than this store must not be surfaced as a value.
func TestStoreGetNonStringBinIsTreatedAsAbsent(t *testing.T) {
	store, stub := newTestStore(t)
	k, err := as.NewKey("ns", "identity", "k")
	require.NoError(t, err)
	stub.records[k.String()] = as.BinMap{valueBin: 123}

	value, gerr := store.Get(context.Background(), "k")
	require.NoError(t, gerr)
	assert.Equal(t, "", value)
}

func TestStoreGetError(t *testing.T) {
	store, stub := newTestStore(t)
	stub.getErr = asError(t)

	_, err := store.Get(context.Background(), "k")
	assert.Error(t, err)
}

func TestStorePutError(t *testing.T) {
	store, stub := newTestStore(t)
	stub.putErr = asError(t)

	assert.Error(t, store.Put(context.Background(), "k", "v", time.Minute))
}

func TestStoreCloseReleasesClient(t *testing.T) {
	stub := newStubClient()
	store := NewWithClient(stub, "ns", "identity")
	require.NoError(t, store.Close())
	assert.True(t, stub.closed)
}

// The Aerospike client dials on construction, so an unreachable cluster fails module startup
// instead of failing open on the first request the way the lazy Redis client does.
func TestNewFailsWhenClusterUnreachable(t *testing.T) {
	store, err := New(Config{Host: unreachableHost, Port: unreachablePort, Namespace: "ns", Set: "identity"})
	assert.Error(t, err)
	assert.Nil(t, store)
}

func TestExpirationSeconds(t *testing.T) {
	tests := []struct {
		name string
		ttl  time.Duration
		want uint32
	}{
		{"zero floors at 1s", 0, 1},
		{"negative floors at 1s", -5 * time.Second, 1},
		{"sub-second rounds up", 500 * time.Millisecond, 1},
		{"fractional rounds up", 1500 * time.Millisecond, 2},
		{"whole seconds pass through", 30 * time.Second, 30},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, expirationSeconds(tc.ttl))
		})
	}
}

func TestClientPolicyDefaults(t *testing.T) {
	p := clientPolicy(ClientPolicy{})

	assert.Equal(t, defaultConnectionQueueSize, p.ConnectionQueueSize, "library default of 100 is undersized")
	assert.Equal(t, defaultMinConnectionsPerNode, p.MinConnectionsPerNode, "keep connections warm")
	assert.True(t, p.LimitConnectionsToQueueSize)
	assert.Equal(t, 30*time.Second, p.Timeout, "unset connect timeout stays at the library default")
	assert.Equal(t, time.Duration(0), p.IdleTimeout, "unset idle timeout stays at the library default")
}

func TestClientPolicyOverrides(t *testing.T) {
	p := clientPolicy(ClientPolicy{
		ConnectionQueueSize:   64,
		MinConnectionsPerNode: 8,
		ConnectTimeoutMs:      250,
		IdleTimeoutMs:         5_000,
	})

	assert.Equal(t, 64, p.ConnectionQueueSize)
	assert.Equal(t, 8, p.MinConnectionsPerNode)
	assert.Equal(t, 250*time.Millisecond, p.Timeout)
	assert.Equal(t, 5*time.Second, p.IdleTimeout)
}

// Reads run on the client defaults, matching iiq-prebid-server; writes cannot, because a zero
// Expiration would inherit the namespace default TTL instead of the cache's per-entry ceiling.
func TestGetUsesClientDefaultPolicy(t *testing.T) {
	store, stub := newTestStore(t)
	_, err := store.Get(context.Background(), "k")
	require.NoError(t, err)

	require.True(t, stub.getCalled)
	assert.Nil(t, stub.lastReadPolicy)
}

func TestPutAlwaysSetsExpiration(t *testing.T) {
	store, stub := newTestStore(t)
	ctx := context.Background()

	require.NoError(t, store.Put(ctx, "a", "v", 42*time.Second))
	assert.Equal(t, uint32(42), stub.lastExpiration)

	require.NoError(t, store.Put(ctx, "b", "v", 7*time.Second))
	assert.Equal(t, uint32(7), stub.lastExpiration, "each write carries its own TTL")
}
