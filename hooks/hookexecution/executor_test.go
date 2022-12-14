package hookexecution

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/metrics"
	metricsConfig "github.com/prebid/prebid-server/metrics/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestEmptyHookExecutor(t *testing.T) {
	executor := EmptyHookExecutor{}
	executor.SetAccount(&config.Account{})

	body := []byte(`{"foo": "bar"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	assert.NoError(t, err, "Failed to create http request.")

	newBody, rejectErr := executor.ExecuteEntrypointStage(req, body)
	outcomes := executor.GetOutcomes()

	assert.Equal(t, EmptyHookExecutor{}, executor, "EmptyHookExecutor shouldn't be changed.")
	assert.Nil(t, rejectErr, "EmptyHookExecutor shouldn't return reject error.")
	assert.Equal(t, body, newBody, "EmptyHookExecutor shouldn't change body.")
	assert.Empty(t, outcomes, "EmptyHookExecutor shouldn't return stage outcomes.")

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
					ExecutionTime: ExecutionTime{},
					Entity:        entityHttpRequest,
					Stage:         hooks.StageEntrypoint.String(),
					Groups: []GroupOutcome{
						{
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
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
			description:            "Stage execution can be timed out",
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
					ExecutionTime: ExecutionTime{},
					Entity:        entityHttpRequest,
					Stage:         hooks.StageEntrypoint.String(),
					Groups: []GroupOutcome{
						{
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
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
									ExecutionTime: ExecutionTime{},
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
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
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
					ExecutionTime: ExecutionTime{},
					Entity:        entityHttpRequest,
					Stage:         hooks.StageEntrypoint.String(),
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
								{
									ExecutionTime: ExecutionTime{},
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
	moduleLabels := metrics.ModuleLabels{
		Module: "module-1",
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

type mockUpdateHeaderEntrypointHook struct{}

func (e mockUpdateHeaderEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		payload.Request.Header.Add("foo", "bar")
		return payload, nil
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateQueryEntrypointHook struct{}

func (e mockUpdateQueryEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("foo", "baz")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, hookstage.MutationUpdate, "param", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateBodyEntrypointHook struct{}

func (e mockUpdateBodyEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(
		func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			payload.Body = []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationUpdate, "body", "foo",
	).AddMutation(
		func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			payload.Body = []byte(`{"last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationDelete, "body", "name",
	)

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockRejectEntrypointHook struct{}

func (e mockRejectEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
}

type mockTimeoutEntrypointHook struct{}

func (e mockTimeoutEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	time.Sleep(3 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("bar", "foo")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, hookstage.MutationUpdate, "param", "bar")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockModuleContextHook struct {
	key, val string
}

func (e mockModuleContextHook) HandleEntrypointHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: miCtx.ModuleContext}, nil
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
				{Module: "foobar", Code: "foobaz", Hook: mockFailedMutationEntrypointHook{}},
				{Module: "foobar", Code: "bar", Hook: mockUpdateQueryEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "baz", Hook: mockUpdateBodyEntrypointHook{}},
				{Module: "foobar", Code: "foo", Hook: mockFailureEntrypointHook{}},
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
				{Module: "foobar", Code: "baz", Hook: mockErrorEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 5 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				// reject stage
				{Module: "foobar", Code: "bar", Hook: mockRejectEntrypointHook{}},
				// next hook rejected: we use timeout hook to make sure
				// that it runs longer than previous one, so it won't be executed earlier
				{Module: "foobar", Code: "baz", Hook: mockTimeoutEntrypointHook{}},
			},
		},
		// group of hooks rejected
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "foobar", Code: "baz", Hook: mockErrorEntrypointHook{}},
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
				{Module: "foobar", Code: "bar", Hook: mockTimeoutEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "baz", Hook: mockUpdateBodyEntrypointHook{}},
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

type mockFailureEntrypointHook struct{}

func (h mockFailureEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, FailureError{Message: "attribute not found"}
}

type mockErrorEntrypointHook struct{}

func (h mockErrorEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, errors.New("unexpected error")
}

type mockFailedMutationEntrypointHook struct{}

func (h mockFailedMutationEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	changeSet := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	changeSet.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: changeSet}, nil
}

type TestAllHookResultsBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestAllHookResultsBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-1", Code: "code-1", Hook: mockUpdateHeaderEntrypointHook{}},
				{Module: "module-1", Code: "code-3", Hook: mockTimeoutEntrypointHook{}},
				{Module: "module-1", Code: "code-4", Hook: mockFailureEntrypointHook{}},
				{Module: "module-1", Code: "code-5", Hook: mockErrorEntrypointHook{}},
				{Module: "module-1", Code: "code-6", Hook: mockFailedMutationEntrypointHook{}},
				{Module: "module-1", Code: "code-7", Hook: mockModuleContextHook{key: "key", val: "val"}},
			},
		},
		// place the reject hook in a separate group because it rejects the stage completely
		// thus we can not make accurate mock calls if it is processed in parallel with others
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 10 * time.Second,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-1", Code: "code-2", Hook: mockRejectEntrypointHook{}},
			},
		},
	}
}
