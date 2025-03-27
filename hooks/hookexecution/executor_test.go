package hookexecution

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/exchange/entities"
	"github.com/prebid/prebid-server/v3/hooks"
	"github.com/prebid/prebid-server/v3/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/metrics"
	metricsConfig "github.com/prebid/prebid-server/v3/metrics/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/privacy"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmptyHookExecutor(t *testing.T) {
	executor := EmptyHookExecutor{}

	body := []byte(`{"foo": "bar"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	assert.NoError(t, err, "Failed to create http request.")

	bidderRequest := &openrtb2.BidRequest{ID: "some-id"}
	expectedBidderRequest := &openrtb2.BidRequest{ID: "some-id"}

	entrypointBody, entrypointRejectErr := executor.ExecuteEntrypointStage(req, body)
	rawAuctionBody, rawAuctionRejectErr := executor.ExecuteRawAuctionStage(body)
	processedAuctionRejectErr := executor.ExecuteProcessedAuctionStage(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}})
	bidderRequestRejectErr := executor.ExecuteBidderRequestStage(&openrtb_ext.RequestWrapper{BidRequest: bidderRequest}, "bidder-name")
	executor.ExecuteAuctionResponseStage(&openrtb2.BidResponse{})

	outcomes := executor.GetOutcomes()
	assert.Equal(t, EmptyHookExecutor{}, executor, "EmptyHookExecutor shouldn't be changed.")
	assert.Empty(t, outcomes, "EmptyHookExecutor shouldn't return stage outcomes.")

	assert.Nil(t, entrypointRejectErr, "EmptyHookExecutor shouldn't return reject error at entrypoint stage.")
	assert.Equal(t, body, entrypointBody, "EmptyHookExecutor shouldn't change body at entrypoint stage.")

	assert.Nil(t, rawAuctionRejectErr, "EmptyHookExecutor shouldn't return reject error at raw-auction stage.")
	assert.Equal(t, body, rawAuctionBody, "EmptyHookExecutor shouldn't change body at raw-auction stage.")

	assert.Nil(t, processedAuctionRejectErr, "EmptyHookExecutor shouldn't return reject error at processed-auction stage.")
	assert.Nil(t, bidderRequestRejectErr, "EmptyHookExecutor shouldn't return reject error at bidder-request stage.")
	assert.Equal(t, expectedBidderRequest, bidderRequest, "EmptyHookExecutor shouldn't change payload at bidder-request stage.")
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate)},
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foobaz"},
									Status:        StatusExecutionFailure,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      []string{"failed to apply hook mutation: key not found"},
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookImplCode: "bar"}, hooks.StageEntrypoint.String()},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "module-2", HookImplCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "baz"},
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

func TestMetricsAreGatheredDuringHookExecution(t *testing.T) {
	reader := bytes.NewReader(nil)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	assert.NoError(t, err)

	metricEngine := &metrics.MetricsEngineMock{}
	builder := TestAllHookResultsBuilder{}
	exec := NewHookExecutor(TestAllHookResultsBuilder{}, "/openrtb2/auction", metricEngine)
	moduleName := "module.x-1"
	moduleLabels := metrics.ModuleLabels{
		Module: moduleReplacer.Replace(moduleName),
		Stage:  "entrypoint",
	}
	rTime := func(dur time.Duration) bool { return dur.Nanoseconds() > 0 }
	plan := builder.PlanForEntrypointStage("")
	hooksCalledDuringStage := 0
	for _, group := range plan {
		for range group.Hooks {
			hooksCalledDuringStage++
		}
	}
	metricEngine.On("RecordModuleCalled", moduleLabels, mock.MatchedBy(rTime)).Times(hooksCalledDuringStage)
	metricEngine.On("RecordModuleSuccessUpdated", moduleLabels).Once()
	metricEngine.On("RecordModuleSuccessRejected", moduleLabels).Once()
	metricEngine.On("RecordModuleTimeout", moduleLabels).Once()
	metricEngine.On("RecordModuleExecutionError", moduleLabels).Twice()
	metricEngine.On("RecordModuleFailed", moduleLabels).Once()
	metricEngine.On("RecordModuleSuccessNooped", moduleLabels).Once()

	_, _ = exec.ExecuteEntrypointStage(req, nil)

	// Assert that all module metrics funcs were called with the parameters we expected
	metricEngine.AssertExpectations(t)
}

func TestExecuteRawAuctionStage(t *testing.T) {
	const body string = `{"name": "John", "last_name": "Doe"}`
	const bodyUpdated string = `{"last_name": "Doe", "foo": "bar"}`
	const urlString string = "https://prebid.com/openrtb2/auction"

	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}

	testCases := []struct {
		description            string
		givenBody              string
		givenUrl               string
		givenPlanBuilder       hooks.ExecutionPlanBuilder
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
			expectedBody:           bodyUpdated,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusExecutionFailure,
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
			expectedBody:           bodyUpdated,
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookImplCode: "bar"}, hooks.StageRawAuctionRequest.String()},
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionReject,
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										`Module foobar (hook: bar) rejected request with code 0 at raw_auction_request stage`,
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
			expectedBody:           bodyUpdated,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
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
			expectedBody:     body,
			expectedReject:   nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"raw-auction-ctx-1": "some-ctx-1", "raw-auction-ctx-3": "some-ctx-3"},
				"module-2": {"raw-auction-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageRawAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-2", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "bar"},
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

			privacyConfig := getModuleActivities("foo", false, false)
			ac := privacy.NewActivityControl(privacyConfig)
			exec.SetActivityControl(ac)

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

func TestExecuteProcessedAuctionStage(t *testing.T) {
	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}
	req := openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id"}}
	reqUpdated := openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id", Yob: 2000, Consent: "true"}}

	testCases := []struct {
		description            string
		givenPlanBuilder       hooks.ExecutionPlanBuilder
		givenRequest           openrtb_ext.RequestWrapper
		expectedRequest        openrtb2.BidRequest
		expectedErr            error
		expectedModuleContexts *moduleContexts
		expectedStageOutcomes  []StageOutcome
	}{
		{
			description:            "Request not changed if hook execution plan empty",
			givenPlanBuilder:       hooks.EmptyPlanBuilder{},
			givenRequest:           openrtb_ext.RequestWrapper{BidRequest: &req},
			expectedRequest:        req,
			expectedErr:            nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:  []StageOutcome{},
		},
		{
			description:            "Request changed if hooks return mutations",
			givenPlanBuilder:       TestApplyHookMutationsBuilder{},
			givenRequest:           openrtb_ext.RequestWrapper{BidRequest: &req},
			expectedRequest:        reqUpdated,
			expectedErr:            nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageProcessedAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.yob, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.consent, mutation type: %s", hookstage.MutationUpdate),
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
			description:            "Stage execution can be rejected - and later hooks rejected",
			givenPlanBuilder:       TestRejectPlanBuilder{},
			givenRequest:           openrtb_ext.RequestWrapper{BidRequest: &req},
			expectedRequest:        req,
			expectedErr:            &RejectError{0, HookID{ModuleCode: "foobar", HookImplCode: "foo"}, hooks.StageProcessedAuctionRequest.String()},
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageProcessedAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionReject,
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										`Module foobar (hook: foo) rejected request with code 0 at processed_auction_request stage`,
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
			givenPlanBuilder:       TestWithTimeoutPlanBuilder{},
			givenRequest:           openrtb_ext.RequestWrapper{BidRequest: &req},
			expectedRequest:        reqUpdated,
			expectedErr:            nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageProcessedAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.yob, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.consent, mutation type: %s", hookstage.MutationUpdate),
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
			givenRequest:     openrtb_ext.RequestWrapper{BidRequest: &req},
			expectedRequest:  req,
			expectedErr:      nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"processed-auction-ctx-1": "some-ctx-1", "processed-auction-ctx-3": "some-ctx-3"},
				"module-2": {"processed-auction-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAuctionRequest,
					Stage:  hooks.StageProcessedAuctionRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "module-2", HookImplCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "baz"},
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
		t.Run(test.description, func(ti *testing.T) {
			exec := NewHookExecutor(test.givenPlanBuilder, EndpointAuction, &metricsConfig.NilMetricsEngine{})

			privacyConfig := getModuleActivities("foo", false, false)
			ac := privacy.NewActivityControl(privacyConfig)
			exec.SetActivityControl(ac)

			err := exec.ExecuteProcessedAuctionStage(&test.givenRequest)

			assert.Equal(ti, test.expectedErr, err, "Unexpected stage reject.")
			assert.Equal(ti, test.expectedRequest, *test.givenRequest.BidRequest, "Incorrect request update.")
			assert.Equal(ti, test.expectedModuleContexts, exec.moduleContexts, "Incorrect module contexts")

			stageOutcomes := exec.GetOutcomes()
			if len(test.expectedStageOutcomes) == 0 {
				assert.Empty(ti, stageOutcomes, "Incorrect stage outcomes.")
			} else {
				assertEqualStageOutcomes(ti, test.expectedStageOutcomes[0], stageOutcomes[0])
			}
		})
	}
}

func TestExecuteBidderRequestStage(t *testing.T) {
	bidderName := "the-bidder"
	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}

	expectedBidderRequest := &openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id"}}
	expectedUpdatedBidderRequest := &openrtb2.BidRequest{
		ID: "some-id",
		User: &openrtb2.User{
			ID:      "user-id",
			Yob:     2000,
			Consent: "true",
		},
	}

	testCases := []struct {
		description            string
		givenBidderRequest     *openrtb2.BidRequest
		givenPlanBuilder       hooks.ExecutionPlanBuilder
		expectedBidderRequest  *openrtb2.BidRequest
		expectedReject         *RejectError
		expectedModuleContexts *moduleContexts
		expectedStageOutcomes  []StageOutcome
		privacyConfig          *config.AccountPrivacy
	}{
		{
			description:            "Payload not changed if hook execution plan empty",
			givenBidderRequest:     &openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id"}},
			givenPlanBuilder:       hooks.EmptyPlanBuilder{},
			expectedBidderRequest:  expectedBidderRequest,
			expectedReject:         nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:  []StageOutcome{},
		},
		{
			description:            "Payload changed if hooks return mutations",
			givenBidderRequest:     &openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id"}},
			givenPlanBuilder:       TestApplyHookMutationsBuilder{},
			expectedBidderRequest:  expectedUpdatedBidderRequest,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entity(bidderName),
					Stage:  hooks.StageBidderRequest.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.yob, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.consent, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusExecutionFailure,
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
			givenBidderRequest:     &openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id"}},
			givenPlanBuilder:       TestRejectPlanBuilder{},
			expectedBidderRequest:  expectedBidderRequest,
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookImplCode: "foo"}, hooks.StageBidderRequest.String()},
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					ExecutionTime: ExecutionTime{},
					Entity:        entity(bidderName),
					Stage:         hooks.StageBidderRequest.String(),
					Groups: []GroupOutcome{
						{
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionReject,
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										`Module foobar (hook: foo) rejected request with code 0 at bidder_request stage`,
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
			description:            "Stage execution can be timed out",
			givenBidderRequest:     &openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id"}},
			givenPlanBuilder:       TestWithTimeoutPlanBuilder{},
			expectedBidderRequest:  expectedUpdatedBidderRequest,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					ExecutionTime: ExecutionTime{},
					Entity:        entity(bidderName),
					Stage:         hooks.StageBidderRequest.String(),
					Groups: []GroupOutcome{
						{
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.yob, mutation type: %s", hookstage.MutationUpdate),
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidRequest.user.consent, mutation type: %s", hookstage.MutationUpdate),
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
			description:           "Modules contexts are preserved and correct",
			givenBidderRequest:    &openrtb2.BidRequest{ID: "some-id", User: &openrtb2.User{ID: "user-id"}},
			givenPlanBuilder:      TestWithModuleContextsPlanBuilder{},
			expectedBidderRequest: expectedBidderRequest,
			expectedReject:        nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"bidder-request-ctx-1": "some-ctx-1"},
				"module-2": {"bidder-request-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					ExecutionTime: ExecutionTime{},
					Entity:        entity(bidderName),
					Stage:         hooks.StageBidderRequest.String(),
					Groups: []GroupOutcome{
						{
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "foo"},
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
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-2", HookImplCode: "bar"},
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
			privacyConfig := getModuleActivities("foo", false, false)
			ac := privacy.NewActivityControl(privacyConfig)
			exec.SetActivityControl(ac)

			reject := exec.ExecuteBidderRequestStage(&openrtb_ext.RequestWrapper{BidRequest: test.givenBidderRequest}, bidderName)

			assert.Equal(t, test.expectedReject, reject, "Unexpected stage reject.")
			assert.Equal(t, test.expectedBidderRequest, test.givenBidderRequest, "Incorrect bidder request.")
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

func getModuleActivities(componentName string, allowTransmitUserFPD, allowTransmitPreciseGeo bool) *config.AccountPrivacy {
	return &config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			TransmitUserFPD:    buildDefaultActivityConfig(componentName, allowTransmitUserFPD),
			TransmitPreciseGeo: buildDefaultActivityConfig(componentName, allowTransmitPreciseGeo),
		},
	}
}

func getTransmitUFPDActivityConfig(componentName string, allow bool) *config.AccountPrivacy {
	return &config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			TransmitUserFPD: buildDefaultActivityConfig(componentName, allow),
		},
	}
}

func getTransmitPreciseGeoActivityConfig(componentName string, allow bool) *config.AccountPrivacy {
	return &config.AccountPrivacy{
		AllowActivities: &config.AllowActivities{
			TransmitPreciseGeo: buildDefaultActivityConfig(componentName, allow),
		},
	}
}

func buildDefaultActivityConfig(componentName string, allow bool) config.Activity {
	return config.Activity{
		Default: ptrutil.ToPtr(true),
		Rules: []config.ActivityRule{
			{
				Allow: allow,
				Condition: config.ActivityCondition{
					ComponentName: []string{componentName},
					ComponentType: []string{"general"},
				},
			},
		},
	}
}

func TestExecuteRawBidderResponseStage(t *testing.T) {
	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}
	resp := adapters.BidderResponse{Bids: []*adapters.TypedBid{{DealPriority: 1}}}
	expResp := adapters.BidderResponse{Bids: []*adapters.TypedBid{{DealPriority: 10}}}
	vEntity := entity("the-bidder")

	testCases := []struct {
		description            string
		givenPlanBuilder       hooks.ExecutionPlanBuilder
		givenBidderResponse    adapters.BidderResponse
		expectedBidderResponse adapters.BidderResponse
		expectedReject         *RejectError
		expectedModuleContexts *moduleContexts
		expectedStageOutcomes  []StageOutcome
	}{
		{
			description:            "Payload not changed if hook execution plan empty",
			givenPlanBuilder:       hooks.EmptyPlanBuilder{},
			givenBidderResponse:    resp,
			expectedBidderResponse: resp,
			expectedReject:         nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:  []StageOutcome{},
		},
		{
			description:            "Payload changed if hooks return mutations",
			givenPlanBuilder:       TestApplyHookMutationsBuilder{},
			givenBidderResponse:    resp,
			expectedBidderResponse: expResp,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: vEntity,
					Stage:  hooks.StageRawBidderResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidderResponse.bid.deal-priority, mutation type: %s", hookstage.MutationUpdate),
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
			description:            "Stage execution can be rejected",
			givenPlanBuilder:       TestRejectPlanBuilder{},
			givenBidderResponse:    resp,
			expectedBidderResponse: resp,
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookImplCode: "foo"}, hooks.StageRawBidderResponse.String()},
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: vEntity,
					Stage:  hooks.StageRawBidderResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionReject,
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										`Module foobar (hook: foo) rejected request with code 0 at raw_bidder_response stage`,
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
			description:            "Response can be changed when a hook times out",
			givenPlanBuilder:       TestWithTimeoutPlanBuilder{},
			givenBidderResponse:    resp,
			expectedBidderResponse: expResp,
			expectedReject:         nil,
			expectedModuleContexts: foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: vEntity,
					Stage:  hooks.StageRawBidderResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{"foobar", "foo"},
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
									HookID:        HookID{"foobar", "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: bidderResponse.bid.deal-priority, mutation type: %s", hookstage.MutationUpdate),
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
			description:            "Modules contexts are preserved and correct",
			givenPlanBuilder:       TestWithModuleContextsPlanBuilder{},
			givenBidderResponse:    resp,
			expectedBidderResponse: expResp,
			expectedReject:         nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"raw-bidder-response-ctx-1": "some-ctx-1", "raw-bidder-response-ctx-3": "some-ctx-3"},
				"module-2": {"raw-bidder-response-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: vEntity,
					Stage:  hooks.StageRawBidderResponse.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-2", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "bar"},
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
		t.Run(test.description, func(ti *testing.T) {
			exec := NewHookExecutor(test.givenPlanBuilder, EndpointAuction, &metricsConfig.NilMetricsEngine{})

			privacyConfig := getModuleActivities("foo", false, false)
			ac := privacy.NewActivityControl(privacyConfig)
			exec.SetActivityControl(ac)

			reject := exec.ExecuteRawBidderResponseStage(&test.givenBidderResponse, "the-bidder")

			assert.Equal(ti, test.expectedReject, reject, "Unexpected stage reject.")
			assert.Equal(ti, test.expectedBidderResponse, test.givenBidderResponse, "Incorrect response update.")
			assert.Equal(ti, test.expectedModuleContexts, exec.moduleContexts, "Incorrect module contexts")

			stageOutcomes := exec.GetOutcomes()
			if len(test.expectedStageOutcomes) == 0 {
				assert.Empty(ti, stageOutcomes, "Incorrect stage outcomes.")
			} else {
				assertEqualStageOutcomes(ti, test.expectedStageOutcomes[0], stageOutcomes[0])
			}
		})
	}
}

func TestExecuteAllProcessedBidResponsesStage(t *testing.T) {
	foobarModuleCtx := &moduleContexts{ctxs: map[string]hookstage.ModuleContext{"foobar": nil}}

	expectedAllProcBidResponses := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		"some-bidder": {Bids: []*entities.PbsOrtbBid{{DealPriority: 1}}},
	}
	expectedUpdatedAllProcBidResponses := map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
		"some-bidder": {Bids: []*entities.PbsOrtbBid{{DealPriority: 10}}},
	}

	testCases := []struct {
		description             string
		givenBiddersResponse    map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		givenPlanBuilder        hooks.ExecutionPlanBuilder
		expectedBiddersResponse map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid
		expectedReject          *RejectError
		expectedModuleContexts  *moduleContexts
		expectedStageOutcomes   []StageOutcome
	}{
		{
			description: "Payload not changed if hook execution plan empty",
			givenBiddersResponse: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"some-bidder": {Bids: []*entities.PbsOrtbBid{{DealPriority: 1}}},
			},
			givenPlanBuilder:        hooks.EmptyPlanBuilder{},
			expectedBiddersResponse: expectedAllProcBidResponses,
			expectedReject:          nil,
			expectedModuleContexts:  &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:   []StageOutcome{},
		},
		{
			description: "Payload changed if hooks return mutations",
			givenBiddersResponse: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"some-bidder": {Bids: []*entities.PbsOrtbBid{{DealPriority: 1}}},
			},
			givenPlanBuilder:        TestApplyHookMutationsBuilder{},
			expectedBiddersResponse: expectedUpdatedAllProcBidResponses,
			expectedReject:          nil,
			expectedModuleContexts:  foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAllProcessedBidResponses,
					Stage:  hooks.StageAllProcessedBidResponses.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: processedBidderResponse.bid.deal-priority, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusExecutionFailure,
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
			description: "Stage execution can't be rejected - stage doesn't support rejection",
			givenBiddersResponse: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"some-bidder": {Bids: []*entities.PbsOrtbBid{{DealPriority: 1}}},
			},
			givenPlanBuilder:        TestRejectPlanBuilder{},
			expectedBiddersResponse: expectedUpdatedAllProcBidResponses,
			expectedReject:          &RejectError{0, HookID{ModuleCode: "foobar", HookImplCode: "foo"}, hooks.StageAllProcessedBidResponses.String()},
			expectedModuleContexts:  foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAllProcessedBidResponses,
					Stage:  hooks.StageAllProcessedBidResponses.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
									Status:        StatusExecutionFailure,
									Action:        "",
									Message:       "",
									DebugMessages: nil,
									Errors: []string{
										fmt.Sprintf("Module (name: foobar, hook code: foo) tried to reject request on the %s stage that does not support rejection", hooks.StageAllProcessedBidResponses),
									},
									Warnings: nil,
								},
							},
						},
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: processedBidderResponse.bid.deal-priority, mutation type: %s", hookstage.MutationUpdate),
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
			description: "Stage execution can be timed out",
			givenBiddersResponse: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"some-bidder": {Bids: []*entities.PbsOrtbBid{{DealPriority: 1}}},
			},
			givenPlanBuilder:        TestWithTimeoutPlanBuilder{},
			expectedBiddersResponse: expectedUpdatedAllProcBidResponses,
			expectedReject:          nil,
			expectedModuleContexts:  foobarModuleCtx,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAllProcessedBidResponses,
					Stage:  hooks.StageAllProcessedBidResponses.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: processedBidderResponse.bid.deal-priority, mutation type: %s", hookstage.MutationUpdate),
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
			description: "Modules contexts are preserved and correct",
			givenBiddersResponse: map[openrtb_ext.BidderName]*entities.PbsOrtbSeatBid{
				"some-bidder": {Bids: []*entities.PbsOrtbBid{{DealPriority: 1}}},
			},
			givenPlanBuilder:        TestWithModuleContextsPlanBuilder{},
			expectedBiddersResponse: expectedAllProcBidResponses,
			expectedReject:          nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
				"module-1": {"all-processed-bid-responses-ctx-1": "some-ctx-1"},
				"module-2": {"all-processed-bid-responses-ctx-2": "some-ctx-2"},
			}},
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityAllProcessedBidResponses,
					Stage:  hooks.StageAllProcessedBidResponses.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "module-2", HookImplCode: "bar"},
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

			privacyConfig := getModuleActivities("foo", false, false)
			ac := privacy.NewActivityControl(privacyConfig)
			exec.SetActivityControl(ac)

			exec.ExecuteAllProcessedBidResponsesStage(test.givenBiddersResponse)

			assert.Equal(t, test.expectedBiddersResponse, test.givenBiddersResponse, "Incorrect bidders response.")
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
	resp := &openrtb2.BidResponse{CustomData: "some-custom-data"}
	expResp := &openrtb2.BidResponse{CustomData: "new-custom-data"}

	testCases := []struct {
		description            string
		givenPlanBuilder       hooks.ExecutionPlanBuilder
		givenResponse          *openrtb2.BidResponse
		expectedResponse       *openrtb2.BidResponse
		expectedReject         *RejectError
		expectedModuleContexts *moduleContexts
		expectedStageOutcomes  []StageOutcome
	}{
		{
			description:            "Payload not changed if hook execution plan empty",
			givenPlanBuilder:       hooks.EmptyPlanBuilder{},
			givenResponse:          resp,
			expectedResponse:       resp,
			expectedReject:         nil,
			expectedModuleContexts: &moduleContexts{ctxs: map[string]hookstage.ModuleContext{}},
			expectedStageOutcomes:  []StageOutcome{},
		},
		{
			description:            "Payload changed if hooks return mutations",
			givenPlanBuilder:       TestApplyHookMutationsBuilder{},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
			givenResponse:          resp,
			expectedResponse:       expResp,
			expectedReject:         &RejectError{0, HookID{ModuleCode: "foobar", HookImplCode: "foo"}, hooks.StageAuctionResponse.String()},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "foo"},
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
									HookID:        HookID{ModuleCode: "foobar", HookImplCode: "bar"},
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
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "foo"},
									Status:        StatusSuccess,
									Action:        ActionNone,
									Message:       "",
									DebugMessages: nil,
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{ModuleCode: "module-2", HookImplCode: "baz"},
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
									HookID:        HookID{ModuleCode: "module-1", HookImplCode: "bar"},
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

			privacyConfig := getModuleActivities("foo", false, false)
			ac := privacy.NewActivityControl(privacyConfig)
			exec.SetActivityControl(ac)

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

	// test that context added at the processed-auction stage merged with existing module contexts
	err = exec.ExecuteProcessedAuctionStage(&openrtb_ext.RequestWrapper{BidRequest: &openrtb2.BidRequest{}})
	assert.Nil(t, err, "Unexpected reject from processed-auction stage.")
	assert.Equal(t, &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
		"module-1": {
			"entrypoint-ctx-1":        "some-ctx-1",
			"entrypoint-ctx-3":        "some-ctx-3",
			"raw-auction-ctx-1":       "some-ctx-1",
			"raw-auction-ctx-3":       "some-ctx-3",
			"processed-auction-ctx-1": "some-ctx-1",
			"processed-auction-ctx-3": "some-ctx-3",
		},
		"module-2": {
			"entrypoint-ctx-2":        "some-ctx-2",
			"raw-auction-ctx-2":       "some-ctx-2",
			"processed-auction-ctx-2": "some-ctx-2",
		},
	}}, exec.moduleContexts, "Wrong module contexts after executing processed-auction hook.")

	// test that context added at the raw bidder response stage merged with existing module contexts
	reject = exec.ExecuteRawBidderResponseStage(&adapters.BidderResponse{}, "some-bidder")
	assert.Nil(t, reject, "Unexpected reject from raw-bidder-response stage.")
	assert.Equal(t, &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
		"module-1": {
			"entrypoint-ctx-1":          "some-ctx-1",
			"entrypoint-ctx-3":          "some-ctx-3",
			"raw-auction-ctx-1":         "some-ctx-1",
			"raw-auction-ctx-3":         "some-ctx-3",
			"processed-auction-ctx-1":   "some-ctx-1",
			"processed-auction-ctx-3":   "some-ctx-3",
			"raw-bidder-response-ctx-1": "some-ctx-1",
			"raw-bidder-response-ctx-3": "some-ctx-3",
		},
		"module-2": {
			"entrypoint-ctx-2":          "some-ctx-2",
			"raw-auction-ctx-2":         "some-ctx-2",
			"processed-auction-ctx-2":   "some-ctx-2",
			"raw-bidder-response-ctx-2": "some-ctx-2",
		},
	}}, exec.moduleContexts, "Wrong module contexts after executing raw-bidder-response hook.")

	// test that context added at the auction-response stage merged with existing module contexts
	exec.ExecuteAuctionResponseStage(&openrtb2.BidResponse{})
	assert.Nil(t, reject, "Unexpected reject from raw-auction stage.")
	assert.Equal(t, &moduleContexts{ctxs: map[string]hookstage.ModuleContext{
		"module-1": {
			"entrypoint-ctx-1":          "some-ctx-1",
			"entrypoint-ctx-3":          "some-ctx-3",
			"raw-auction-ctx-1":         "some-ctx-1",
			"raw-auction-ctx-3":         "some-ctx-3",
			"processed-auction-ctx-1":   "some-ctx-1",
			"processed-auction-ctx-3":   "some-ctx-3",
			"raw-bidder-response-ctx-1": "some-ctx-1",
			"raw-bidder-response-ctx-3": "some-ctx-3",
			"auction-response-ctx-1":    "some-ctx-1",
			"auction-response-ctx-3":    "some-ctx-3",
		},
		"module-2": {
			"entrypoint-ctx-2":          "some-ctx-2",
			"raw-auction-ctx-2":         "some-ctx-2",
			"processed-auction-ctx-2":   "some-ctx-2",
			"raw-bidder-response-ctx-2": "some-ctx-2",
			"auction-response-ctx-2":    "some-ctx-2",
		},
	}}, exec.moduleContexts, "Wrong module contexts after executing auction-response hook.")
}

type TestApplyHookMutationsBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestApplyHookMutationsBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "foobaz", Hook: mockFailedMutationHook{}},
				{Module: "foobar", Code: "bar", Hook: mockUpdateQueryEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Millisecond,
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
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
				{Module: "foobar", Code: "bar", Hook: mockFailedMutationHook{}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "baz", Hook: mockFailureHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForProcessedAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.ProcessedAuctionRequest] {
	return hooks.Plan[hookstage.ProcessedAuctionRequest]{
		hooks.Group[hookstage.ProcessedAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.ProcessedAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBidRequestHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForBidderRequestStage(_ string, _ *config.Account) hooks.Plan[hookstage.BidderRequest] {
	return hooks.Plan[hookstage.BidderRequest]{
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBidRequestHook{}},
				{Module: "foobar", Code: "bar", Hook: mockFailedMutationHook{}},
			},
		},
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar", Code: "baz", Hook: mockFailureHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBidderResponseHook{}},
			},
		},
	}
}

func (e TestApplyHookMutationsBuilder) PlanForAllProcessedBidResponsesStage(_ string, _ *config.Account) hooks.Plan[hookstage.AllProcessedBidResponses] {
	return hooks.Plan[hookstage.AllProcessedBidResponses]{
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBiddersResponsesHook{}},
				{Module: "foobar", Code: "bar", Hook: mockFailedMutationHook{}},
			},
		},
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
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
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Millisecond,
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
			Timeout: 10 * time.Millisecond,
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
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "bar", Hook: mockRejectHook{}},
				// next hook rejected: we use timeout hook to make sure
				// that it runs longer than previous one, so it won't be executed earlier
				{Module: "foobar", Code: "baz", Hook: mockTimeoutHook{}},
			},
		},
		// group of hooks rejected
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForProcessedAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.ProcessedAuctionRequest] {
	return hooks.Plan[hookstage.ProcessedAuctionRequest]{
		hooks.Group[hookstage.ProcessedAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.ProcessedAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockRejectHook{}},
			},
		},
		hooks.Group[hookstage.ProcessedAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.ProcessedAuctionRequest]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidRequestHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForBidderRequestStage(_ string, _ *config.Account) hooks.Plan[hookstage.BidderRequest] {
	return hooks.Plan[hookstage.BidderRequest]{
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar", Code: "foo", Hook: mockRejectHook{}},
			},
		},
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidRequestHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "foo", Hook: mockRejectHook{}},
			},
		},
	}
}

func (e TestRejectPlanBuilder) PlanForAllProcessedBidResponsesStage(_ string, _ *config.Account) hooks.Plan[hookstage.AllProcessedBidResponses] {
	return hooks.Plan[hookstage.AllProcessedBidResponses]{
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "foobar", Code: "baz", Hook: mockErrorHook{}},
			},
		},
		// rejection ignored, stage doesn't support rejection
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "foobar", Code: "foo", Hook: mockRejectHook{}},
			},
		},
		// hook executed and payload updated because this stage doesn't support rejection
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBiddersResponsesHook{}},
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
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "bar", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "baz", Hook: mockUpdateBodyHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForRawAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawAuctionRequest] {
	return hooks.Plan[hookstage.RawAuctionRequest]{
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateBodyHook{}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "foobar", Code: "bar", Hook: mockTimeoutHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForProcessedAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.ProcessedAuctionRequest] {
	return hooks.Plan[hookstage.ProcessedAuctionRequest]{
		hooks.Group[hookstage.ProcessedAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.ProcessedAuctionRequest]{
				{Module: "foobar", Code: "foo", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.ProcessedAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.ProcessedAuctionRequest]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidRequestHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForBidderRequestStage(_ string, _ *config.Account) hooks.Plan[hookstage.BidderRequest] {
	return hooks.Plan[hookstage.BidderRequest]{
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar", Code: "foo", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidRequestHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "foo", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBidderResponseHook{}},
			},
		},
	}
}

func (e TestWithTimeoutPlanBuilder) PlanForAllProcessedBidResponsesStage(_ string, _ *config.Account) hooks.Plan[hookstage.AllProcessedBidResponses] {
	return hooks.Plan[hookstage.AllProcessedBidResponses]{
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "foobar", Code: "foo", Hook: mockTimeoutHook{}},
			},
		},
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "foobar", Code: "bar", Hook: mockUpdateBiddersResponsesHook{}},
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
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "entrypoint-ctx-1", val: "some-ctx-1"}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Millisecond,
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
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "raw-auction-ctx-1", val: "some-ctx-1"}},
				{Module: "module-2", Code: "baz", Hook: mockModuleContextHook{key: "raw-auction-ctx-2", val: "some-ctx-2"}},
			},
		},
		hooks.Group[hookstage.RawAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawAuctionRequest]{
				{Module: "module-1", Code: "bar", Hook: mockModuleContextHook{key: "raw-auction-ctx-3", val: "some-ctx-3"}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForProcessedAuctionStage(_ string, _ *config.Account) hooks.Plan[hookstage.ProcessedAuctionRequest] {
	return hooks.Plan[hookstage.ProcessedAuctionRequest]{
		hooks.Group[hookstage.ProcessedAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.ProcessedAuctionRequest]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "processed-auction-ctx-1", val: "some-ctx-1"}},
			},
		},
		hooks.Group[hookstage.ProcessedAuctionRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.ProcessedAuctionRequest]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextHook{key: "processed-auction-ctx-2", val: "some-ctx-2"}},
				{Module: "module-1", Code: "baz", Hook: mockModuleContextHook{key: "processed-auction-ctx-3", val: "some-ctx-3"}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForBidderRequestStage(_ string, _ *config.Account) hooks.Plan[hookstage.BidderRequest] {
	return hooks.Plan[hookstage.BidderRequest]{
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "bidder-request-ctx-1", val: "some-ctx-1"}},
			},
		},
		hooks.Group[hookstage.BidderRequest]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.BidderRequest]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextHook{key: "bidder-request-ctx-2", val: "some-ctx-2"}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForRawBidderResponseStage(_ string, _ *config.Account) hooks.Plan[hookstage.RawBidderResponse] {
	return hooks.Plan[hookstage.RawBidderResponse]{
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "raw-bidder-response-ctx-1", val: "some-ctx-1"}},
				{Module: "module-2", Code: "baz", Hook: mockModuleContextHook{key: "raw-bidder-response-ctx-2", val: "some-ctx-2"}},
			},
		},
		hooks.Group[hookstage.RawBidderResponse]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.RawBidderResponse]{
				{Module: "module-1", Code: "bar", Hook: mockModuleContextHook{key: "raw-bidder-response-ctx-3", val: "some-ctx-3"}},
			},
		},
	}
}

func (e TestWithModuleContextsPlanBuilder) PlanForAllProcessedBidResponsesStage(_ string, _ *config.Account) hooks.Plan[hookstage.AllProcessedBidResponses] {
	return hooks.Plan[hookstage.AllProcessedBidResponses]{
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "module-1", Code: "foo", Hook: mockModuleContextHook{key: "all-processed-bid-responses-ctx-1", val: "some-ctx-1"}},
			},
		},
		hooks.Group[hookstage.AllProcessedBidResponses]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.AllProcessedBidResponses]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextHook{key: "all-processed-bid-responses-ctx-2", val: "some-ctx-2"}},
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

type TestAllHookResultsBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestAllHookResultsBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module.x-1", Code: "code-1", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "module.x-1", Code: "code-3", Hook: mockTimeoutHook{}},
				{Module: "module.x-1", Code: "code-4", Hook: mockFailureHook{}},
				{Module: "module.x-1", Code: "code-5", Hook: mockErrorHook{}},
				{Module: "module.x-1", Code: "code-6", Hook: mockFailedMutationHook{}},
				{Module: "module.x-1", Code: "code-7", Hook: mockModuleContextHook{key: "key", val: "val"}},
			},
		},
		// place the reject hook in a separate group because it rejects the stage completely
		// thus we can not make accurate mock calls if it is processed in parallel with others
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Second,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module.x-1", Code: "code-2", Hook: mockRejectHook{}},
			},
		},
	}
}
