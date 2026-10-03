package provider

import (
	"strconv"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/aerospike"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/redis"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache/provider/valkey"
)

func TestNewRedis(t *testing.T) {
	host, port := miniredisHostPort(t)

	store, err := New(CacheTypeRedis, Configs{Redis: &redis.Config{Host: host, Port: port}})
	require.NoError(t, err)
	require.NotNil(t, store)
	t.Cleanup(func() { _ = store.Close() })

	assert.Equal(t, "redis", store.Name())
}

func TestNewValkey(t *testing.T) {
	host, port := miniredisHostPort(t)

	store, err := New(CacheTypeValkey, Configs{Valkey: &valkey.Config{Host: host, Port: port}})
	require.NoError(t, err)
	require.NotNil(t, store)
	t.Cleanup(func() { _ = store.Close() })

	assert.Equal(t, "valkey", store.Name())
}

func TestNewRedisUnreachableReturnsError(t *testing.T) {
	store, err := New(CacheTypeRedis, Configs{Redis: &redis.Config{Host: "127.0.0.1", Port: 1}})
	require.Error(t, err)
	// Compared as an interface, not via assert.Nil: reflection reports a typed nil pointer as nil,
	// so assert.Nil would pass on the very leak this guards.
	assert.True(t, store == nil, "a failed ping must not leak a nil pointer inside a non-nil interface")
}

func TestNewValkeyUnreachableReturnsError(t *testing.T) {
	store, err := New(CacheTypeValkey, Configs{Valkey: &valkey.Config{Host: "127.0.0.1", Port: 1}})
	require.Error(t, err)
	assert.True(t, store == nil, "a failed ping must not leak a nil pointer inside a non-nil interface")
}

func TestNewAerospikeUnreachableReturnsError(t *testing.T) {
	store, err := New(CacheTypeAerospike, Configs{Aerospike: &aerospike.Config{
		Host: "127.0.0.1", Port: 1, Namespace: "ns", Set: "identity",
	}})
	require.Error(t, err)
	assert.True(t, store == nil, "a failed dial must not leak a nil pointer inside a non-nil interface")
}

func TestNewRejectsUnusableCacheType(t *testing.T) {
	tests := []struct {
		name      string
		cacheType CacheType
		cfgs      Configs
		want      string
	}{
		{
			// The type is never inferred from whichever block happens to be filled in.
			name: "unset",
			cfgs: Configs{Redis: &redis.Config{Host: "localhost", Port: 6379}},
			want: `unknown cache type ""`,
		},
		{
			name:      "unknown name",
			cacheType: "memcached",
			cfgs:      Configs{Redis: &redis.Config{Host: "localhost", Port: 6379}},
			want:      `unknown cache type "memcached"`,
		},
		{
			name:      "not normalized",
			cacheType: "Redis",
			cfgs:      Configs{Redis: &redis.Config{Host: "localhost", Port: 6379}},
			want:      `unknown cache type "Redis"`,
		},
		{
			name:      "selected block absent",
			cacheType: CacheTypeAerospike,
			cfgs:      Configs{Redis: &redis.Config{Host: "localhost", Port: 6379}},
			want:      "aerospike: no aerospike config",
		},
		{
			name:      "selected valkey block absent",
			cacheType: CacheTypeValkey,
			cfgs:      Configs{Redis: &redis.Config{Host: "localhost", Port: 6379}},
			want:      "valkey: no valkey config",
		},
		{
			name:      "selected block hostless",
			cacheType: CacheTypeRedis,
			cfgs:      Configs{Redis: &redis.Config{Port: 6379}},
			want:      "host: cannot be blank",
		},
		{
			name:      "selected valkey block hostless",
			cacheType: CacheTypeValkey,
			cfgs:      Configs{Valkey: &valkey.Config{Port: 6379}},
			want:      "host: cannot be blank",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			store, err := New(tc.cacheType, tc.cfgs)
			require.ErrorContains(t, err, tc.want)
			assert.Nil(t, store)
		})
	}
}

func TestCacheTypePicksNamedBackend(t *testing.T) {
	host, port := miniredisHostPort(t)

	all := func(ct CacheType) (cache.Store, error) {
		return New(ct, Configs{
			Redis:     &redis.Config{Host: host, Port: port},
			Valkey:    &valkey.Config{Host: host, Port: port},
			Aerospike: &aerospike.Config{Host: "127.0.0.1", Port: 1, Namespace: "ns", Set: "iiq"},
		})
	}

	for _, ct := range []CacheType{CacheTypeRedis, CacheTypeValkey} {
		store, err := all(ct)
		require.NoError(t, err, "%s is reachable and was the one selected", ct)
		t.Cleanup(func() { _ = store.Close() })
		assert.Equal(t, string(ct), store.Name())
	}

	// The aerospike block points nowhere, so selecting it must surface a dial failure rather than
	// quietly falling back to a reachable RESP backend.
	_, err := all(CacheTypeAerospike)
	require.Error(t, err)
}

func TestNewValidatesOnlyTheSelectedBackend(t *testing.T) {
	host, port := miniredisHostPort(t)

	store, err := New(CacheTypeRedis, Configs{
		Redis: &redis.Config{Host: host, Port: port},
		// Invalid, but never selected.
		Valkey:    &valkey.Config{Port: 6379},
		Aerospike: &aerospike.Config{Host: "as1", Port: 3000},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Close() })
	assert.Equal(t, "redis", store.Name())
}

func TestNewRejectsInvalidSelectedBackend(t *testing.T) {
	store, err := New(CacheTypeAerospike, Configs{
		Aerospike: &aerospike.Config{Host: "as1", Port: 3000}, // no namespace/set
	})

	require.ErrorContains(t, err, "aerospike:")
	assert.ErrorContains(t, err, "namespace")
	assert.Nil(t, store)
}

// miniredisHostPort starts a RESP server and returns its host and port. Valkey is wire-compatible,
// so both RESP backends can be pointed at it.
func miniredisHostPort(t *testing.T) (string, int) {
	t.Helper()
	mr := miniredis.RunT(t)
	port, err := strconv.Atoi(mr.Port())
	require.NoError(t, err)
	return mr.Host(), port
}
