package rulesengine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	hs "github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/prebid/rulesengine/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/rules"
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

// TestExcludeBiddersCallEmitsWarnings verifies that ExcludeBidders.Call appends a debug warning
// describing why bidders were removed, and that only bidders actually present in the request are
// mentioned (not the rule's full configured exclusion list).
func TestExcludeBiddersCallEmitsWarnings(t *testing.T) {
	tests := []struct {
		name             string
		argBidders       []string
		meta             rules.ResultFunctionMeta
		req              *openrtb_ext.RequestWrapper
		expectedWarnings []string
	}{
		{
			name:       "value_function_quoted_country",
			argBidders: []string{"bidder1"},
			meta: rules.ResultFunctionMeta{
				AnalyticsKey: "bidderConfig",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountry", FuncResult: "JPN"},
				},
			},
			req: mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2", "bidder3"}),
			expectedWarnings: []string{
				`Bidder [bidder1] was removed from the request by the rules engine ruleset "bidderConfig": deviceCountry rule evaluated to "JPN"`,
			},
		},
		{
			name:       "ruleset_name_preferred_over_analytics_key",
			argBidders: []string{"rise"},
			meta: rules.ResultFunctionMeta{
				RulesetName:  "microsoft-account-rise",
				AnalyticsKey: "someKey",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountryIn", FuncResult: "false"},
				},
			},
			req: mockRequestWrapperWithBidders(t, []string{"rise", "bidder2"}),
			expectedWarnings: []string{
				`Bidder [rise] was removed from the request by the rules engine ruleset "microsoft-account-rise": deviceCountryIn rule evaluated to false`,
			},
		},
		{
			name:       "warn_only_for_bidders_present_in_request",
			argBidders: []string{"bidder1", "bidder2", "bidder3"},
			meta: rules.ResultFunctionMeta{
				AnalyticsKey: "bidderConfig",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountryIn", FuncResult: "true"},
				},
			},
			req: mockRequestWrapperWithBidders(t, []string{"bidder2"}),
			expectedWarnings: []string{
				`Bidder [bidder2] was removed from the request by the rules engine ruleset "bidderConfig": deviceCountryIn rule evaluated to true`,
			},
		},
		{
			name:             "no_warning_when_no_configured_bidder_present",
			argBidders:       []string{"bidder1", "bidder2"},
			meta:             rules.ResultFunctionMeta{},
			req:              mockRequestWrapperWithBidders(t, []string{"bidder3"}),
			expectedWarnings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			eb := &ExcludeBidders{Args: config.ResultFuncParams{Bidders: tt.argBidders}}
			result := &ProcessedAuctionHookResult{
				HookResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{
					ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
				},
				AllowedBidders: make(map[string]struct{}),
			}

			err := eb.Call(tt.req, result, tt.meta)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedWarnings, result.HookResult.Warnings)
			if len(tt.expectedWarnings) == 0 {
				assert.Empty(t, result.HookResult.AnalyticsTags.Activities)
				return
			}
			assert.Len(t, result.HookResult.AnalyticsTags.Activities, 1)
			activity := result.HookResult.AnalyticsTags.Activities[0]
			assert.Equal(t, rulesEngineBidderFilteringActivity, activity.Name)
			assert.Equal(t, hookanalytics.ActivityStatusSuccess, activity.Status)
			assert.Len(t, activity.Results, 1)
			assert.Equal(t, hookanalytics.ResultStatusBlock, activity.Results[0].Status)
			assert.Equal(t, errortypes.RulesEngineBidderExcludedWarningCode, activity.Results[0].Values["code"])
			assert.Equal(t, rulesEngineReasonExcludedByRule, activity.Results[0].Values["reason"])
			assert.NotContains(t, activity.Results[0].Values, "warning", "human-readable warning belongs in debug output, not analytics tags")
		})
	}
}

