package scope3

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/stretchr/testify/assert"
)

func TestBuilder(t *testing.T) {
	config := json.RawMessage(`{
		"enabled": true,
		"endpoint": "https://rtdp.scope3.com/amazonaps/rtii",
		"auth_key": "test-key",
		"timeout_ms": 1000
	}`)

	deps := moduledeps.ModuleDeps{}
	module, err := Builder(config, deps)

	assert.NoError(t, err)
	assert.NotNil(t, module)
	assert.IsType(t, &Module{}, module)
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
