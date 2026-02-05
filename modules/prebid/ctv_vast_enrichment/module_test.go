package vast

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/moduledeps"
	"github.com/prebid/prebid-server/v3/modules/prebid/ctv_vast_enrichment/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuilder(t *testing.T) {
	testCases := []struct {
		name        string
		config      json.RawMessage
		expectError bool
	}{
		{
			name:        "empty config",
			config:      json.RawMessage(`{}`),
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: false,
		},
		{
			name:        "valid config",
			config:      json.RawMessage(`{"enabled": true, "receiver": "GAM_SSU", "default_currency": "USD"}`),
			expectError: false,
		},
		{
			name:        "invalid json",
			config:      json.RawMessage(`{invalid}`),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			module, err := Builder(tc.config, moduledeps.ModuleDeps{})

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, module)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, module)

				_, ok := module.(Module)
				assert.True(t, ok, "Builder should return Module type")
			}
		})
	}
}

func TestHandleRawBidderResponseHook_NoAccountConfig(t *testing.T) {
	module := Module{}

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:  "bid1",
						AdM: `<VAST version="3.0"><Ad><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`,
					},
				},
			},
		},
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: nil,
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
}

func TestHandleRawBidderResponseHook_ModuleDisabled(t *testing.T) {
	module := Module{}

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:  "bid1",
						AdM: `<VAST version="3.0"><Ad><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`,
					},
				},
			},
		},
	}

	// Module is disabled
	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": false}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
	// No mutation should be applied when disabled
}

func TestHandleRawBidderResponseHook_EmptyBidResponse(t *testing.T) {
	module := Module{}

	payload := hookstage.RawBidderResponsePayload{
		Bidder:         "appnexus",
		BidderResponse: nil,
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
}

func TestHandleRawBidderResponseHook_NoBids(t *testing.T) {
	module := Module{}

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{},
		},
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
}

func TestHandleRawBidderResponseHook_EnrichesVAST(t *testing.T) {
	module := Module{
		hostConfig: CTVVastConfig{
			DefaultCurrency: "USD",
		},
	}

	originalVast := `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:      "bid1",
						Price:   1.50,
						ADomain: []string{"advertiser.com"},
						AdM:     originalVast,
					},
				},
			},
		},
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	require.NoError(t, err)
	assert.Empty(t, result.Errors)

	// Verify the bid was enriched
	enrichedAdM := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, enrichedAdM, "Pricing")
	assert.Contains(t, enrichedAdM, "1.500000")
	assert.Contains(t, enrichedAdM, "CPM")
	assert.Contains(t, enrichedAdM, "USD")
}

func TestHandleRawBidderResponseHook_SkipsNonVAST(t *testing.T) {
	module := Module{}

	originalAdM := `<html><body>Banner ad content</body></html>`

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "bid1",
						Price: 1.50,
						AdM:   originalAdM,
					},
				},
			},
		},
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)

	// Non-VAST content should be unchanged
	assert.Equal(t, originalAdM, payload.BidderResponse.Bids[0].Bid.AdM)
}

func TestHandleRawBidderResponseHook_SkipsEmptyAdM(t *testing.T) {
	module := Module{}

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "bid1",
						Price: 1.50,
						AdM:   "",
					},
				},
			},
		},
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	assert.NoError(t, err)
	assert.Empty(t, result.Errors)
}

func TestHandleRawBidderResponseHook_InvalidAccountConfig(t *testing.T) {
	module := Module{}

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:  "bid1",
						AdM: `<VAST version="3.0"></VAST>`,
					},
				},
			},
		},
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{invalid json}`),
	}

	_, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	assert.Error(t, err)
}

func TestHandleRawBidderResponseHook_MergesHostAndAccountConfig(t *testing.T) {
	// Host config with USD currency
	module := Module{
		hostConfig: CTVVastConfig{
			DefaultCurrency: "USD",
			Receiver:        "GENERIC",
		},
	}

	originalVast := `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "bid1",
						Price: 2.00,
						AdM:   originalVast,
					},
				},
			},
		},
	}

	// Account config overrides currency to EUR
	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true, "default_currency": "EUR"}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	require.NoError(t, err)
	assert.Empty(t, result.Errors)

	// Verify EUR currency was used (account overrides host)
	enrichedAdM := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, enrichedAdM, "EUR")
}

func TestHandleRawBidderResponseHook_MultipleBids(t *testing.T) {
	module := Module{
		hostConfig: CTVVastConfig{
			DefaultCurrency: "USD",
		},
	}

	vastTemplate := `<VAST version="3.0"><Ad id="%s"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "bid1",
						Price: 1.50,
						AdM:   `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`,
					},
				},
				{
					Bid: &openrtb2.Bid{
						ID:    "bid2",
						Price: 2.00,
						AdM:   `<VAST version="3.0"><Ad id="ad2"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad 2</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`,
					},
				},
			},
		},
	}
	_ = vastTemplate // For reference

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	require.NoError(t, err)
	assert.Empty(t, result.Errors)

	// Both bids should be enriched
	assert.Contains(t, payload.BidderResponse.Bids[0].Bid.AdM, "1.500000")
	assert.Contains(t, payload.BidderResponse.Bids[1].Bid.AdM, "2.000000")
}

