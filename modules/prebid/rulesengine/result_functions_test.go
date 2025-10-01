package rulesengine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
)

func TestNewProcessedAuctionRequestResultFunction(t *testing.T) {
	tests := []struct {
		name       string
		funcName   string
		params     json.RawMessage
		expectType ProcessedAuctionResultFunc
		expectErr  bool
	}{
		{
			name:       "valid_excludeBidders",
			funcName:   ExcludeBiddersName,
			params:     json.RawMessage(`{"bidders":["bidder1","bidder2"]}`),
			expectType: &ExcludeBidders{},
			expectErr:  false,
		},
		{
			name:       "valid_includeBidders",
			funcName:   IncludeBiddersName,
			params:     json.RawMessage(`{"bidders":["bidder3","bidder4"]}`),
			expectType: &IncludeBidders{},
			expectErr:  false,
		},
		{
			name:      "valid_excludeBidders_empty_bidders",
			funcName:  ExcludeBiddersName,
			params:    json.RawMessage(`{"bidders":null}`),
			expectErr: true,
		},
		{
			name:      "valid_includeBidders_empty_bidders",
			funcName:  IncludeBiddersName,
			params:    json.RawMessage(`{"bidders":null}`),
			expectErr: true,
		},
		{
			name:      "invalid_function_name",
			funcName:  "invalidFunction",
			params:    json.RawMessage(`{}`),
			expectErr: true,
		},
		{
			name:      "invalid-exclude-bidders-params",
			funcName:  ExcludeBiddersName,
			params:    json.RawMessage(`invalid-json`),
			expectErr: true,
		},
		{
			name:      "invalid-include-bidders-params",
			funcName:  IncludeBiddersName,
			params:    json.RawMessage(`invalid-json`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := NewProcessedAuctionRequestResultFunction(tt.funcName, tt.params)
			if tt.expectErr {
				assert.Error(t, err, "expected error but got nil")
			} else {
				assert.IsType(t, tt.expectType, v)
			}
		})
	}
}

func TestExcludeBiddersCall(t *testing.T) {
	tests := []struct {
		name       string
		argBidders []string
		req        *openrtb_ext.RequestWrapper
	}{
		{
			name:       "exclude-one-bidder",
			argBidders: []string{"bidder1"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "exclude_all_bidders",
			argBidders: []string{"bidder1", "bidder2", "bidder3"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "no_bidders_to_exclude",
			argBidders: []string{},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "nil_bidders",
			argBidders: nil,
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &ExcludeBidders{Args: config.ResultFuncParams{Bidders: tt.argBidders}}
			hookResult := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
			}
			result := &ProcessedAuctionHookResult{
				HookResult:     hookResult,
				AllowedBidders: make(map[string]struct{}),
			}

			err := eb.Call(tt.req, result, rules.ResultFunctionMeta{})

			assert.NoError(t, err)
			assert.NotEmptyf(t, result.HookResult.ChangeSet, "change set is empty")
			assert.Len(t, result.HookResult.ChangeSet.Mutations(), 1)
			assert.Equal(t, hs.MutationDelete, result.HookResult.ChangeSet.Mutations()[0].Type())

		})
	}
}

func TestIncludeBiddersName(t *testing.T) {
	ib := &IncludeBidders{}
	actualName := ib.Name()
	assert.Equal(t, IncludeBiddersName, actualName, "IncludeBidders name should match expected value")
}

func TestExcludeBiddersName(t *testing.T) {
	eb := &ExcludeBidders{}
	actualName := eb.Name()
	assert.Equal(t, ExcludeBiddersName, actualName, "ExcludeBidders name should match expected value")
}

func TestIncludeBiddersCall(t *testing.T) {
	tests := []struct {
		name       string
		argBidders []string
		req        *openrtb_ext.RequestWrapper
	}{
		{
			name:       "include_valid_bidders",
			argBidders: []string{"bidder1", "bidder2"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "include_no_bidders",
			argBidders: []string{},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "include_non_existing_bidders",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "nil_bidders",
			argBidders: nil,
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ib := &IncludeBidders{Args: config.ResultFuncParams{Bidders: tt.argBidders}}
			hookResult := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
			}

			result := &ProcessedAuctionHookResult{
				HookResult:     hookResult,
				AllowedBidders: make(map[string]struct{}),
			}

			err := ib.Call(tt.req, result, rules.ResultFunctionMeta{})

			assert.NoError(t, err)
			assert.Emptyf(t, result.HookResult.ChangeSet, "change set is empty")
			assert.Len(t, result.HookResult.ChangeSet.Mutations(), 0)
			assert.Len(t, result.AllowedBidders, len(tt.argBidders))
		})
	}
}

// Helper function to mock RequestWrapper with bidders
func mockRequestWrapperWithBidders(t *testing.T, bidders []string) *openrtb_ext.RequestWrapper {
	impWrapper := &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "imp1"}}

	impExt, err := impWrapper.GetImpExt()
	assert.NoError(t, err, "Failed to get ImpExt")
	impPrebid := &openrtb_ext.ExtImpPrebid{Bidder: make(map[string]json.RawMessage)}

	for _, bidder := range bidders {
		impPrebid.Bidder[bidder] = json.RawMessage(`{}`)
	}
	impExt.SetPrebid(impPrebid)
	rw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}
	rw.SetImp([]*openrtb_ext.ImpWrapper{impWrapper})

	return rw
}
