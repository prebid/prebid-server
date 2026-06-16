package doohqty

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/openrtb/v20/adcom1"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAccountID = "acct"

type fakeValueProvider struct {
	values   map[lookupKey]impressionValue
	warnings []string
	err      error
	calls    [][]lookupKey
}

func (p *fakeValueProvider) Lookup(_ context.Context, _ moduleConfig, _ string, lookups []lookupKey) (map[lookupKey]impressionValue, []string, error) {
	p.calls = append(p.calls, append([]lookupKey(nil), lookups...))
	if p.err != nil {
		return nil, p.warnings, p.err
	}

	values := make(map[lookupKey]impressionValue)
	for _, lookup := range lookups {
		if value, ok := p.values[lookup]; ok {
			values[lookup] = value
		}
	}

	return values, p.warnings, nil
}

func newTestModule(provider valueProvider, policy overwritePolicy) *Module {
	return &Module{
		cfg: moduleConfig{
			Enabled: true,
			Source: sourceConfig{
				Type:     sourceTypeRequestLookup,
				Endpoint: "https://values.example.com/lookup",
			},
			LookupPaths:             []string{lookupPathDOOHID},
			OverwritePolicy:         policy,
			CacheTTLSeconds:         60,
			NegativeCacheTTLSeconds: 60,
		},
		provider:     provider,
		requestCache: newValueCache(1024 * 1024),
	}
}

func newDOOHRequest(dooh *openrtb2.DOOH, imps ...openrtb2.Imp) *openrtb_ext.RequestWrapper {
	return &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			DOOH: dooh,
			Imp:  imps,
		},
	}
}

func applyProcessedAuctionMutations(t *testing.T, result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], payload hookstage.ProcessedAuctionRequestPayload) hookstage.ProcessedAuctionRequestPayload {
	t.Helper()

	var err error
	for _, mutation := range result.ChangeSet.Mutations() {
		payload, err = mutation.Apply(payload)
		require.NoError(t, err)
	}
	require.NoError(t, payload.Request.RebuildRequest())

	return payload
}

func testLookupValue(path, key string, multiplier float64) impressionValue {
	return impressionValue{
		Path:       path,
		Key:        key,
		Multiplier: multiplier,
		SourceType: adcom1.MultiplierMeasurementVendorProvided,
		Vendor:     "measurement.example",
	}
}

func TestParseModuleConfig(t *testing.T) {
	cfg, err := parseModuleConfig(json.RawMessage(`{
		"enabled": true,
		"source": {
			"type": "request_lookup",
			"endpoint": "https://values.example.com/lookup"
		},
		"lookup_paths": ["dooh.id", "dooh.id", "imp.id"]
	}`))

	require.NoError(t, err)
	assert.Equal(t, sourceTypeRequestLookup, cfg.Source.Type)
	assert.Equal(t, "https://values.example.com/lookup", cfg.Source.Endpoint)
	assert.Equal(t, []string{lookupPathDOOHID, lookupPathImpID}, cfg.LookupPaths)
	assert.Equal(t, overwritePolicyMissingOnly, cfg.OverwritePolicy)
	assert.Equal(t, defaultTimeoutMS, cfg.TimeoutMS)
	assert.Equal(t, defaultCacheTTLSeconds, cfg.CacheTTLSeconds)
	assert.Equal(t, defaultNegativeCacheTTLSeconds, cfg.NegativeCacheTTLSeconds)
	assert.Equal(t, defaultCacheSizeBytes, cfg.CacheSizeBytes)
	assert.Equal(t, defaultSyncRateSeconds, cfg.Source.SyncRateSeconds)

	testCases := []struct {
		name string
		data json.RawMessage
	}{
		{
			name: "unsupported source type",
			data: json.RawMessage(`{"enabled": true, "source": {"type": "static"}}`),
		},
		{
			name: "unsupported lookup path",
			data: json.RawMessage(`{"source": {"endpoint": "https://values.example.com/lookup"}, "lookup_paths": ["site.id"]}`),
		},
		{
			name: "unsupported overwrite policy",
			data: json.RawMessage(`{"source": {"endpoint": "https://values.example.com/lookup"}, "overwrite_policy": "replace"}`),
		},
		{
			name: "negative timeout",
			data: json.RawMessage(`{"source": {"endpoint": "https://values.example.com/lookup"}, "timeout_ms": -1}`),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseModuleConfig(test.data)
			assert.Error(t, err)
		})
	}
}

