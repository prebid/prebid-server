package fetcher

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCacheSelectsConfiguredImplementation(t *testing.T) {
	tests := []struct {
		name       string
		cfg        CacheConfig
		wantStored bool
		wantErr    string
	}{
		{
			name: "none",
			cfg:  CacheConfig{Type: "none"},
		},
		{
			name:       "unbounded",
			cfg:        CacheConfig{Type: "unbounded", TTL: time.Minute},
			wantStored: true,
		},
		{
			name:       "default lru",
			cfg:        CacheConfig{MaxEntries: 1, TTL: time.Minute},
			wantStored: true,
		},
		{
			name:       "explicit lru",
			cfg:        CacheConfig{Type: "lru", MaxEntries: 1, TTL: time.Minute},
			wantStored: true,
		},
		{
			name:    "unsupported",
			cfg:     CacheConfig{Type: "bogus"},
			wantErr: `cache.type "bogus" is not supported`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cache, err := buildCache[string, string](test.cfg, newFakeTime())
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)

			cache.Save("key", "value")
			value, found, stale := cache.Get("key")
			assert.Equal(t, test.wantStored, found)
			if test.wantStored {
				assert.Equal(t, "value", value)
				assert.False(t, stale)
			}
		})
	}
}

func TestEffectiveCacheConfigDisablesTTLWhenRefreshIsNone(t *testing.T) {
	cfg := Config{
		Cache:   CacheConfig{Type: "lru", MaxEntries: 10, TTL: time.Minute},
		Refresh: RefreshConfig{Mode: "none"},
	}

	effective := effectiveCacheConfig(cfg)
	assert.Zero(t, effective.TTL)

	cfg.Refresh.Mode = "ttl"
	effective = effectiveCacheConfig(cfg)
	assert.Equal(t, time.Minute, effective.TTL)
}

func TestBuildNegativeStoreSelectsConfiguredImplementation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     NegativeConfig
		wantNil bool
		wantErr string
	}{
		{
			name:    "disabled",
			cfg:     NegativeConfig{},
			wantNil: true,
		},
		{
			name: "default lru",
			cfg:  NegativeConfig{Enabled: true, MaxEntries: 1, TTL: time.Minute},
		},
		{
			name: "explicit lru",
			cfg:  NegativeConfig{Enabled: true, Type: "lru", MaxEntries: 1, TTL: time.Minute},
		},
		{
			name: "unbounded",
			cfg:  NegativeConfig{Enabled: true, Type: "unbounded", TTL: time.Minute},
		},
		{
			name:    "unsupported",
			cfg:     NegativeConfig{Enabled: true, Type: "bogus"},
			wantErr: `negative.type "bogus" is not supported`,
		},
		{
			name:    "invalid lru",
			cfg:     NegativeConfig{Enabled: true, Type: "lru", TTL: time.Minute},
			wantErr: "negative:",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store, err := buildNegativeStore[string](test.cfg, newFakeTime())
			if test.wantErr != "" {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.NoError(t, err)
			if test.wantNil {
				assert.Nil(t, store)
				return
			}

			verdict := errors.New("verdict")
			store.mark("key", verdict)
			got, found := store.isCached("key")
			require.True(t, found)
			assert.ErrorIs(t, got, verdict)
		})
	}
}

func TestNewNegativeStoreRequiresCache(t *testing.T) {
	store, err := NewNegativeStore[string](nil)

	require.Nil(t, store)
	require.EqualError(t, err, "negative cache is required")
}

func TestApplyRefreshConfigSelectsBackgroundAndPreloadBehavior(t *testing.T) {
	plainSource := &stubSource{data: map[string]json.RawMessage{}}
	preloadSource := bulkStub{data: map[string]json.RawMessage{}}
	tests := []struct {
		name            string
		cfg             RefreshConfig
		source          Source[string]
		wantBackground  bool
		wantPreload     bool
		wantErrContains string
	}{
		{
			name:   "default",
			source: plainSource,
		},
		{
			name:           "serve stale",
			cfg:            RefreshConfig{ServeStale: true},
			source:         plainSource,
			wantBackground: true,
		},
		{
			name:           "ttl",
			cfg:            RefreshConfig{Mode: "ttl"},
			source:         plainSource,
			wantBackground: true,
		},
		{
			name:   "none",
			cfg:    RefreshConfig{Mode: "none"},
			source: plainSource,
		},
		{
			name:           "preload",
			cfg:            RefreshConfig{Mode: "preload"},
			source:         preloadSource,
			wantBackground: true,
			wantPreload:    true,
		},
		{
			name:            "preload requires bulk source",
			cfg:             RefreshConfig{Mode: "preload"},
			source:          plainSource,
			wantErrContains: "requires a source that supports bulk loading",
		},
		{
			name:            "unsupported",
			cfg:             RefreshConfig{Mode: "bogus"},
			source:          plainSource,
			wantErrContains: `cache.refresh "bogus" is not supported`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			background, preload, err := applyRefreshConfig(test.cfg, test.source)
			if test.wantErrContains != "" {
				require.ErrorContains(t, err, test.wantErrContains)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantBackground, background)
			assert.Equal(t, test.wantPreload, preload != nil)
		})
	}
}

func TestNewPropagatesConfigurationErrors(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr string
	}{
		{
			name:    "positive cache",
			cfg:     Config{Cache: CacheConfig{Type: "bogus"}},
			wantErr: `cache.type "bogus" is not supported`,
		},
		{
			name: "negative cache",
			cfg: Config{
				Cache:    CacheConfig{Type: "none"},
				Negative: NegativeConfig{Enabled: true, Type: "bogus"},
			},
			wantErr: `negative.type "bogus" is not supported`,
		},
		{
			name: "refresh",
			cfg: Config{
				Cache:   CacheConfig{Type: "none"},
				Refresh: RefreshConfig{Mode: "bogus"},
			},
			wantErr: `cache.refresh "bogus" is not supported`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			f, err := New(Params[string, string]{
				Source:    &stubSource{data: map[string]json.RawMessage{}},
				Transform: identityTransform,
				Config:    test.cfg,
				Time:      newFakeTime(),
				Metrics:   NilRecorder{},
			})

			require.Nil(t, f)
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}
