package rulesengine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	hs "github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/rules"
	"github.com/prebid/prebid-server/v3/util/fetchutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
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
		name                 string
		argBidders           []string
		req                  *openrtb_ext.RequestWrapper
		userSync             IdFetcherMock
		ifSynced             *bool
		expectedMutationsLen int
	}{
		{
			name:                 "exclude-one-bidder",
			argBidders:           []string{"bidder1"},
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 1,
		},
		{
			name:                 "exclude_all_bidders",
			argBidders:           []string{"bidder1", "bidder2", "bidder3"},
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 1,
		},
		{
			name:                 "no_bidders_to_exclude",
			argBidders:           []string{},
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 0,
		},
		{
			name:                 "nil_bidders",
			argBidders:           nil,
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 0,
		},
		{
			name:                 "exclude-one-bidder_synced_valid_usersync",
			argBidders:           []string{"bidder1"},
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 1,
			ifSynced:             ptrutil.ToPtr(true),
			userSync:             IdFetcherMock{uid: "test", exists: true, notExpired: true},
		},
		{
			name:                 "exclude-one-bidder_not_synced_valid_usersync",
			argBidders:           []string{"bidder1"},
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 0,
			ifSynced:             ptrutil.ToPtr(false),
			userSync:             IdFetcherMock{uid: "test", exists: true, notExpired: true},
		},
		{
			name:                 "exclude-one-bidder_synced_invalid_usersync",
			argBidders:           []string{"bidder1"},
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 0,
			ifSynced:             ptrutil.ToPtr(true),
			userSync:             IdFetcherMock{uid: "test", exists: false, notExpired: true},
		},
		{
			name:                 "exclude-one-bidder_not_synced_invalid_usersync",
			argBidders:           []string{"bidder1"},
			req:                  mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedMutationsLen: 1,
			ifSynced:             ptrutil.ToPtr(false),
			userSync:             IdFetcherMock{uid: "test", exists: true, notExpired: false},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &ExcludeBidders{Args: config.ResultFuncParams{Bidders: tt.argBidders, IfSyncedId: tt.ifSynced}}
			hookResult := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
			}
			result := &ProcessedAuctionHookResult{
				HookResult:     hookResult,
				AllowedBidders: make(map[string]struct{}),
			}

			var us fetchutil.IdFetcher = &tt.userSync

			payload := &hs.ProcessedAuctionRequestPayload{
				Request:   tt.req,
				Usersyncs: &us,
			}
			err := eb.Call(payload, result, rules.ResultFunctionMeta{})

			assert.NoError(t, err)
			assert.Len(t, result.HookResult.ChangeSet.Mutations(), tt.expectedMutationsLen)
			if tt.expectedMutationsLen > 0 {
				assert.NotEmptyf(t, result.HookResult.ChangeSet, "change set is empty")
				assert.Equal(t, hs.MutationDelete, result.HookResult.ChangeSet.Mutations()[0].Type())
			}

		})
	}
}

type IdFetcherMock struct {
	uid        string
	exists     bool
	notExpired bool
}

func (fm *IdFetcherMock) GetUID(key string) (uid string, exists bool, notExpired bool) {
	return fm.uid, fm.exists, fm.notExpired
}

// HasAnyLiveSyncs is not executed in the result functions, but needed to complete the IDFetcher Interface
func (fm *IdFetcherMock) HasAnyLiveSyncs() bool {
	return false
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
		name            string
		argBidders      []string
		req             *openrtb_ext.RequestWrapper
		userSync        IdFetcherMock
		ifSynced        *bool
		expectedBidders []string
	}{
		{
			name:            "include_valid_bidders",
			argBidders:      []string{"bidder1", "bidder2"},
			expectedBidders: []string{"bidder1", "bidder2"},
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:            "include_no_bidders",
			argBidders:      []string{},
			expectedBidders: []string{},
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:            "include_non_existing_bidders",
			argBidders:      []string{"bidder4"},
			expectedBidders: []string{"bidder4"},
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:            "nil_bidders",
			argBidders:      nil,
			expectedBidders: nil,
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
		},
		{
			name:            "include_valid_bidders_synced_valid_usersync",
			argBidders:      []string{"bidder1", "bidder2"},
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			ifSynced:        ptrutil.ToPtr(true),
			userSync:        IdFetcherMock{uid: "test", exists: true, notExpired: true},
			expectedBidders: []string{"bidder1", "bidder2"},
		},
		{
			name:            "include_valid_bidders_not_synced_valid_usersync",
			argBidders:      []string{"bidder1", "bidder2"},
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			ifSynced:        ptrutil.ToPtr(false),
			userSync:        IdFetcherMock{uid: "test", exists: true, notExpired: true},
			expectedBidders: []string{},
		},
		{
			name:            "include_valid_bidders_synced_invalid_usersync",
			argBidders:      []string{"bidder1", "bidder2"},
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			ifSynced:        ptrutil.ToPtr(true),
			userSync:        IdFetcherMock{uid: "test", exists: false, notExpired: true},
			expectedBidders: []string{},
		},
		{
			name:            "include_valid_bidders_not_synced_invalid_usersync",
			argBidders:      []string{"bidder1", "bidder2"},
			req:             mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			ifSynced:        ptrutil.ToPtr(false),
			userSync:        IdFetcherMock{uid: "test", exists: false, notExpired: true},
			expectedBidders: []string{"bidder1", "bidder2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ib := &IncludeBidders{Args: config.ResultFuncParams{Bidders: tt.argBidders, IfSyncedId: tt.ifSynced}}
			hookResult := hs.HookResult[hs.ProcessedAuctionRequestPayload]{
				ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
			}

			result := &ProcessedAuctionHookResult{
				HookResult:     hookResult,
				AllowedBidders: make(map[string]struct{}),
			}

			var us fetchutil.IdFetcher = &tt.userSync

			payload := &hs.ProcessedAuctionRequestPayload{
				Request:   tt.req,
				Usersyncs: &us,
			}

			err := ib.Call(payload, result, rules.ResultFunctionMeta{})

			assert.NoError(t, err)
			assert.Emptyf(t, result.HookResult.ChangeSet, "change set is empty")
			assert.Len(t, result.HookResult.ChangeSet.Mutations(), 0)
			assert.Len(t, result.AllowedBidders, len(tt.expectedBidders))
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