func TestHandleRawBidderResponseHook_PreservesExistingPricing(t *testing.T) {
	module := Module{
		hostConfig: CTVVastConfig{
			DefaultCurrency: "USD",
		},
	}

	// VAST already has pricing
	vastWithPricing := `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test Ad</AdTitle><Pricing model="CPM" currency="GBP">3.00</Pricing><Creatives></Creatives></InLine></Ad></VAST>`

	payload := hookstage.RawBidderResponsePayload{
		Bidder: "appnexus",
		BidderResponse: &adapters.BidderResponse{
			Bids: []*adapters.TypedBid{
				{
					Bid: &openrtb2.Bid{
						ID:    "bid1",
						Price: 1.50, // Different price
						AdM:   vastWithPricing,
					},
				},
			},
		},
	}

	miCtx := hookstage.ModuleInvocationContext{
		AccountConfig: json.RawMessage(`{"enabled": true}`),
	}

	result, err := module.HandleRawBidderResponseHook(context.Background(), miCtx, payload)

	require.NoError(t, err)
	assert.Empty(t, result.Errors)

	// Original pricing should be preserved (VAST wins)
	enrichedAdM := payload.BidderResponse.Bids[0].Bid.AdM
	assert.Contains(t, enrichedAdM, "GBP")
	assert.Contains(t, enrichedAdM, "3.00")
	assert.NotContains(t, enrichedAdM, "1.50")
}

func TestConfigToReceiverConfig(t *testing.T) {
	testCases := []struct {
		name     string
		input    CTVVastConfig
		expected ReceiverConfig
	}{
		{
			name:     "empty config uses defaults",
			input:    CTVVastConfig{},
			expected: DefaultConfig(),
		},
		{
			name: "receiver GAM_SSU",
			input: CTVVastConfig{
				Receiver: "GAM_SSU",
			},
			expected: func() ReceiverConfig {
				rc := DefaultConfig()
				rc.Receiver = ReceiverGAMSSU
				return rc
			}(),
		},
		{
			name: "receiver GENERIC",
			input: CTVVastConfig{
				Receiver: "GENERIC",
			},
			expected: func() ReceiverConfig {
				rc := DefaultConfig()
				rc.Receiver = ReceiverGeneric
				return rc
			}(),
		},
		{
			name: "custom currency",
			input: CTVVastConfig{
				DefaultCurrency: "EUR",
			},
			expected: func() ReceiverConfig {
				rc := DefaultConfig()
				rc.DefaultCurrency = "EUR"
				return rc
			}(),
		},
		{
			name: "selection strategy max_revenue",
			input: CTVVastConfig{
				SelectionStrategy: "max_revenue",
			},
			expected: func() ReceiverConfig {
				rc := DefaultConfig()
				rc.SelectionStrategy = SelectionMaxRevenue
				return rc
			}(),
		},
		{
			name: "collision policy reject",
			input: CTVVastConfig{
				CollisionPolicy: "reject",
			},
			expected: func() ReceiverConfig {
				rc := DefaultConfig()
				rc.CollisionPolicy = CollisionReject
				return rc
			}(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := configToReceiverConfig(tc.input)
			assert.Equal(t, tc.expected.Receiver, result.Receiver)
			assert.Equal(t, tc.expected.DefaultCurrency, result.DefaultCurrency)
			assert.Equal(t, tc.expected.SelectionStrategy, result.SelectionStrategy)
			assert.Equal(t, tc.expected.CollisionPolicy, result.CollisionPolicy)
		})
	}
}

func TestEnrichVastDocument(t *testing.T) {
	testCases := []struct {
		name           string
		inputVast      string
		meta           CanonicalMeta
		cfg            ReceiverConfig
		expectPricing  bool
		expectAdomain  bool
	}{
		{
			name:      "adds pricing when missing",
			inputVast: `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`,
			meta: CanonicalMeta{
				Price:    1.50,
				Currency: "USD",
			},
			cfg: ReceiverConfig{
				DefaultCurrency: "USD",
			},
			expectPricing: true,
			expectAdomain: false,
		},
		{
			name:      "adds advertiser when missing",
			inputVast: `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`,
			meta: CanonicalMeta{
				Price:   1.50,
				Adomain: "advertiser.com",
			},
			cfg: ReceiverConfig{
				DefaultCurrency: "USD",
			},
			expectPricing: true,
			expectAdomain: true,
		},
		{
			name:      "does not add pricing when price is zero",
			inputVast: `<VAST version="3.0"><Ad id="ad1"><InLine><AdSystem>Test</AdSystem><AdTitle>Test</AdTitle><Creatives></Creatives></InLine></Ad></VAST>`,
			meta: CanonicalMeta{
				Price: 0,
			},
			cfg: ReceiverConfig{
				DefaultCurrency: "USD",
			},
			expectPricing: false,
			expectAdomain: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			vastDoc, err := parseTestVast(tc.inputVast)
			require.NoError(t, err)

			result := enrichVastDocument(vastDoc, tc.meta, tc.cfg)
			require.NotNil(t, result)

			xmlBytes, err := result.Marshal()
			require.NoError(t, err)

			xmlStr := string(xmlBytes)

			if tc.expectPricing {
				assert.Contains(t, xmlStr, "Pricing")
			} else {
				assert.NotContains(t, xmlStr, "Pricing")
			}

			if tc.expectAdomain {
				assert.Contains(t, xmlStr, tc.meta.Adomain)
			}
		})
	}
}

func TestEnrichVastDocument_NilInput(t *testing.T) {
	result := enrichVastDocument(nil, CanonicalMeta{}, ReceiverConfig{})
	assert.Nil(t, result)
}

// parseTestVast is a helper to parse VAST XML for tests
func parseTestVast(xmlStr string) (*model.Vast, error) {
	return model.ParseVastAdm(xmlStr)
}