// TestExcludeBiddersCallMultipleExclusions proves that two separate ExcludeBidders.Call
// invocations for different reasons produce two distinct warnings - reasons are never merged.
func TestExcludeBiddersCallMultipleExclusions(t *testing.T) {
	req := mockRequestWrapperWithBidders(t, []string{"bidderA", "bidderB", "bidderC"})
	result := &ProcessedAuctionHookResult{
		HookResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{
			ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
		},
		AllowedBidders: make(map[string]struct{}),
	}

	ebA := &ExcludeBidders{Args: config.ResultFuncParams{Bidders: []string{"bidderA"}}}
	metaA := rules.ResultFunctionMeta{
		AnalyticsKey: "bidderConfig",
		SchemaFunctionResults: []rules.SchemaFunctionStep{
			{FuncName: "deviceCountry", FuncResult: "JPN"},
		},
	}
	assert.NoError(t, ebA.Call(req, result, metaA))

	ebB := &ExcludeBidders{Args: config.ResultFuncParams{Bidders: []string{"bidderB"}}}
	metaB := rules.ResultFunctionMeta{
		RulesetName: "customRuleset",
		SchemaFunctionResults: []rules.SchemaFunctionStep{
			{FuncName: "channel", FuncResult: "web"},
		},
	}
	assert.NoError(t, ebB.Call(req, result, metaB))

	assert.Len(t, result.HookResult.ChangeSet.Mutations(), 2)
	assert.Equal(t, []string{
		`Bidder [bidderA] was removed from the request by the rules engine ruleset "bidderConfig": deviceCountry rule evaluated to "JPN"`,
		`Bidder [bidderB] was removed from the request by the rules engine ruleset "customRuleset": channel rule evaluated to "web"`,
	}, result.HookResult.Warnings)
}

func TestExcludeBiddersCallMultipleBiddersEmitsOneAnalyticsResultPerBidder(t *testing.T) {
	req := mockRequestWrapperWithBidders(t, []string{"openx", "rubicon", "pubmatic", "appnexus"})
	result := &ProcessedAuctionHookResult{
		HookResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{
			ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
		},
		AllowedBidders: make(map[string]struct{}),
	}
	exclude := &ExcludeBidders{Args: config.ResultFuncParams{Bidders: []string{"openx", "rubicon", "pubmatic"}}}
	meta := rules.ResultFunctionMeta{
		RulesetName: "geo-ruleset",
		SchemaFunctionResults: []rules.SchemaFunctionStep{
			{FuncName: "deviceCountryIn", FuncResult: "true"},
		},
	}

	err := exclude.Call(req, result, meta)

	assert.NoError(t, err)
	assert.Equal(t, []string{
		`Bidders [openx, rubicon, pubmatic] were removed from the request by the rules engine ruleset "geo-ruleset": deviceCountryIn rule evaluated to true`,
	}, result.HookResult.Warnings)
	assert.Len(t, result.HookResult.AnalyticsTags.Activities, 1)
	activity := result.HookResult.AnalyticsTags.Activities[0]
	assert.Equal(t, rulesEngineBidderFilteringActivity, activity.Name)
	assert.Equal(t, hookanalytics.ActivityStatusSuccess, activity.Status)
	assert.Len(t, activity.Results, 3)
	for i, bidder := range []string{"openx", "rubicon", "pubmatic"} {
		assert.Equal(t, hookanalytics.ResultStatusBlock, activity.Results[i].Status)
		assert.Equal(t, bidder, activity.Results[i].AppliedTo.Bidder)
		assert.Equal(t, true, activity.Results[i].AppliedTo.Request)
		assert.Equal(t, errortypes.RulesEngineBidderExcludedWarningCode, activity.Results[i].Values["code"])
		assert.Equal(t, rulesEngineReasonExcludedByRule, activity.Results[i].Values["reason"])
		assert.Equal(t, "geo-ruleset", activity.Results[i].Values["ruleset"])
	}
}

