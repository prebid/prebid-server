package valkey

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*Store, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	port, err := strconv.Atoi(mr.Port())
	require.NoError(t, err)

	store, err := New(Config{Host: mr.Host(), Port: port})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	return store, mr
}

func TestNewUnreachableReturnsError(t *testing.T) {
	store, err := New(Config{Host: "127.0.0.1", Port: 1})
	assert.Error(t, err)
	assert.Nil(t, store)
}

func TestStoreName(t *testing.T) {
	store, _ := newTestStore(t)
	assert.Equal(t, "valkey", store.Name())
}

func TestStorePutSetsTTL(t *testing.T) {
	store, mr := newTestStore(t)
	require.NoError(t, store.Put(context.Background(), "k", "v", 90*time.Second))
	assert.Equal(t, 90*time.Second, mr.TTL("k"))
}

func TestStorePutWithoutTTL(t *testing.T) {
	store, mr := newTestStore(t)
	require.NoError(t, store.Put(context.Background(), "k", "v", 0))
	assert.Zero(t, mr.TTL("k"), "a non-positive TTL stores the key without an expiry")
}

func TestStoreGetRoundTrip(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.Put(ctx, "k", "v", time.Minute))

	value, err := store.Get(ctx, "k")
	require.NoError(t, err)
	assert.Equal(t, "v", value)
}

func TestStoreGetMissingKey(t *testing.T) {
	store, _ := newTestStore(t)
	value, err := store.Get(context.Background(), "absent")
	assert.NoError(t, err, "an absent key is a miss, not a backend failure")
	assert.Empty(t, value)
}

func TestStoreGetError(t *testing.T) {
	store, mr := newTestStore(t)
	mr.Close()
	_, err := store.Get(context.Background(), "k")
	assert.Error(t, err)
}
