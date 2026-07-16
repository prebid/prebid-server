package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStore(t *testing.T) (*RedisStore, *miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewRedisStore(client), mr, client
}

func TestRedisStoreGetAbsentReturnsEmptyNoError(t *testing.T) {
	store, _, _ := newTestStore(t)
	value, err := store.Get(context.Background(), "missing")
	assert.NoError(t, err)
	assert.Equal(t, "", value)
}

func TestRedisStorePutGetRoundTrip(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.Put(ctx, "k", "v", time.Minute))

	value, err := store.Get(ctx, "k")
	assert.NoError(t, err)
	assert.Equal(t, "v", value)
}

func TestRedisStorePutSetsTTL(t *testing.T) {
	store, mr, _ := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.Put(ctx, "k", "v", 90*time.Second))
	assert.Equal(t, 90*time.Second, mr.TTL("k"))
}

func TestRedisStoreGetError(t *testing.T) {
	store, mr, _ := newTestStore(t)
	mr.Close()
	_, err := store.Get(context.Background(), "k")
	assert.Error(t, err)
}

func TestRedisStoreDBSize(t *testing.T) {
	store, _, _ := newTestStore(t)
	ctx := context.Background()
	require.NoError(t, store.Put(ctx, "a", "1", time.Minute))
	require.NoError(t, store.Put(ctx, "b", "2", time.Minute))

	n, err := store.DBSize(ctx)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), n)
}

func TestRedisStoreEvictedKeys(t *testing.T) {
	store, mr, _ := newTestStore(t)
	// miniredis INFO does not report evicted_keys, so a real poll yields 0 (missing field).
	n, err := store.EvictedKeys(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, int64(0), n)
	_ = mr
}

func TestParseEvictedKeys(t *testing.T) {
	tests := []struct {
		name string
		info string
		want int64
	}{
		{
			name: "present crlf",
			info: "# Stats\r\nkeyspace_hits:10\r\nevicted_keys:42\r\nkeyspace_misses:3\r\n",
			want: 42,
		},
		{
			name: "present lf",
			info: "evicted_keys:7\n",
			want: 7,
		},
		{
			name: "present with whitespace",
			info: "evicted_keys:  99  \n",
			want: 99,
		},
		{
			name: "missing field",
			info: "keyspace_hits:10\nkeyspace_misses:3\n",
			want: 0,
		},
		{
			name: "unparsable value",
			info: "evicted_keys:notanumber\n",
			want: 0,
		},
		{
			name: "empty",
			info: "",
			want: 0,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseEvictedKeys(tc.info))
		})
	}
}