// TestBuildExclusionWarning asserts the warning-string builder directly across value/boolean/empty
// conditions, ruleset-name-vs-analytics-key preference, and single/plural bidder phrasing.
func TestBuildExclusionWarning(t *testing.T) {
	tests := []struct {
		name     string
		bidders  []string
		meta     rules.ResultFunctionMeta
		expected string
	}{
		{
			name:    "plural_full_context",
			bidders: []string{"bidder1", "bidder3"},
			meta: rules.ResultFunctionMeta{
				AnalyticsKey: "bidderConfig",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountry", FuncResult: "JPN"},
				},
			},
			expected: `Bidders [bidder1, bidder3] were removed from the request by the rules engine ruleset "bidderConfig": deviceCountry rule evaluated to "JPN"`,
		},
		{
			name:    "ruleset_name_preferred_over_analytics_key",
			bidders: []string{"openx"},
			meta: rules.ResultFunctionMeta{
				RulesetName:  "cross-account-openx",
				AnalyticsKey: "someAnalyticsKey",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountryIn", FuncResult: "true"},
				},
			},
			expected: `Bidder [openx] was removed from the request by the rules engine ruleset "cross-account-openx": deviceCountryIn rule evaluated to true`,
		},
		{
			name:    "analytics_key_used_when_no_ruleset_name",
			bidders: []string{"openx"},
			meta: rules.ResultFunctionMeta{
				AnalyticsKey: "bidderConfig",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountry", FuncResult: "USA"},
				},
			},
			expected: `Bidder [openx] was removed from the request by the rules engine ruleset "bidderConfig": deviceCountry rule evaluated to "USA"`,
		},
		{
			name:     "no_schema_context",
			bidders:  []string{"bidder1"},
			meta:     rules.ResultFunctionMeta{},
			expected: `Bidder [bidder1] was removed from the request by the rules engine`,
		},
		{
			name:    "ruleset_only",
			bidders: []string{"bidder1"},
			meta: rules.ResultFunctionMeta{
				AnalyticsKey: "bidderConfig",
			},
			expected: `Bidder [bidder1] was removed from the request by the rules engine ruleset "bidderConfig"`,
		},
		{
			name:    "boolean_condition_false",
			bidders: []string{"rise"},
			meta: rules.ResultFunctionMeta{
				RulesetName: "microsoft-account-rise",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountryIn", FuncResult: "false"},
				},
			},
			expected: `Bidder [rise] was removed from the request by the rules engine ruleset "microsoft-account-rise": deviceCountryIn rule evaluated to false`,
		},
		{
			name:    "empty_value_result_renders_no_value",
			bidders: []string{"openx"},
			meta: rules.ResultFunctionMeta{
				AnalyticsKey: "bidderConfig",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountry", FuncResult: ""},
				},
			},
			expected: `Bidder [openx] was removed from the request by the rules engine ruleset "bidderConfig": deviceCountry rule evaluated to (no value)`,
		},
		{
			name:    "mixed_value_boolean_and_empty_conditions",
			bidders: []string{"openx"},
			meta: rules.ResultFunctionMeta{
				AnalyticsKey: "mixed-empty-ruleset",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountry", FuncResult: ""},
					{FuncName: "deviceCountryIn", FuncResult: "false"},
					{FuncName: "channel", FuncResult: "web"},
				},
			},
			expected: `Bidder [openx] was removed from the request by the rules engine ruleset "mixed-empty-ruleset": deviceCountry rule evaluated to (no value), deviceCountryIn rule evaluated to false, channel rule evaluated to "web"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildExclusionWarning(tt.bidders, tt.meta))
		})
	}
}

// TestDescribeSchemaStep exercises single-step rendering across every result kind: value functions
// (quoted), boolean membership/availability functions (unquoted true/false), and the empty-value
// edge case (rendered as "(no value)").
func TestDescribeSchemaStep(t *testing.T) {
	tests := []struct {
		name     string
		step     rules.SchemaFunctionStep
		expected string
	}{
		{name: "value_deviceCountry", step: rules.SchemaFunctionStep{FuncName: "deviceCountry", FuncResult: "JPN"}, expected: `deviceCountry rule evaluated to "JPN"`},
		{name: "value_dataCenter", step: rules.SchemaFunctionStep{FuncName: "dataCenter", FuncResult: "us-west"}, expected: `dataCenter rule evaluated to "us-west"`},
		{name: "value_channel", step: rules.SchemaFunctionStep{FuncName: "channel", FuncResult: "web"}, expected: `channel rule evaluated to "web"`},
		{name: "empty_value_deviceCountry", step: rules.SchemaFunctionStep{FuncName: "deviceCountry", FuncResult: ""}, expected: `deviceCountry rule evaluated to (no value)`},
		{name: "empty_value_channel", step: rules.SchemaFunctionStep{FuncName: "channel", FuncResult: ""}, expected: `channel rule evaluated to (no value)`},
		{name: "boolean_deviceCountryIn_true", step: rules.SchemaFunctionStep{FuncName: "deviceCountryIn", FuncResult: "true"}, expected: `deviceCountryIn rule evaluated to true`},
		{name: "boolean_deviceCountryIn_false", step: rules.SchemaFunctionStep{FuncName: "deviceCountryIn", FuncResult: "false"}, expected: `deviceCountryIn rule evaluated to false`},
		{name: "boolean_dataCenterIn", step: rules.SchemaFunctionStep{FuncName: "dataCenterIn", FuncResult: "true"}, expected: `dataCenterIn rule evaluated to true`},
		{name: "boolean_eidAvailable", step: rules.SchemaFunctionStep{FuncName: "eidAvailable", FuncResult: "false"}, expected: `eidAvailable rule evaluated to false`},
		{name: "boolean_eidIn", step: rules.SchemaFunctionStep{FuncName: "eidIn", FuncResult: "true"}, expected: `eidIn rule evaluated to true`},
		{name: "boolean_userFpdAvailable", step: rules.SchemaFunctionStep{FuncName: "userFpdAvailable", FuncResult: "false"}, expected: `userFpdAvailable rule evaluated to false`},
		{name: "boolean_fpdAvailable", step: rules.SchemaFunctionStep{FuncName: "fpdAvailable", FuncResult: "true"}, expected: `fpdAvailable rule evaluated to true`},
		{name: "boolean_gppSidAvailable", step: rules.SchemaFunctionStep{FuncName: "gppSidAvailable", FuncResult: "true"}, expected: `gppSidAvailable rule evaluated to true`},
		{name: "boolean_gppSidIn", step: rules.SchemaFunctionStep{FuncName: "gppSidIn", FuncResult: "false"}, expected: `gppSidIn rule evaluated to false`},
		{name: "boolean_percent", step: rules.SchemaFunctionStep{FuncName: "percent", FuncResult: "true"}, expected: `percent rule evaluated to true`},
		{name: "boolean_tcfInScope", step: rules.SchemaFunctionStep{FuncName: "tcfInScope", FuncResult: "false"}, expected: `tcfInScope rule evaluated to false`},
		{name: "unexpected_value_quoted", step: rules.SchemaFunctionStep{FuncName: "deviceCountry", FuncResult: "US-CA"}, expected: `deviceCountry rule evaluated to "US-CA"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, describeSchemaStep(tt.step))
		})
	}
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
			// Every include invocation records its ruleset context so the removal warning can be
			// built after all rulesets have run.
			assert.Len(t, result.IncludeContexts, 1)
		})
	}
}

