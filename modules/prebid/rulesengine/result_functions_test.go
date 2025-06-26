package rulesengine

import (
	"encoding/json"
	"github.com/prebid/openrtb/v20/openrtb2"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/stretchr/testify/assert"
	"testing"
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
			result := &hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
			}

			err := eb.Call(tt.req, result, rules.ResultFunctionMeta{})

			assert.NoError(t, err)
			assert.NotEmptyf(t, result.ChangeSet, "change set is empty")
			assert.Len(t, result.ChangeSet.Mutations(), 1)
			assert.Equal(t, hs.MutationUpdate, result.ChangeSet.Mutations()[0].Type())

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
			result := &hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
			}

			err := ib.Call(tt.req, result, rules.ResultFunctionMeta{})

			assert.NoError(t, err)
			assert.NotEmptyf(t, result.ChangeSet, "change set is empty")
			assert.Len(t, result.ChangeSet.Mutations(), 1)
			assert.Equal(t, hs.MutationUpdate, result.ChangeSet.Mutations()[0].Type())
		})
	}
}

func TestBuildIncludeBidders(t *testing.T) {
	tests := []struct {
		name       string
		argBidders []string
		req        *openrtb_ext.RequestWrapper
		expected   map[string]map[string]json.RawMessage
		expectErr  bool
	}{
		{
			name:       "include_valid_bidders",
			argBidders: []string{"bidder1", "bidder2"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder1": json.RawMessage(`{}`),
					"bidder2": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
		{
			name:       "no_matching_bidders",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expected:   map[string]map[string]json.RawMessage{"imp1": {}},
			expectErr:  false,
		},
		{
			name:       "req-imp-is-nil",
			argBidders: []string{"bidder4"},
			req:        &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
			expected:   map[string]map[string]json.RawMessage{},
			expectErr:  false,
		},
		{
			name:       "req-imp-is-empty",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithEmptyImp(t),
			expected:   map[string]map[string]json.RawMessage{},
			expectErr:  false,
		},
		{
			name:       "req-imp-ext-is-nil",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithImpExtNil(t),
			expectErr:  true,
		},
		{
			name:       "req-imp-ext-error",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithInvalidImpExt(t),
			expectErr:  true,
		},
		{
			name:       "req-imp-ext-prebid-is-nil",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithImpExtPrebidNil(t),
			expectErr:  true,
		},
		{
			name:       "include-one-bidder-already-in-req",
			argBidders: []string{"bidder1"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder1": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
		{
			name:       "include-one-bidder-not-in-req",
			argBidders: []string{"bidder2"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1"}),
			expected:   map[string]map[string]json.RawMessage{"imp1": {}},
			expectErr:  false,
		},
		{
			name:       "include-multiple-bidders-not-in-req",
			argBidders: []string{"bidder1", "bidder2", "bidder3"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder4"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {},
			},
			expectErr: false,
		},
		{
			name:       "include-one-bidder-in-req-and-one-not-in-req",
			argBidders: []string{"bidder1", "bidder2"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder2", "bidder3"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder2": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
		{
			name:       "multiple-imps",
			argBidders: []string{"bidder1", "bidder2"},
			req:        mockRequestWrapperWithBMultipleImpsWithBidders(t, []string{"bidder2", "bidder3"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder2": json.RawMessage(`{}`),
				},
				"imp2": {
					"bidder2": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildIncludeBidders(tt.req, tt.argBidders)
			if tt.expectErr {
				assert.Error(t, err, "expected error but got nil")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.True(t, compareMaps(result, tt.expected), "bidders to include do not match")
			}
		})
	}
}

func TestBuildExcludeBidders(t *testing.T) {
	tests := []struct {
		name       string
		argBidders []string
		req        *openrtb_ext.RequestWrapper
		expected   map[string]map[string]json.RawMessage
		expectErr  bool
	}{
		{
			name:       "exclude_valid_bidders",
			argBidders: []string{"bidder1"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder2": json.RawMessage(`{}`),
					"bidder3": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
		{
			name:       "exclude_all_bidders",
			argBidders: []string{"bidder1", "bidder2", "bidder3"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expected:   map[string]map[string]json.RawMessage{"imp1": {}},
			expectErr:  false,
		},
		{
			name:       "req-imp-is-nil",
			argBidders: []string{"bidder4"},
			req:        &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}},
			expected:   map[string]map[string]json.RawMessage{},
			expectErr:  false,
		},
		{
			name:       "req-imp-is-empty",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithEmptyImp(t),
			expected:   map[string]map[string]json.RawMessage{},
			expectErr:  false,
		},
		{
			name:       "req-imp-ext-is-nil",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithImpExtNil(t),
			expectErr:  true,
		},
		{
			name:       "req-imp-ext-error",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithInvalidImpExt(t),
			expectErr:  true,
		},
		{
			name:       "req-imp-ext-prebid-is-nil",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithImpExtPrebidNil(t),
			expectErr:  true,
		},
		{
			name:       "exclude-one-bidder-already-in-req",
			argBidders: []string{"bidder1"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {},
			},
			expectErr: false,
		},
		{
			name:       "exclude-one-bidder-not-in-req",
			argBidders: []string{"bidder2"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder1": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
		{
			name:       "exclude-multiple-bidders-not-in-req",
			argBidders: []string{"bidder1", "bidder2", "bidder3"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder4"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder4": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
		{
			name:       "exclude-one-bidder-in-req-and-one-not-in-req",
			argBidders: []string{"bidder1", "bidder2"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder2", "bidder3"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder3": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
		{
			name:       "multiple-imps",
			argBidders: []string{"bidder1", "bidder2"},
			req:        mockRequestWrapperWithBMultipleImpsWithBidders(t, []string{"bidder2", "bidder3"}),
			expected: map[string]map[string]json.RawMessage{
				"imp1": {
					"bidder3": json.RawMessage(`{}`),
				},
				"imp2": {
					"bidder3": json.RawMessage(`{}`),
				},
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildExcludeBidders(tt.req, tt.argBidders)
			if tt.expectErr {
				assert.Error(t, err, "expected error but got nil")
			} else {
				assert.NoError(t, err, "unexpected error")
				assert.True(t, compareMaps(result, tt.expected), "bidders to exclude do not match")
			}
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

func mockRequestWrapperWithBMultipleImpsWithBidders(t *testing.T, bidders []string) *openrtb_ext.RequestWrapper {
	//---imp1---
	imp1Wrapper := &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "imp1"}}
	imp1Ext, err := imp1Wrapper.GetImpExt()
	assert.NoError(t, err, "Failed to get ImpExt")
	imp1Prebid := &openrtb_ext.ExtImpPrebid{Bidder: make(map[string]json.RawMessage)}
	for _, bidder := range bidders {
		imp1Prebid.Bidder[bidder] = json.RawMessage(`{}`)
	}
	imp1Ext.SetPrebid(imp1Prebid)

	//---imp2---
	imp2Wrapper := &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "imp2"}}
	imp2Ext, err := imp2Wrapper.GetImpExt()
	assert.NoError(t, err, "Failed to get ImpExt")
	imp2Prebid := &openrtb_ext.ExtImpPrebid{Bidder: make(map[string]json.RawMessage)}

	for _, bidder := range bidders {
		imp2Prebid.Bidder[bidder] = json.RawMessage(`{}`)
	}
	imp2Ext.SetPrebid(imp2Prebid)

	rw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}
	rw.SetImp([]*openrtb_ext.ImpWrapper{imp1Wrapper, imp2Wrapper})

	return rw
}

func mockRequestWrapperWithImpExtPrebidNil(t *testing.T) *openrtb_ext.RequestWrapper {
	impWrapper := &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "imp1"}}

	impExt, err := impWrapper.GetImpExt()
	assert.NoError(t, err, "Failed to get ImpExt")
	impExt.SetPrebid(nil)
	rw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}
	rw.SetImp([]*openrtb_ext.ImpWrapper{impWrapper})

	return rw
}

func mockRequestWrapperWithInvalidImpExt(t *testing.T) *openrtb_ext.RequestWrapper {
	impWrapper := &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "imp1", Ext: json.RawMessage(`{"prebid":invalid}`)}}
	rw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}
	rw.SetImp([]*openrtb_ext.ImpWrapper{impWrapper})

	return rw
}

func mockRequestWrapperWithEmptyImp(t *testing.T) *openrtb_ext.RequestWrapper {
	rw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}
	rw.SetImp([]*openrtb_ext.ImpWrapper{})
	return rw
}

func mockRequestWrapperWithImpExtNil(t *testing.T) *openrtb_ext.RequestWrapper {
	impWrapper := &openrtb_ext.ImpWrapper{Imp: &openrtb2.Imp{ID: "imp1"}}
	rw := &openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}}
	rw.SetImp([]*openrtb_ext.ImpWrapper{impWrapper})
	return rw
}

// Helper function to compare maps
func compareMaps(a, b map[string]map[string]json.RawMessage) bool {
	if len(a) != len(b) {
		return false
	}
	for key, val := range a {
		if len(val) != len(b[key]) {
			return false
		}
		for subKey, subVal := range val {
			if string(subVal) != string(b[key][subKey]) {
				return false
			}
		}
	}
	return true
}
