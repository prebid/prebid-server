package doohqty

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/moduledeps"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	module, err := Builder(json.RawMessage(`{
		"enabled": true,
		"source": {
			"type": "request_lookup",
			"endpoint": "https://values.example.com/lookup"
		},
		"cache_size_bytes": 1048576
	}`), moduledeps.ModuleDeps{HTTPClient: http.DefaultClient})

	require.NoError(t, err)
	doohQtyModule, ok := module.(*Module)
	require.True(t, ok)
	assert.Equal(t, "https://values.example.com/lookup", doohQtyModule.cfg.Source.Endpoint)
	assert.NotNil(t, doohQtyModule.provider)
	assert.NotNil(t, doohQtyModule.requestCache)
	assert.NotNil(t, doohQtyModule.csvSource)
	require.NoError(t, doohQtyModule.Shutdown())
}

func TestBuilderInvalidConfig(t *testing.T) {
	module, err := Builder(json.RawMessage(`{"source":{"type":"bad"}}`), moduledeps.ModuleDeps{})

	require.Error(t, err)
	assert.Nil(t, module)
}

func TestModuleImplementsProcessedAuctionHook(t *testing.T) {
	module := &Module{}

	assert.Implements(t, (*hookstage.ProcessedAuctionRequest)(nil), module)
}

func TestHandleProcessedAuctionHookNilRequestFails(t *testing.T) {
	module := newTestModule(&fakeValueProvider{}, overwritePolicyMissingOnly)

	_, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{}, hookstage.ProcessedAuctionRequestPayload{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "payload contains a nil bid request")
}

func TestHandleProcessedAuctionHookDisabledByAccount(t *testing.T) {
	provider := &fakeValueProvider{}
	module := newTestModule(provider, overwritePolicyMissingOnly)
	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}),
	}

	result, err := module.HandleProcessedAuctionHook(
		context.Background(),
		hookstage.ModuleInvocationContext{AccountID: testAccountID, AccountConfig: json.RawMessage(`{"enabled":false}`)},
		payload,
	)

	require.NoError(t, err)
	assert.Empty(t, result.ChangeSet.Mutations())
	assert.Empty(t, provider.calls)
}

func TestHandleProcessedAuctionHookMissingEndpointWarns(t *testing.T) {
	provider := &fakeValueProvider{}
	module := newTestModule(provider, overwritePolicyMissingOnly)
	module.cfg.Source.Endpoint = ""
	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}),
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

	require.NoError(t, err)
	assert.Empty(t, result.ChangeSet.Mutations())
	assert.Empty(t, provider.calls)
	require.Len(t, result.Warnings, 1)
	assert.Equal(t, "DOOH qty source endpoint is not configured", result.Warnings[0])
}

func TestHandleProcessedAuctionHookInvalidAccountConfigFails(t *testing.T) {
	module := newTestModule(&fakeValueProvider{}, overwritePolicyMissingOnly)
	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{DOOH: &openrtb2.DOOH{ID: "screen-1"}}},
	}

	_, err := module.HandleProcessedAuctionHook(
		context.Background(),
		hookstage.ModuleInvocationContext{AccountID: testAccountID, AccountConfig: json.RawMessage(`{"lookup_paths":["site.id"]}`)},
		payload,
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `lookup path "site.id" is not supported`)
}