func TestResolveImpressionLookups(t *testing.T) {
	request := newDOOHRequest(
		&openrtb2.DOOH{
			ID:        "screen-id",
			Name:      "screen-name",
			Publisher: &openrtb2.Publisher{ID: "publisher-id"},
		},
		openrtb2.Imp{ID: "imp-1", TagID: "tag-1"},
		openrtb2.Imp{ID: "imp-2", TagID: "tag-2"},
	)

	assignments, uniqueLookups, warnings := resolveImpressionLookups(request, testAccountID, []string{lookupPathDOOHName, lookupPathImpID})

	require.Empty(t, warnings)
	assert.Len(t, assignments, 2)
	assert.Equal(t, []lookupKey{{AccountID: testAccountID, Path: lookupPathDOOHName, Key: "screen-name"}}, uniqueLookups)

	request.DOOH.Name = ""
	assignments, uniqueLookups, warnings = resolveImpressionLookups(request, testAccountID, []string{lookupPathDOOHName, lookupPathImpID})

	require.Empty(t, warnings)
	assert.Equal(t, lookupKey{AccountID: testAccountID, Path: lookupPathImpID, Key: "imp-1"}, assignments[0])
	assert.Equal(t, lookupKey{AccountID: testAccountID, Path: lookupPathImpID, Key: "imp-2"}, assignments[1])
	assert.ElementsMatch(t, []lookupKey{
		{AccountID: testAccountID, Path: lookupPathImpID, Key: "imp-1"},
		{AccountID: testAccountID, Path: lookupPathImpID, Key: "imp-2"},
	}, uniqueLookups)
}

