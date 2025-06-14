package scope3

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"endpoint": "https://rtdp.scope3.com/amazonaps/rtii",
		"auth_key": "test-key",
		"timeout_ms": 1000,
		"cache_ttl_seconds": 60,
		"bid_meta_data": false
	}`)

	deps := moduledeps.ModuleDeps{}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.IsType(t, &Module{}, module)

	m := module.(*Module)
	assert.Equal(t, "https://rtdp.scope3.com/amazonaps/rtii", m.cfg.Endpoint)
	assert.Equal(t, "test-key", m.cfg.AuthKey)
	assert.Equal(t, 1000, m.cfg.Timeout)
	assert.Equal(t, 60, m.cfg.CacheTTL)
	assert.Equal(t, false, m.cfg.BidMetaData)
	assert.NotNil(t, m.cache)
}

func TestBuilderInvalidConfig(t *testing.T) {
	config := json.RawMessage(`invalid json`)
	deps := moduledeps.ModuleDeps{}

	module, err := Builder(config, deps)

	assert.Error(t, err)
	assert.Nil(t, module)
}

func TestHandleEntrypointHook(t *testing.T) {
	module := &Module{}
	ctx := context.Background()
	miCtx := hookstage.ModuleInvocationContext{}
	payload := hookstage.EntrypointPayload{}

	result, err := module.HandleEntrypointHook(ctx, miCtx, payload)

	assert.NoError(t, err)
	assert.NotNil(t, result.ModuleContext["segments"])
}

func TestHandleProcessedAuctionHook_NoSegments(t *testing.T) {
	module := &Module{}
	ctx := context.Background()
	miCtx := hookstage.ModuleInvocationContext{
		ModuleContext: hookstage.ModuleContext{
			"segments": &sync.Map{},
		},
	}
	payload := hookstage.ProcessedAuctionRequestPayload{}

	result, err := module.HandleProcessedAuctionHook(ctx, miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.ChangeSet)
}

func TestBuilderDefaults(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"auth_key": "test-key"
	}`)

	deps := moduledeps.ModuleDeps{}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	m := module.(*Module)
	assert.Equal(t, "https://rtdp.scope3.com/amazonaps/rtii", m.cfg.Endpoint)
	assert.Equal(t, 1000, m.cfg.Timeout)
	assert.Equal(t, 60, m.cfg.CacheTTL)
	assert.Equal(t, false, m.cfg.BidMetaData)
}

func TestCacheOperations(t *testing.T) {
	cache := &segmentCache{data: make(map[string]cacheEntry)}

	// Test cache miss
	segments, found := cache.get("test-key", time.Minute)
	assert.False(t, found)
	assert.Nil(t, segments)

	// Test cache set and hit
	testSegments := []string{"segment1", "segment2"}
	cache.set("test-key", testSegments)

	segments, found = cache.get("test-key", time.Minute)
	assert.True(t, found)
	assert.Equal(t, testSegments, segments)

	// Test cache expiry
	segments, found = cache.get("test-key", time.Nanosecond)
	assert.False(t, found)
	assert.Nil(t, segments)
}
