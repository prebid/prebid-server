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
		name      string
		funcName  string
		params    json.RawMessage
		expectErr bool
	}{
		{
			name:      "Valid ExcludeBidders",
			funcName:  ExcludeBiddersName,
			params:    json.RawMessage(`{"bidders":["bidder1","bidder2"]}`),
			expectErr: false,
		},
		{
			name:      "Valid IncludeBidders",
			funcName:  IncludeBiddersName,
			params:    json.RawMessage(`{"bidders":["bidder3","bidder4"]}`),
			expectErr: false,
		},
		{
			name:      "Invalid Function Name",
			funcName:  "invalidFunction",
			params:    json.RawMessage(`{}`),
			expectErr: true,
		},
		{
			name:      "Invalid Params",
			funcName:  ExcludeBiddersName,
			params:    json.RawMessage(`invalid-json`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewProcessedAuctionRequestResultFunction(tt.funcName, tt.params)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
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
			name:       "Exclude valid bidders",
			argBidders: []string{"bidder1"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "Exclude all bidders",
			argBidders: []string{"bidder1", "bidder2", "bidder3"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "No bidders to exclude",
			argBidders: []string{},
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
			name:       "Include valid bidders",
			argBidders: []string{"bidder1", "bidder2"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "Include no bidders",
			argBidders: []string{},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:       "Include non-existent bidders",
			argBidders: []string{"bidder4"},
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
			name:       "Include valid bidders",
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
			name:       "No matching bidders",
			argBidders: []string{"bidder4"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expected:   map[string]map[string]json.RawMessage{"imp1": {}},
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildIncludeBidders(tt.req, tt.argBidders)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !compareMaps(result, tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
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
			name:       "Exclude valid bidders",
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
			name:       "Exclude all bidders",
			argBidders: []string{"bidder1", "bidder2", "bidder3"},
			req:        mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expected:   map[string]map[string]json.RawMessage{"imp1": {}},
			expectErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := buildExcludeBidders(tt.req, tt.argBidders)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if !compareMaps(result, tt.expected) {
					t.Errorf("expected %v, got %v", tt.expected, result)
				}
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