func TestHTTPValueProviderSendsBulkLookupRequest(t *testing.T) {
	var received bulkLookupRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))

		require.NoError(t, json.NewDecoder(r.Body).Decode(&received))
		_, err := w.Write([]byte(`{"values":[{"path":"dooh.id","key":"screen-1","multiplier":12.5,"sourcetype":1,"vendor":"measurement.example"}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	provider := newHTTPValueProvider(server.Client())
	cfg := defaultModuleConfig()
	cfg.Source.Endpoint = server.URL
	cfg.Source.Headers = map[string]string{"Authorization": "Bearer token"}

	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}
	values, warnings, err := provider.Lookup(context.Background(), cfg, testAccountID, []lookupKey{lookup})

	require.NoError(t, err)
	require.Empty(t, warnings)
	assert.Equal(t, testAccountID, received.AccountID)
	require.Len(t, received.Lookups, 1)
	assert.Empty(t, received.Lookups[0].AccountID)
	assert.Equal(t, lookupPathDOOHID, received.Lookups[0].Path)
	assert.Equal(t, "screen-1", received.Lookups[0].Key)
	assert.Equal(t, 12.5, values[lookup].Multiplier)
}

func TestSuccessfulMutationWritesImpQty(t *testing.T) {
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}
	provider := &fakeValueProvider{values: map[lookupKey]impressionValue{
		lookup: testLookupValue(lookupPathDOOHID, "screen-1", 14.2),
	}}
	module := newTestModule(provider, overwritePolicyMissingOnly)
	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}),
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

	require.NoError(t, err)
	require.Len(t, result.ChangeSet.Mutations(), 1)
	payload = applyProcessedAuctionMutations(t, result, payload)

	require.NotNil(t, payload.Request.Imp[0].Qty)
	assert.Equal(t, 14.2, payload.Request.Imp[0].Qty.Multiplier)
	assert.Equal(t, adcom1.MultiplierMeasurementVendorProvided, payload.Request.Imp[0].Qty.SourceType)
	assert.Equal(t, "measurement.example", payload.Request.Imp[0].Qty.Vendor)
}

func TestLookupDedupesAndUsesCache(t *testing.T) {
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}
	provider := &fakeValueProvider{values: map[lookupKey]impressionValue{
		lookup: testLookupValue(lookupPathDOOHID, "screen-1", 11.0),
	}}
	module := newTestModule(provider, overwritePolicyMissingOnly)

	firstPayload := hookstage.ProcessedAuctionRequestPayload{
		Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}, openrtb2.Imp{ID: "imp-2"}),
	}
	firstResult, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, firstPayload)
	require.NoError(t, err)
	require.Len(t, provider.calls, 1)
	assert.Equal(t, []lookupKey{lookup}, provider.calls[0])
	firstPayload = applyProcessedAuctionMutations(t, firstResult, firstPayload)
	assert.Equal(t, 11.0, firstPayload.Request.Imp[0].Qty.Multiplier)
	assert.Equal(t, 11.0, firstPayload.Request.Imp[1].Qty.Multiplier)

	secondPayload := hookstage.ProcessedAuctionRequestPayload{
		Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}),
	}
	secondResult, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, secondPayload)
	require.NoError(t, err)
	require.Len(t, provider.calls, 1)
	secondPayload = applyProcessedAuctionMutations(t, secondResult, secondPayload)
	assert.Equal(t, 11.0, secondPayload.Request.Imp[0].Qty.Multiplier)
}

func TestNonDOOHRequestNoops(t *testing.T) {
	provider := &fakeValueProvider{}
	module := newTestModule(provider, overwritePolicyMissingOnly)
	payload := hookstage.ProcessedAuctionRequestPayload{
		Request: &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{
			Site: &openrtb2.Site{},
			Imp:  []openrtb2.Imp{{ID: "imp-1"}},
		}},
	}

	result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

	require.NoError(t, err)
	assert.Empty(t, provider.calls)
	assert.Empty(t, result.ChangeSet.Mutations())
}

func TestOverwritePolicy(t *testing.T) {
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}

	t.Run("missing only preserves existing qty and skips lookup", func(t *testing.T) {
		provider := &fakeValueProvider{values: map[lookupKey]impressionValue{
			lookup: testLookupValue(lookupPathDOOHID, "screen-1", 20.0),
		}}
		module := newTestModule(provider, overwritePolicyMissingOnly)
		payload := hookstage.ProcessedAuctionRequestPayload{
			Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1", Qty: &openrtb2.Qty{Multiplier: 1.5}}),
		}

		result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

		require.NoError(t, err)
		assert.Empty(t, provider.calls)
		assert.Empty(t, result.ChangeSet.Mutations())
		assert.Equal(t, 1.5, payload.Request.Imp[0].Qty.Multiplier)
	})

	t.Run("always overwrites existing qty", func(t *testing.T) {
		provider := &fakeValueProvider{values: map[lookupKey]impressionValue{
			lookup: testLookupValue(lookupPathDOOHID, "screen-1", 20.0),
		}}
		module := newTestModule(provider, overwritePolicyAlways)
		payload := hookstage.ProcessedAuctionRequestPayload{
			Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1", Qty: &openrtb2.Qty{Multiplier: 1.5}}),
		}

		result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

		require.NoError(t, err)
		require.Len(t, provider.calls, 1)
		payload = applyProcessedAuctionMutations(t, result, payload)
		assert.Equal(t, 20.0, payload.Request.Imp[0].Qty.Multiplier)
	})
}

func TestLookupMissErrorAndInvalidValueSkipMutation(t *testing.T) {
	lookup := lookupKey{AccountID: testAccountID, Path: lookupPathDOOHID, Key: "screen-1"}

	t.Run("provider error", func(t *testing.T) {
		provider := &fakeValueProvider{err: errors.New("timeout")}
		module := newTestModule(provider, overwritePolicyMissingOnly)
		payload := hookstage.ProcessedAuctionRequestPayload{
			Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}),
		}

		result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

		require.NoError(t, err)
		require.Len(t, provider.calls, 1)
		assert.Empty(t, result.ChangeSet.Mutations())
		assert.Contains(t, result.Warnings[0], "lookup failed")
	})

	t.Run("miss is negatively cached", func(t *testing.T) {
		provider := &fakeValueProvider{values: map[lookupKey]impressionValue{}}
		module := newTestModule(provider, overwritePolicyMissingOnly)
		payload := hookstage.ProcessedAuctionRequestPayload{
			Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}),
		}

		firstResult, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)
		require.NoError(t, err)
		secondResult, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

		require.NoError(t, err)
		require.Len(t, provider.calls, 1)
		assert.Empty(t, firstResult.ChangeSet.Mutations())
		assert.Empty(t, secondResult.ChangeSet.Mutations())
		assert.Contains(t, firstResult.Warnings[0], "no DOOH qty found")
		assert.Contains(t, secondResult.Warnings[0], "no DOOH qty found")
	})

	t.Run("invalid value", func(t *testing.T) {
		provider := &fakeValueProvider{values: map[lookupKey]impressionValue{
			lookup: {Path: lookupPathDOOHID, Key: "screen-1", Multiplier: 0, SourceType: adcom1.MultiplierMeasurementVendorProvided, Vendor: "measurement.example"},
		}}
		module := newTestModule(provider, overwritePolicyMissingOnly)
		payload := hookstage.ProcessedAuctionRequestPayload{
			Request: newDOOHRequest(&openrtb2.DOOH{ID: "screen-1"}, openrtb2.Imp{ID: "imp-1"}),
		}

		result, err := module.HandleProcessedAuctionHook(context.Background(), hookstage.ModuleInvocationContext{AccountID: testAccountID}, payload)

		require.NoError(t, err)
		assert.Empty(t, result.ChangeSet.Mutations())
		require.NotEmpty(t, result.Warnings)
		assert.Contains(t, result.Warnings[0], "multiplier must be greater than 0")
	})
}