// TestIncludeBiddersCallRecordsContext verifies that each IncludeBidders.Call appends its
// ResultFunctionMeta to IncludeContexts (accumulating across multiple invocations) so the final
// removal warning can attribute the removal to the correct ruleset(s).
func TestIncludeBiddersCallRecordsContext(t *testing.T) {
	req := mockRequestWrapperWithBidders(t, []string{"bidder1", "bidder2"})
	result := &ProcessedAuctionHookResult{
		HookResult:     hs.HookResult[hs.ProcessedAuctionRequestPayload]{ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{}},
		AllowedBidders: make(map[string]struct{}),
	}

	metaA := rules.ResultFunctionMeta{RulesetName: "rulesetA"}
	metaB := rules.ResultFunctionMeta{AnalyticsKey: "bidderConfig"}

	ibA := &IncludeBidders{Args: config.ResultFuncParams{Bidders: []string{"bidder1"}}}
	assert.NoError(t, ibA.Call(req, result, metaA))

	ibB := &IncludeBidders{Args: config.ResultFuncParams{Bidders: []string{"bidder2"}}}
	assert.NoError(t, ibB.Call(req, result, metaB))

	assert.Equal(t, []rules.ResultFunctionMeta{metaA, metaB}, result.IncludeContexts)
	assert.Equal(t, map[string]struct{}{"bidder1": {}, "bidder2": {}}, result.AllowedBidders)
}

// TestBiddersRemovedByInclude asserts that only bidders present in the request but absent from the
// accumulated allow-list are reported, and that the output is sorted for deterministic warnings.
func TestBiddersRemovedByInclude(t *testing.T) {
	tests := []struct {
		name     string
		req      *openrtb_ext.RequestWrapper
		allowed  map[string]struct{}
		expected []string
	}{
		{
			name:     "removes_present_bidders_not_allowed_sorted",
			req:      mockRequestWrapperWithBidders(t, []string{"openx", "rise", "appnexus"}),
			allowed:  map[string]struct{}{"appnexus": {}},
			expected: []string{"openx", "rise"},
		},
		{
			name:     "nothing_removed_when_all_allowed",
			req:      mockRequestWrapperWithBidders(t, []string{"appnexus", "rise"}),
			allowed:  map[string]struct{}{"appnexus": {}, "rise": {}},
			expected: []string{},
		},
		{
			name:     "allowed_bidder_absent_from_request_is_ignored",
			req:      mockRequestWrapperWithBidders(t, []string{"appnexus"}),
			allowed:  map[string]struct{}{"rise": {}},
			expected: []string{"appnexus"},
		},
		{
			name:     "nil_request",
			req:      nil,
			allowed:  map[string]struct{}{"appnexus": {}},
			expected: nil,
		},
		{
			name:     "empty_allow_list_removes_all_present",
			req:      mockRequestWrapperWithBidders(t, []string{"rise", "openx"}),
			allowed:  map[string]struct{}{},
			expected: []string{"openx", "rise"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, biddersRemovedByInclude(tt.req, tt.allowed))
		})
	}
}

