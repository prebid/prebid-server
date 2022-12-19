package hookexecution

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/stretchr/testify/assert"
)

func TestEmptyHookExecutor(t *testing.T) {
	executor := EmptyHookExecutor{}
	executor.SetAccount(&config.Account{})

	body := []byte(`{"foo": "bar"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	assert.NoError(t, err, "Failed to create http request.")

	entrypointBody, entrypointRejectErr := executor.ExecuteEntrypointStage(req, body)
	rawAuctionBody, rawAuctionRejectErr := executor.ExecuteRawAuctionStage(body)
	executor.ExecuteAuctionResponseStage(&openrtb2.BidResponse{})

	outcomes := executor.GetOutcomes()
	assert.Equal(t, EmptyHookExecutor{}, executor, "EmptyHookExecutor shouldn't be changed.")
	assert.Empty(t, outcomes, "EmptyHookExecutor shouldn't return stage outcomes.")

	assert.Nil(t, entrypointRejectErr, "EmptyHookExecutor shouldn't return reject error at entrypoint stage.")
	assert.Equal(t, body, entrypointBody, "EmptyHookExecutor shouldn't change body at entrypoint stage.")

	assert.Nil(t, rawAuctionRejectErr, "EmptyHookExecutor shouldn't return reject error at raw-auction stage.")
	assert.Equal(t, body, rawAuctionBody, "EmptyHookExecutor shouldn't change body at raw-auction stage.")

}

func TestExecuteEntrypointStage(t *testing.T) {
	const body string = `{"name": "John", "last_name": "Doe"}`
	const urlString string = "https://prebid.com/openrtb2/auction"

	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}

	testCases := []struct {
		description            string
		givenBody              string
		givenUrl               string
		givenPlanBuilder       hooks.ExecutionPlanBuilder
		expectedBody           string
		expectedHeader         http.Header
		expectedQuery          url.Values
		expectedReject         *RejectError
		expectedModuleContexts *moduleContexts
		expectedStageOutcomes  []StageOutcome
	}{
		{
			description:            "Payload not changed if hook execution plan empty",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       hooks.EmptyPlanBuilder{},
			expectedBody:           body,
			expectedHeader:         http.Header{},
			expectedQuery:          url.Values{},
			expectedReject:         nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:  []StageOutcome{},
		},
		{
			description:            "Payload changed if hooks return mutations",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       TestApplyHookMutationsBuilder{},
			expectedBody:           `{"last_name": "Doe", "foo": "bar"}`,
			expectedHeader:         http.Header{"Foo": []string{"bar"}},
			expectedQuery:          url.Values{"foo": []string{"baz"}},
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityHttpRequest,
					Stage:  hooks.StageEntrypoint.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate)},
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foobaz"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      []string{"failed to apply hook mutation: key not found"},
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: param.foo, mutation type: %s", hookstage.MutationUpdate)},
									Errors:        nil,
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "baz"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
									},
									Errors:   nil,
									Warnings: nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusFailure,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"hook execution failed: attribute not found"},
									Warnings:      nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:            "Stage execution can be rejected - and later hooks rejected",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       TestRejectPlanBuilder{},
			expectedBody:           body,
			expectedHeader:         http.Header{"Foo": []string{"bar"}},
			expectedQuery:          url.Values{},
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookCode: "bar"}, hooks.StageEntrypoint.String()},
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityHttpRequest,
					Stage:  hooks.StageEntrypoint.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "baz"},
									Status:        StatusExecutionFailure,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"unexpected error"},
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionReject,
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										`Module foobar (hook: bar) rejected request with code 0 at entrypoint stage`,
									},
									Warnings: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:            "Request can be changed when a hook times out",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       TestWithTimeoutPlanBuilder{},
			expectedBody:           `{"foo":"bar", "last_name":"Doe"}`,
			expectedHeader:         http.Header{"Foo": []string{"bar"}},
			expectedQuery:          url.Values{},
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityHttpRequest,
					Stage:  hooks.StageEntrypoint.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusTimeout,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"Hook execution timeout"},
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "baz"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
									},
									Errors:   nil,
									Warnings: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:      "Modules contexts are preserved and correct",
			givenBody:        body,
			givenUrl:         urlString,
			givenPlanBuilder: TestWithModuleContextsPlanBuilder{},
			expectedBody:     body,
			expectedHeader:   http.Header{},
			expectedQuery:    url.Values{},
			expectedReject:   nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"entrypoint-ctx-1": "some-ctx-1", "entrypoint-ctx-3": "some-ctx-3"},
				"module-2": {"entrypoint-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityHttpRequest,
					Stage:  hooks.StageEntrypoint.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-2", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookCode: "baz"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			body := []byte(test.givenBody)
			reader := bytes.NewReader(body)
			req, err := http.NewRequest(http.MethodPost, test.givenUrl, reader)
			assert.NoError(t, err)

			exec := NewHookExecutor(test.givenPlanBuilder, EndpointAuction, &metricsConfig.NilMetricsEngine{})
			newBody, reject := exec.ExecuteEntrypointStage(req, body)

			assert.Equal(t, test.expectedReject, reject, "Unexpected stage reject.")
			assert.JSONEq(t, test.expectedBody, string(newBody), "Incorrect request body.")
			assert.Equal(t, test.expectedHeader, req.Header, "Incorrect request header.")
			assert.Equal(t, test.expectedQuery, req.URL.Query(), "Incorrect request query.")
			assert.Equal(t, test.expectedModuleContexts, exec.moduleContexts, "Incorrect module contexts")

			stageOutcomes := exec.GetOutcomes()
			if len(test.expectedStageOutcomes) == 0 {
				assert.Empty(t, stageOutcomes, "Incorrect stage outcomes.")
			} else {
				assertEqualStageOutcomes(t, test.expectedStageOutcomes[0], stageOutcomes[0])
			}
		})
	}
}

func TestExecuteRawAuctionStage(t *testing.T) {
	const body string = `{"name": "John", "last_name": "Doe"}`
	const bodyUpdated string = `{"last_name": "Doe", "foo": "bar"}`
	const urlString string = "https://prebid.com/openrtb2/auction"

	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}
	account := &config.Account{}

	testCases := []struct {
		description            string
		givenBody              string
		givenUrl               string
		givenPlanBuilder       hooks.ExecutionPlanBuilder
		givenAccount           *config.Account
		expectedBody           string
		expectedReject         *RejectError
		expectedModuleContexts *moduleContexts
		expectedStageOutcomes  []StageOutcome
	}{
		{
			description:            "Payload not changed if hook execution plan empty",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       hooks.EmptyPlanBuilder{},
			givenAccount:           account,
			expectedBody:           body,
			expectedReject:         nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:  []StageOutcome{},
		},
		{
			description:            "Payload changed if hooks return mutations",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       TestApplyHookMutationsBuilder{},
			givenAccount:           account,
			expectedBody:           bodyUpdated,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuction.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
									},
									Errors:   nil,
									Warnings: nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      []string{"failed to apply hook mutation: key not found"},
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "baz"},
									Status:        StatusFailure,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"hook execution failed: attribute not found"},
									Warnings:      nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:            "Stage execution can be rejected - and later hooks rejected",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       TestRejectPlanBuilder{},
			givenAccount:           nil,
			expectedBody:           bodyUpdated,
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookCode: "bar"}, hooks.StageRawAuction.String()},
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuction.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
									},
									Errors:   nil,
									Warnings: nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "baz"},
									Status:        StatusExecutionFailure,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"unexpected error"},
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionReject,
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										`Module foobar (hook: bar) rejected request with code 0 at raw-auction stage`,
									},
									Warnings: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:            "Request can be changed when a hook times out",
			givenBody:              body,
			givenUrl:               urlString,
			givenPlanBuilder:       TestWithTimeoutPlanBuilder{},
			givenAccount:           account,
			expectedBody:           bodyUpdated,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuction.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.foo, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: body.name, mutation type: %s", hookstage.MutationDelete),
									},
									Errors:   nil,
									Warnings: nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusTimeout,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"Hook execution timeout"},
									Warnings:      nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:      "Modules contexts are preserved and correct",
			givenBody:        body,
			givenUrl:         urlString,
			givenPlanBuilder: TestWithModuleContextsPlanBuilder{},
			givenAccount:     account,
			expectedBody:     body,
			expectedReject:   nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"raw-auction-ctx-1": "some-ctx-1", "raw-auction-ctx-3": "some-ctx-3"},
				"module-2": {"raw-auction-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuction.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-2", HookCode: "baz"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			exec := NewHookExecutor(test.givenPlanBuilder, EndpointAuction, &metricsConfig.NilMetricsEngine{})
			exec.SetAccount(test.givenAccount)

			newBody, reject := exec.ExecuteRawAuctionStage([]byte(test.givenBody))

			assert.Equal(t, test.expectedReject, reject, "Unexpected stage reject.")
			assert.JSONEq(t, test.expectedBody, string(newBody), "Incorrect request body.")
			assert.Equal(t, test.expectedModuleContexts, exec.moduleContexts, "Incorrect module contexts")

			stageOutcomes := exec.GetOutcomes()
			if len(test.expectedStageOutcomes) == 0 {
				assert.Empty(t, stageOutcomes, "Incorrect stage outcomes.")
			} else {
				assertEqualStageOutcomes(t, test.expectedStageOutcomes[0], stageOutcomes[0])
			}
		})
	}
}

func TestExecuteAuctionResponseStage(t *testing.T) {
	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}
	account := &config.Account{}
	resp := &openrtb2.BidResponse{CustomData: "some-custom-data"}
	expResp := &openrtb2.BidResponse{CustomData: "new-custom-data"}

	testCases := []struct {
		description            string
		givenPlanBuilder       hooks.ExecutionPlanBuilder
		givenAccount           *config.Account
		givenResponse          *openrtb2.BidResponse
		expectedResponse       *openrtb2.BidResponse
		expectedReject         *RejectError
		expectedModuleContexts *moduleContexts
		expectedStageOutcomes  []StageOutcome
	}{
		{
			description:            "Payload not changed if hook execution plan empty",
			givenPlanBuilder:       hooks.EmptyPlanBuilder{},
			givenAccount:           account,
			givenResponse:          resp,
			expectedResponse:       resp,
			expectedReject:         nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:  []StageOutcome{},
		},
		{
			description:            "Payload changed if hooks return mutations",
			givenPlanBuilder:       TestApplyHookMutationsBuilder{},
			givenAccount:           account,
			givenResponse:          resp,
			expectedResponse:       expResp,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionResponse,
					Stage:  hooks.StageAuctionResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: auctionResponse.bidResponse.custom-data, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:            "Stage execution can't be rejected - stage doesn't support rejection",
			givenPlanBuilder:       TestRejectPlanBuilder{},
			givenAccount:           nil,
			givenResponse:          resp,
			expectedResponse:       expResp,
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookCode: "foo"}, hooks.StageAuctionResponse.String()},
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionResponse,
					Stage:  hooks.StageAuctionResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "baz"},
									Status:        StatusExecutionFailure,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"unexpected error"},
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusExecutionFailure,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										fmt.Sprintf("Module (name: foobar, hook code: foo) tried to reject request on the %s stage that does not support rejection", hooks.StageAuctionResponse),
									},
									Warnings: nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: auctionResponse.bidResponse.custom-data, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:            "Request can be changed when a hook times out",
			givenPlanBuilder:       TestWithTimeoutPlanBuilder{},
			givenAccount:           account,
			givenResponse:          resp,
			expectedResponse:       expResp,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionResponse,
					Stage:  hooks.StageAuctionResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "foo"},
									Status:        StatusTimeout,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors:        []string{"Hook execution timeout"},
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: auctionResponse.bidResponse.custom-data, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
							},
						},
					},
				},
			},
		},
		{
			description:      "Modules contexts are preserved and correct",
			givenPlanBuilder: TestWithModuleContextsPlanBuilder{},
			givenAccount:     account,
			givenResponse:    resp,
			expectedResponse: resp,
			expectedReject:   nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"auction-response-ctx-1": "some-ctx-1", "auction-response-ctx-3": "some-ctx-3"},
				"module-2": {"auction-response-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionResponse,
					Stage:  hooks.StageAuctionResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-2", HookCode: "baz"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			exec := NewHookExecutor(test.givenPlanBuilder, EndpointAuction, &metricsConfig.NilMetricsEngine{})
			exec.SetAccount(test.givenAccount)

			exec.ExecuteAuctionResponseStage(test.givenResponse)

			assert.Equal(t, test.expectedResponse, test.givenResponse, "Incorrect response update.")
			assert.Equal(t, test.expectedModuleContexts, exec.moduleContexts, "Incorrect module contexts")

			stageOutcomes := exec.GetOutcomes()
			if len(test.expectedStageOutcomes) == 0 {
				assert.Empty(t, stageOutcomes, "Incorrect stage outcomes.")
			} else {
				assertEqualStageOutcomes(t, test.expectedStageOutcomes[0], stageOutcomes[0])
			}
		})
	}
}

func TestInterStageContextCommunication(t *testing.T) {
	body := []byte(`{"foo": "bar"}`)
	reader := bytes.NewReader(body)
	exec := NewHookExecutor(TestWithModuleContextsPlanBuilder{}, EndpointAuction, &metricsConfig.NilMetricsEngine{})
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	assert.NoError(t, err)

	// test that context added at the entrypoint stage
	_, reject := exec.ExecuteEntrypointStage(req, body)
	assert.Nil(t, reject, "Unexpected reject from entrypoint stage.")
	assert.Equal(
		t,
		&moduleContexts{ctxs: map[string]hookstage.ModuleContext{
			"module-1": {
				"entrypoint-ctx-1": "some-ctx-1",
				"entrypoint-ctx-3": "some-ctx-3",
			},
			"module-2": {"entrypoint-ctx-2": "some-ctx-2"},
		}},
		exec.moduleContexts,
		"Wrong module contexts after executing entrypoint hook.",
	)

	// test that context added at the raw-auction stage merged with existing module contexts
	_, reject = exec.ExecuteRawAuctionStage(body)
	assert.Nil(t, reject, "Unexpected reject from raw-auction stage.")
	assert.Equal(t, &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
		"module-1": {
			"entrypoint-ctx-1":  "some-ctx-1",
			"entrypoint-ctx-3":  "some-ctx-3",
			"raw-auction-ctx-1": "some-ctx-1",
			"raw-auction-ctx-3": "some-ctx-3",
		},
		"module-2": {
			"entrypoint-ctx-2":  "some-ctx-2",
			"raw-auction-ctx-2": "some-ctx-2",
		},
	}}, exec.moduleContexts, "Wrong module contexts after executing raw-auction hook.")

	// test that context added at the auction-response stage merged with existing module contexts
	exec.ExecuteAuctionResponseStage(&openrtb2.BidResponse{})
	assert.Nil(t, reject, "Unexpected reject from raw-auction stage.")
	assert.Equal(t, &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
		"module-1": {
			"entrypoint-ctx-1":       "some-ctx-1",
			"entrypoint-ctx-3":       "some-ctx-3",
			"raw-auction-ctx-1":      "some-ctx-1",
			"raw-auction-ctx-3":      "some-ctx-3",
			"auction-response-ctx-1": "some-ctx-1",
			"auction-response-ctx-3": "some-ctx-3",
		},
		"module-2": {
			"entrypoint-ctx-2":       "some-ctx-2",
			"raw-auction-ctx-2":      "some-ctx-2",
			"auction-response-ctx-2": "some-ctx-2",
		},
	}}, exec.moduleContexts, "Wrong module contexts after executing auction-response hook.")
}

type TestApplyHookMutationsBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestApplyHookMutationsBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "foobaz", Hook: mockFailedMutationHook{}},
				{Module: "foobar", Code: "bar", Hook: mockUpdateQueryEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "baz", Hook: mockUpdateBodyHook{}},
				{Module: "foobar", Code: "foo", Hook: mockFailureHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuctionRequest] {
	return hooks.Plan[hookstage.RawAuctionRequest]{
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
				{Module: "foobar", Code: "bar", Hook: mockFailedMutationHook{}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "baz", Hook: mockFailureHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForAuctionResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.AuctionResponse] {
	return hooks.Plan[hookstage.AuctionResponse]{
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBidResponseHook{}},
			},
		},
	}
}

type TestRejectPlanBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestRejectPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 5 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				// reject stage
				{Module: "foobar", Code: "bar", Hook: mockRejectHook{}},
				// next hook rejected: we use timeout hook to make sure
				// that it runs longer than previous one, so it won't be executed earlier
				{Module: "foobar", Code: "baz", Hook: mockTimeoutHook{}},
			},
		},
		// group of hooks rejected
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuctionRequest] {
	return hooks.Plan[hookstage.RawAuctionRequest]{
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "bar", Hook: mockRejectHook{}},
				// next hook rejected: we use timeout hook to make sure
				// that it runs longer than previous one, so it won't be executed earlier
				{Module: "foobar", Code: "baz", Hook: mockTimeoutHook{}},
			},
		},
		// group of hooks rejected
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForAuctionResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.AuctionResponse] {
	return hooks.Plan[hookstage.AuctionResponse]{
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
		// rejection ignored, stage doesn't support rejection
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "foobar", Code: "foo", Hook: mockRejectHook{}},
			},
		},
		// hook executed and payload updated because this stage doesn't support rejection
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidResponseHook{}},
			},
		},
	}
}

type TestWithTimeoutPlanBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestWithTimeoutPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "bar", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "baz", Hook: mockUpdateBodyHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuctionRequest] {
	return hooks.Plan[hookstage.RawAuctionRequest]{
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "bar", Hook: mockTimeoutHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForAuctionResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.AuctionResponse] {
	return hooks.Plan[hookstage.AuctionResponse]{
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "foobar", Code: "foo", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidResponseHook{}},
			},
		},
	}
}

type TestWithModuleContextsPlanBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestWithModuleContextsPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "entrypoint-ctx-1", val: "some-ctx-1"}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextHook{key: "entrypoint-ctx-2", val: "some-ctx-2"}},
				{Module: "module-1", Code: "baz", Hook: mockModuleContextHook{key: "entrypoint-ctx-3", val: "some-ctx-3"}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuctionRequest] {
	return hooks.Plan[hookstage.RawAuctionRequest]{
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "raw-auction-ctx-1", val: "some-ctx-1"}},
				{Module: "module-2", Code: "baz", Hook: mockModuleContextHook{key: "raw-auction-ctx-2", val: "some-ctx-2"}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "module-1", Code: "bar", Hook: mockModuleContextHook{key: "raw-auction-ctx-3", val: "some-ctx-3"}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForAuctionResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.AuctionResponse] {
	return hooks.Plan[hookstage.AuctionResponse]{
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "auction-response-ctx-1", val: "some-ctx-1"}},
				{Module: "module-2", Code: "baz", Hook: mockModuleContextHook{key: "auction-response-ctx-2", val: "some-ctx-2"}},
			},
		},
		hooks.Group[hookstage.AuctionResponse]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AuctionResponse]{
				{Module: "module-1", Code: "bar", Hook: mockModuleContextHook{key: "auction-response-ctx-3", val: "some-ctx-3"}},
			},
		},
	}
}
