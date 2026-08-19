package redis

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
	assert.Equal(t, "redis", store.Name())
}

func TestStorePutSetsTTL(t *testing.T) {
	store, mr := newTestStore(t)
	require.NoError(t, store.Put(context.Background(), "k", "v", 90*time.Second))
	assert.Equal(t, 90*time.Second, mr.TTL("k"))
}

func TestStoreGetError(t *testing.T) {
	store, mr := newTestStore(t)
	mr.Close()
	_, err := store.Get(context.Background(), "k")
	assert.Error(t, err)
}

func TestNewAppliesConfiguredConnectTimeout(t *testing.T) {
	mr := miniredis.RunT(t)
	port, _ := strconv.Atoi(mr.Port())

	store, err := New(Config{Host: mr.Host(), Port: port, ConnectTimeoutMs: 150})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	assert.Equal(t, 150*time.Millisecond, store.client.Options().DialTimeout)

	deflt, err := New(Config{Host: mr.Host(), Port: port})
	require.NoError(t, err)
	t.Cleanup(func() { _ = deflt.Close() })
	assert.Equal(t, 5*time.Second, deflt.client.Options().DialTimeout, "unset -> 5s default")
}