// TestAppendInclusionWarnings verifies the end-to-end warning emission: a warning is appended only
// when at least one include rule fired and something was actually removed.
func TestAppendInclusionWarnings(t *testing.T) {
	tests := []struct {
		name             string
		req              *openrtb_ext.RequestWrapper
		allowed          map[string]struct{}
		includeContexts  []rules.ResultFunctionMeta
		expectedWarnings []string
	}{
		{
			name:    "warns_for_bidders_removed_by_single_include_rule",
			req:     mockRequestWrapperWithBidders(t, []string{"appnexus", "rise", "openx"}),
			allowed: map[string]struct{}{"appnexus": {}},
			includeContexts: []rules.ResultFunctionMeta{
				{
					RulesetName:           "microsoft-account-rise",
					SchemaFunctionResults: []rules.SchemaFunctionStep{{FuncName: "deviceCountry", FuncResult: "JPN"}},
				},
			},
			expectedWarnings: []string{
				`Bidders [openx, rise] were removed from the request by the rules engine because they were not in the include list applied by ruleset "microsoft-account-rise" (deviceCountry rule evaluated to "JPN")`,
			},
		},
		{
			name:    "single_bidder_phrasing",
			req:     mockRequestWrapperWithBidders(t, []string{"appnexus", "openx"}),
			allowed: map[string]struct{}{"appnexus": {}},
			includeContexts: []rules.ResultFunctionMeta{
				{AnalyticsKey: "bidderConfig"},
			},
			expectedWarnings: []string{
				`Bidder [openx] was removed from the request by the rules engine because it was not in the include list applied by ruleset "bidderConfig"`,
			},
		},
		{
			name:    "multiple_include_contexts_joined",
			req:     mockRequestWrapperWithBidders(t, []string{"appnexus", "rise", "openx"}),
			allowed: map[string]struct{}{"appnexus": {}, "rise": {}},
			includeContexts: []rules.ResultFunctionMeta{
				{RulesetName: "rulesetA", SchemaFunctionResults: []rules.SchemaFunctionStep{{FuncName: "deviceCountry", FuncResult: "JPN"}}},
				{RulesetName: "rulesetB", SchemaFunctionResults: []rules.SchemaFunctionStep{{FuncName: "channel", FuncResult: "web"}}},
			},
			expectedWarnings: []string{
				`Bidder [openx] was removed from the request by the rules engine because it was not in the include list applied by ruleset "rulesetA" (deviceCountry rule evaluated to "JPN"), ruleset "rulesetB" (channel rule evaluated to "web")`,
			},
		},
		{
			name:             "no_warning_when_no_include_context",
			req:              mockRequestWrapperWithBidders(t, []string{"appnexus", "openx"}),
			allowed:          map[string]struct{}{"appnexus": {}},
			includeContexts:  nil,
			expectedWarnings: nil,
		},
		{
			name:    "no_warning_when_nothing_removed",
			req:     mockRequestWrapperWithBidders(t, []string{"appnexus"}),
			allowed: map[string]struct{}{"appnexus": {}},
			includeContexts: []rules.ResultFunctionMeta{
				{RulesetName: "rulesetA"},
			},
			expectedWarnings: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &ProcessedAuctionHookResult{
				HookResult:      hs.HookResult[hs.ProcessedAuctionRequestPayload]{ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{}},
				AllowedBidders:  tt.allowed,
				IncludeContexts: tt.includeContexts,
			}

			appendInclusionWarnings(tt.req, result)

			assert.Equal(t, tt.expectedWarnings, result.HookResult.Warnings)
		})
	}
}

