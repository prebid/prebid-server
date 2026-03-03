package hookexecution

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StageOutcomeTest is used for test purpose instead of the original structure,
// so we can unmarshal hidden fields such as StageOutcomeTest.Stage and HookOutcomeTest.Errors/Warnings
type StageOutcomeTest struct {
	ExecutionTime
	Entity entity             `json:"entity"`
	Groups []GroupOutcomeTest `json:"groups"`
	Stage  string             `json:"stage"`
}

type GroupOutcomeTest struct {
	ExecutionTime
	InvocationResults []HookOutcomeTest `json:"invocation_results"`
}

type HookOutcomeTest struct {
	ExecutionTime
	AnalyticsTags hookanalytics.Analytics `json:"analytics_tags"`
	HookID        HookID                  `json:"hook_id"`
	Status        Status                  `json:"status"`
	Action        Action                  `json:"action"`
	Message       string                  `json:"message"`
	DebugMessages []string                `json:"debug_messages"`
	Errors        []string                `json:"errors"`
	Warnings      []string                `json:"warnings"`
}

func TestEnrichBidResponse(t *testing.T) {
	testCases := []struct {
		description             string
		expectedWarnings        []error
		expectedBidResponseFile string
		stageOutcomesFile       string
		bidResponse             *openrtb2.BidResponse
		bidRequest              *openrtb2.BidRequest
		account                 *config.Account
	}{
		{
			description:             "BidResponse enriched with verbose trace and debug info when bidRequest.test=1 and trace=verbose",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-verbose-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: []byte(`{"prebid": {"trace": "verbose"}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse enriched with basic trace and debug info when bidRequest.ext.prebid.debug=true and trace=basic",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-basic-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Ext: []byte(`{"prebid": {"debug": true, "trace": "basic"}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse enriched with debug info when bidRequest.ext.prebid.debug=true",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Ext: []byte(`{"prebid": {"debug": true, "trace": ""}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse not enriched when bidRequest.ext.prebid.debug=false",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-empty-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse enriched only with verbose trace when bidRequest.ext.prebid.trace=verbose and account.DebugAllow=false",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-verbose-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: []byte(`{"prebid": {"debug": true, "trace": "verbose"}}`)},
			account:                 &config.Account{DebugAllow: false},
		},
		{
			description:             "BidResponse enriched with debug info if bidResponse.Ext is nil",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-pure-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{},
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse not enriched with modules if stage outcome groups empty",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/empty-stage-outcomes/empty.json",
			stageOutcomesFile:       "test/empty-stage-outcomes/empty-stage-outcomes-v1.json",
			bidResponse:             &openrtb2.BidResponse{},
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse not enriched with modules if stage outcomes empty",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/empty-stage-outcomes/empty.json",
			stageOutcomesFile:       "test/empty-stage-outcomes/empty-stage-outcomes-v2.json",
			bidResponse:             &openrtb2.BidResponse{},
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			expectedResponse := readFile(t, test.expectedBidResponseFile)
			stageOutcomes := getStageOutcomes(t, test.stageOutcomesFile)

			ext, warns, err := EnrichExtBidResponse(test.bidResponse.Ext, stageOutcomes, test.bidRequest, test.account)
			require.NoError(t, err, "Failed to enrich BidResponse with hook debug information: %s", err)
			assert.Equal(t, test.expectedWarnings, warns, "Unexpected warnings")

			test.bidResponse.Ext = ext
			if test.bidResponse.Ext == nil {
				assert.Empty(t, expectedResponse)
			} else {
				assert.JSONEq(t, string(expectedResponse), string(test.bidResponse.Ext))
			}
		})
	}
}

func TestGetModulesJSON(t *testing.T) {
	testCases := []struct {
		description             string
		expectedWarnings        []error
		expectedBidResponseFile string
		stageOutcomesFile       string
		bidRequest              *openrtb2.BidRequest
		account                 *config.Account
	}{
		{
			description:             "Modules Outcome contains verbose trace and debug info when bidRequest.test=1 and trace=verbose",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-verbose-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: []byte(`{"prebid": {"trace": "verbose"}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "Modules Outcome contains verbose trace and debug info when bidRequest.test=1 and trace=verbose and account is not defined",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-verbose-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: []byte(`{"prebid": {"trace": "verbose"}}`)},
			account:                 nil,
		},
		{
			description:             "Modules Outcome contains basic trace and debug info when bidRequest.ext.prebid.debug=true and trace=basic",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-basic-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{Ext: []byte(`{"prebid": {"debug": true, "trace": "basic"}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "Modules Outcome contains debug info when bidRequest.ext.prebid.debug=true",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{Ext: []byte(`{"prebid": {"debug": true, "trace": ""}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "Modules Outcome empty when bidRequest.ext.prebid.debug=false",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/empty-stage-outcomes/empty.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "Modules Outcome empty when bidRequest is nil",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/empty-stage-outcomes/empty.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              nil,
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "Modules Outcome contains only verbose trace when bidRequest.ext.prebid.trace=verbose and account.DebugAllow=false",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-verbose-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: []byte(`{"prebid": {"debug": true, "trace": "verbose"}}`)},
			account:                 &config.Account{DebugAllow: false},
		},
		{
			description:             "Modules Outcome contains debug info if bidResponse.Ext is nil",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-pure-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "Modules Outcome empty if stage outcome groups empty",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/empty-stage-outcomes/empty.json",
			stageOutcomesFile:       "test/empty-stage-outcomes/empty-stage-outcomes-v1.json",
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "Modules Outcome empty if stage outcomes empty",
			expectedWarnings:        nil,
			expectedBidResponseFile: "test/empty-stage-outcomes/empty.json",
			stageOutcomesFile:       "test/empty-stage-outcomes/empty-stage-outcomes-v2.json",
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description: "Warnings returned if debug info invalid",
			expectedWarnings: []error{
				errors.New("Value is not a string: 1"),
				errors.New("Value is not a boolean: active"),
			},
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-pure-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: json.RawMessage(`{"prebid": {"debug": "active", "trace": 1}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			expectedResponse := readFile(t, test.expectedBidResponseFile)
			stageOutcomes := getStageOutcomes(t, test.stageOutcomesFile)

			modules, warns, err := GetModulesJSON(stageOutcomes, test.bidRequest, test.account)
			require.NoError(t, err, "Failed to get modules outcome as json: %s", err)
			assert.Equal(t, test.expectedWarnings, warns, "Unexpected warnings")

			if modules == nil {
				assert.Empty(t, expectedResponse)
			} else {
				var expectedExtBidResponse openrtb_ext.ExtBidResponse
				err := jsonutil.UnmarshalValid(expectedResponse, &expectedExtBidResponse)
				assert.NoError(t, err, "Failed to unmarshal prebid response extension")
				assert.JSONEq(t, string(expectedExtBidResponse.Prebid.Modules), string(modules))
			}
		})
	}
}

func getStageOutcomes(t *testing.T, file string) []StageOutcome {
	var stageOutcomes []StageOutcome
	var stageOutcomesTest []StageOutcomeTest

	data := readFile(t, file)
	err := jsonutil.UnmarshalValid(data, &stageOutcomesTest)
	require.NoError(t, err, "Failed to unmarshal stage outcomes: %s", err)

	for _, stageT := range stageOutcomesTest {
		stage := StageOutcome{
			ExecutionTime: stageT.ExecutionTime,
			Entity:        stageT.Entity,
			Stage:         stageT.Stage,
		}

		for _, groupT := range stageT.Groups {
			group := GroupOutcome{ExecutionTime: groupT.ExecutionTime}
			for _, hookT := range groupT.InvocationResults {
				group.InvocationResults = append(group.InvocationResults, HookOutcome(hookT))
			}

			stage.Groups = append(stage.Groups, group)
		}
		stageOutcomes = append(stageOutcomes, stage)
	}

	return stageOutcomes
}

func readFile(t *testing.T, filename string) []byte {
	data, err := os.ReadFile(filename)
	require.NoError(t, err, "Failed to read file %s: %v", filename, err)
	return data
}
