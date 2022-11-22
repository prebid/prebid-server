package hookexecution

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteEntrypointStage(t *testing.T) {
	const body string = `{"name": "John", "last_name": "Doe"}`
	const urlString string = "https://prebid.com/openrtb2/auction"

	testCases := []struct {
		description           string
		givenBody             string
		givenUrl              string
		givenPlanBuilder      hooks.ExecutionPlanBuilder
		expectedBody          string
		expectedHeader        http.Header
		expectedQuery         url.Values
		expectedReject        *RejectError
		expectedStageOutcomes []StageOutcome
	}{
		{
			description:           "Payload not changed if hook execution plan empty",
			givenBody:             body,
			givenUrl:              urlString,
			givenPlanBuilder:      hooks.EmptyPlanBuilder{},
			expectedBody:          body,
			expectedHeader:        http.Header{},
			expectedQuery:         url.Values{},
			expectedReject:        nil,
			expectedStageOutcomes: []StageOutcome{},
		},
		{
			description:      "Payload changed if hooks return mutations",
			givenBody:        body,
			givenUrl:         urlString,
			givenPlanBuilder: TestApplyHookMutationsBuilder{},
			expectedBody:     `{"last_name": "Doe", "foo": "bar"}`,
			expectedHeader:   http.Header{"Foo": []string{"bar"}},
			expectedQuery:    url.Values{"foo": []string{"baz"}},
			expectedReject:   nil,
			expectedStageOutcomes: []StageOutcome{
				{
					Entity: entityHttpRequest,
					Stage:  hooks.StageEntrypoint.String(),
					Groups: []GroupOutcome{
						{
							InvocationResults: []HookOutcome{
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{"foobar", "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate)},
									Errors:        nil,
									Warnings:      nil,
								},
								{
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{"foobar", "bar"},
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
									HookID:        HookID{"foobar", "baz"},
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
			description:      "Stage execution can be rejected",
			givenBody:        body,
			givenUrl:         urlString,
			givenPlanBuilder: TestRejectPlanBuilder{},
			expectedBody:     body,
			expectedHeader:   http.Header{"Foo": []string{"bar"}},
			expectedQuery:    url.Values{},
			expectedReject:   &RejectError{0, HookID{"foobar", "bar"}, hooks.StageEntrypoint.String()},
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
									HookID:        HookID{"foobar", "foo"},
									Status:        StatusSuccess,
									Action:        ActionUpdate,
									Message:       "",
									DebugMessages: []string{
										fmt.Sprintf("Hook mutation successfully applied, affected key: header.foo, mutation type: %s", hookstage.MutationUpdate),
									},
									Errors:   nil,
									Warnings: nil,
								},
							},
						},
						{
							ExecutionTime: ExecutionTime{},
							InvocationResults: []HookOutcome{
								{
									ExecutionTime: ExecutionTime{},
									AnalyticsTags: hookanalytics.Analytics{},
									HookID:        HookID{"foobar", "bar"},
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
			description:      "Stage execution can be timed out",
			givenBody:        body,
			givenUrl:         urlString,
			givenPlanBuilder: TestWithTimeoutPlanBuilder{},
			expectedBody:     `{"foo":"bar", "last_name":"Doe"}`,
			expectedHeader:   http.Header{"Foo": []string{"bar"}},
			expectedQuery:    url.Values{},
			expectedReject:   nil,
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
									HookID:        HookID{"foobar", "foo"},
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
									HookID:        HookID{"foobar", "bar"},
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
									HookID:        HookID{"foobar", "baz"},
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
	}

	for _, test := range testCases {
		t.Run(test.description, func(t *testing.T) {
			body := []byte(test.givenBody)
			reader := bytes.NewReader(body)
			req, err := http.NewRequest(http.MethodPost, test.givenUrl, reader)
			assert.NoError(t, err)

			exec := NewHookExecutor(test.givenPlanBuilder, EndpointAuction)
			newBody, reject := exec.ExecuteEntrypointStage(req, body)

			assert.Equal(t, test.expectedReject, reject, "Unexpected stage reject.")
			assert.JSONEq(t, test.expectedBody, string(newBody), "Incorrect request body.")
			assert.Equal(t, test.expectedHeader, req.Header, "Incorrect request header.")
			assert.Equal(t, test.expectedQuery, req.URL.Query(), "Incorrect request query.")

			stageOutcomes := exec.GetOutcomes()
			if len(test.expectedStageOutcomes) == 0 {
				assert.Empty(t, stageOutcomes, "Incorrect stage outcomes.")
			} else {
				assertEqualStageOutcomes(t, test.expectedStageOutcomes[0], stageOutcomes[0])
			}
		})
	}
}

func TestExecuteEntrypointStage_ModuleContextsAreCreated(t *testing.T) {
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	exec := NewHookExecutor(TestWithModuleContextsPlanBuilder{}, EndpointAuction)
	_, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assert.Len(t, stOut.Groups, 2, "some hook groups have not been processed")

	ctx1, ok := exec.moduleContexts.get("module-1")
	assert.True(t, ok, "Failed to find context for module-1")
	assert.Equal(t, ctx1["some-ctx-1"], "some-ctx-1", "Invalid value for some-ctx-1")

	ctx2, ok := exec.moduleContexts.get("module-2")
	assert.True(t, ok, "Failed to find context for module-2")
	assert.Equal(t, ctx2["some-ctx-2"], "some-ctx-2", "Invalid value for some-ctx-2")
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
	time.Sleep(2 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("bar", "foo")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, hookstage.MutationUpdate, "param", "bar")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockModuleContextEntrypointHook1 struct{}

func (e mockModuleContextEntrypointHook1) HandleEntrypointHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{"some-ctx-1": "some-ctx-1"}
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

type mockModuleContextEntrypointHook2 struct{}

func (e mockModuleContextEntrypointHook2) HandleEntrypointHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{"some-ctx-2": "some-ctx-2"}
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
				{Module: "foobar", Code: "bar", Hook: mockUpdateQueryEntrypointHook{}},
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

type TestRejectPlanBuilder struct {
	hooks.EmptyPlanBuilder
}

func (e TestRejectPlanBuilder) PlanForEntrypointStage(_ string) hooks.Plan[hookstage.Entrypoint] {
	return hooks.Plan[hookstage.Entrypoint]{
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "foo", Hook: mockUpdateHeaderEntrypointHook{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "foobar", Code: "bar", Hook: mockRejectEntrypointHook{}},
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
				{Module: "module-1", Code: "foo", Hook: mockModuleContextEntrypointHook1{}},
			},
		},
		hooks.Group[hookstage.Entrypoint]{
			Timeout: 1 * time.Millisecond,
			Hooks: []hooks.HookWrapper[hookstage.Entrypoint]{
				{Module: "module-2", Code: "bar", Hook: mockModuleContextEntrypointHook2{}},
			},
		},
	}
}