func TestAppendInclusionWarningsEmitsStructuredAnalytics(t *testing.T) {
	req := mockRequestWrapperWithBidders(t, []string{"openx", "rubicon"})
	result := &ProcessedAuctionHookResult{
		HookResult: hs.HookResult[hs.ProcessedAuctionRequestPayload]{
			ChangeSet: hs.ChangeSet[hs.ProcessedAuctionRequestPayload]{},
		},
		AllowedBidders: map[string]struct{}{"rubicon": {}},
		IncludeContexts: []rules.ResultFunctionMeta{
			{
				RulesetName: "include-ruleset",
				SchemaFunctionResults: []rules.SchemaFunctionStep{
					{FuncName: "deviceCountryIn", FuncResult: "true"},
				},
			},
		},
	}

	appendInclusionWarnings(req, result)

	assert.Equal(t, []string{
		`Bidder [openx] was removed from the request by the rules engine because it was not in the include list applied by ruleset "include-ruleset" (deviceCountryIn rule evaluated to true)`,
	}, result.HookResult.Warnings)
	assert.Len(t, result.HookResult.AnalyticsTags.Activities, 1)
	activity := result.HookResult.AnalyticsTags.Activities[0]
	assert.Equal(t, rulesEngineBidderFilteringActivity, activity.Name)
	assert.Equal(t, hookanalytics.ActivityStatusSuccess, activity.Status)
	assert.Len(t, activity.Results, 1)
	assert.Equal(t, hookanalytics.ResultStatusBlock, activity.Results[0].Status)
	assert.Equal(t, "openx", activity.Results[0].AppliedTo.Bidder)
	assert.Equal(t, true, activity.Results[0].AppliedTo.Request)
	assert.Equal(t, errortypes.RulesEngineBidderNotInIncludeListWarningCode, activity.Results[0].Values["code"])
	assert.Equal(t, rulesEngineReasonNotInIncludeList, activity.Results[0].Values["reason"])
	assert.NotContains(t, activity.Results[0].Values, "warning", "human-readable warning belongs in debug output, not analytics tags")
	assert.NotEmpty(t, activity.Results[0].Values["contexts"])
}

// TestBuildInclusionWarning asserts the warning-string builder directly across single/plural bidder
// phrasing, ruleset-name-vs-analytics-key preference, missing reasons, and multiple contexts.
func TestBuildInclusionWarning(t *testing.T) {
	tests := []struct {
		name     string
		bidders  []string
		contexts []rules.ResultFunctionMeta
		expected string
	}{
		{
			name:    "single_bidder_full_context",
			bidders: []string{"openx"},
			contexts: []rules.ResultFunctionMeta{
				{RulesetName: "rulesetA", SchemaFunctionResults: []rules.SchemaFunctionStep{{FuncName: "deviceCountry", FuncResult: "JPN"}}},
			},
			expected: `Bidder [openx] was removed from the request by the rules engine because it was not in the include list applied by ruleset "rulesetA" (deviceCountry rule evaluated to "JPN")`,
		},
		{
			name:    "plural_bidders_analytics_key_fallback",
			bidders: []string{"openx", "rise"},
			contexts: []rules.ResultFunctionMeta{
				{AnalyticsKey: "bidderConfig", SchemaFunctionResults: []rules.SchemaFunctionStep{{FuncName: "deviceCountryIn", FuncResult: "true"}}},
			},
			expected: `Bidders [openx, rise] were removed from the request by the rules engine because they were not in the include list applied by ruleset "bidderConfig" (deviceCountryIn rule evaluated to true)`,
		},
		{
			name:    "ruleset_name_preferred_over_analytics_key",
			bidders: []string{"openx"},
			contexts: []rules.ResultFunctionMeta{
				{RulesetName: "cross-account-openx", AnalyticsKey: "someKey"},
			},
			expected: `Bidder [openx] was removed from the request by the rules engine because it was not in the include list applied by ruleset "cross-account-openx"`,
		},
		{
			name:     "no_context_details",
			bidders:  []string{"openx"},
			contexts: []rules.ResultFunctionMeta{{}},
			expected: `Bidder [openx] was removed from the request by the rules engine because it was not in the include list`,
		},
		{
			name:    "multiple_contexts_joined",
			bidders: []string{"openx"},
			contexts: []rules.ResultFunctionMeta{
				{RulesetName: "rulesetA", SchemaFunctionResults: []rules.SchemaFunctionStep{{FuncName: "deviceCountry", FuncResult: "JPN"}}},
				{RulesetName: "rulesetB", SchemaFunctionResults: []rules.SchemaFunctionStep{{FuncName: "channel", FuncResult: "web"}}},
			},
			expected: `Bidder [openx] was removed from the request by the rules engine because it was not in the include list applied by ruleset "rulesetA" (deviceCountry rule evaluated to "JPN"), ruleset "rulesetB" (channel rule evaluated to "web")`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, buildInclusionWarning(tt.bidders, tt.contexts))
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
