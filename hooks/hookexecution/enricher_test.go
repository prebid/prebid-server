package hookexecution

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// StageOutcomeTest is used for test purpose instead of the original structure,
// so we can unmarshal hidden fields such as StageOutcomeTest.Stage and HookOutcomeTest.Errors/Warnings
type StageOutcomeTest struct {
	ExecutionTime
	Entity hookstage.Entity   `json:"entity"`
	Groups []GroupOutcomeTest `json:"groups"`
	Stage  string             `json:"stage"`
}

type GroupOutcomeTest struct {
	ExecutionTime
	InvocationResults []HookOutcomeTest `json:"invocationresults"`
}

type HookOutcomeTest struct {
	ExecutionTime
	AnalyticsTags hookanalytics.Analytics `json:"analyticstags"`
	HookID        HookID                  `json:"hookid"`
	Status        Status                  `json:"status"`
	Action        Action                  `json:"action"`
	Message       string                  `json:"message"`
	DebugMessages []string                `json:"debugmessages"`
	Errors        []string                `json:"errors"`
	Warnings      []string                `json:"warnings"`
}

func TestEnrichResponse(t *testing.T) {
	testCases := []struct {
		description             string
		expectedBidResponseFile string
		stageOutcomesFile       string
		bidResponse             *openrtb2.BidResponse
		bidRequest              *openrtb2.BidRequest
		account                 *config.Account
	}{
		{
			description:             "BidResponse enriched with verbose trace and debug info when bidRequest.test=1 and trace=verbose",
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-verbose-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: []byte(`{"prebid": {"trace": "verbose"}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse enriched with basic trace and debug info when bidRequest.ext.prebid.debug=true and trace=basic",
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-basic-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Ext: []byte(`{"prebid": {"debug": true, "trace": "basic"}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse enriched with debug info when bidRequest.ext.prebid.debug=true",
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Ext: []byte(`{"prebid": {"debug": true, "trace": ""}}`)},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse not enriched when bidRequest.ext.prebid.debug=false",
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-empty-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse enriched only with verbose trace when bidRequest.ext.prebid.trace=verbose and account.DebugAllow=false",
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-verbose-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{Ext: []byte(`{"prebid": {"foo": "bar"}}`)},
			bidRequest:              &openrtb2.BidRequest{Test: 1, Ext: []byte(`{"prebid": {"debug": true, "trace": "verbose"}}`)},
			account:                 &config.Account{DebugAllow: false},
		},
		{
			description:             "BidResponse enriched with debug info if bidResponse.Ext is nil",
			expectedBidResponseFile: "test/complete-stage-outcomes/expected-pure-debug-response.json",
			stageOutcomesFile:       "test/complete-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{},
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
		{
			description:             "BidResponse enriched with empty ModulesExt if stage outcomes empty",
			expectedBidResponseFile: "test/empty-stage-outcomes/expected-response.json",
			stageOutcomesFile:       "test/empty-stage-outcomes/stage-outcomes.json",
			bidResponse:             &openrtb2.BidResponse{},
			bidRequest:              &openrtb2.BidRequest{Test: 1},
			account:                 &config.Account{DebugAllow: true},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			expectedResponse := readFile(t, test.expectedBidResponseFile)
			stageOutcomes := getStageOutcomes(t, test.stageOutcomesFile)

			err := EnrichResponse(test.bidResponse, stageOutcomes, test.bidRequest, test.account)
			require.NoError(t, err, "Failed to enrich BidResponse with hook debug information: %s", err)
			assert.JSONEq(t, string(expectedResponse), string(test.bidResponse.Ext))
		})
	}
}

func getStageOutcomes(t *testing.T, file string) []StageOutcome {
	var stageOutcomes []StageOutcome
	var stageOutcomesTest []StageOutcomeTest

	data := readFile(t, file)
	err := json.Unmarshal(data, &stageOutcomesTest)
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
				hook := HookOutcome(hookT)
				group.InvocationResults = append(group.InvocationResults, &hook)
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
