package hookexecution

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookanalytics"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteEntrypointStage_DoesNotChangeRequestForEmptyPlan(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{0},
		Entity:        hookstage.EntityHttpRequest,
		Stage:         hooks.StageEntrypoint,
		Groups:        []GroupOutcome{},
	}
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      Auction_endpoint,
		PlanBuilder:   hooks.EmptyPlanBuilder{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)
	if bytes.Compare(body, newBody) != 0 {
		t.Error("request body should not change")
	}
}

func TestExecuteEntrypointStage_CanApplyHookMutations(t *testing.T) {
	expectedOutcome := StageOutcome{
		Entity: hookstage.EntityHttpRequest,
		Stage:  hooks.StageEntrypoint,
		Groups: []GroupOutcome{
			{
				InvocationResults: []*HookOutcome{
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
				InvocationResults: []*HookOutcome{
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
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      Auction_endpoint,
		PlanBuilder:   TestApplyHookMutationsBuilder{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if bytes.Compare(body, newBody) == 0 {
		t.Error("request body not changed after applying hook result")
	}

	if _, dt, _, _ := jsonparser.Get(newBody, "name"); dt != jsonparser.NotExist {
		t.Error("'name' property expected to be deleted from request body.")
	}

	if req.Header.Get("foo") == "" {
		t.Error("header not changed inside hook.Call method")
	}

	if req.URL.Query().Get("foo") == "" {
		t.Error("query params not changed inside hook.Call method")
	}
}

type mockUpdateHeaderEntrypointHook struct{}

func (e mockUpdateHeaderEntrypointHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		payload.Request.Header.Add("foo", "bar")
		return payload, nil
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateQueryEntrypointHook struct{}

func (e mockUpdateQueryEntrypointHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
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

func (e mockUpdateBodyEntrypointHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
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

func TestExecuteEntrypointStage_CanRejectHook(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        hookstage.EntityHttpRequest,
		Stage:         hooks.StageEntrypoint,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []*HookOutcome{
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
				InvocationResults: []*HookOutcome{
					{
						ExecutionTime: ExecutionTime{},
						AnalyticsTags: hookanalytics.Analytics{},
						HookID:        HookID{"foobar", "bar"},
						Status:        StatusSuccess,
						Action:        ActionReject,
						Message:       "",
						DebugMessages: nil,
						Errors: []string{
							`Module rejected stage, reason: ""`,
						},
						Warnings: nil,
					},
				},
			},
		},
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	require.NoError(t, err, "Unexpected error creating http request: %s", err)
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      Auction_endpoint,
		PlanBuilder:   TestRejectPlanBuilder{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.NotNil(t, reject, "Unexpected successful execution of entrypoint hook")
	require.Equal(t, reject, &RejectError{}, "Unexpected reject returned from entrypoint hook")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)
	assert.Equal(t, body, newBody, "request body shouldn't change if request rejected")
}

type mockRejectEntrypointHook struct{}

func (e mockRejectEntrypointHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
}

func TestExecuteEntrypointStage_CanTimeoutOneOfHooks(t *testing.T) {
	expectedOutcome := StageOutcome{
		ExecutionTime: ExecutionTime{},
		Entity:        hookstage.EntityHttpRequest,
		Stage:         hooks.StageEntrypoint,
		Groups: []GroupOutcome{
			{
				ExecutionTime: ExecutionTime{},
				InvocationResults: []*HookOutcome{
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
				InvocationResults: []*HookOutcome{
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
	}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}
	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      Auction_endpoint,
		PlanBuilder:   TestWithTimeoutPlanBuilder{},
	}

	newBody, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	assertEqualStageOutcomes(t, expectedOutcome, stOut)

	if bytes.Compare(body, newBody) == 0 {
		t.Error("request body not changed after applying hook result")
	}

	if req.Header.Get("foo") == "" {
		t.Error("header not changed inside hook.Call method")
	}

	if req.URL.Query().Get("bar") != "" {
		t.Errorf("query params should not change inside hook.Call method because of timeout")
	}
}

type mockTimeoutEntrypointHook struct{}

func (e mockTimeoutEntrypointHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
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

func TestExecuteEntrypointStage_ModuleContextsAreCreated(t *testing.T) {
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	exec := HookExecutor{
		InvocationCtx: &hookstage.InvocationContext{},
		Endpoint:      Auction_endpoint,
		PlanBuilder:   TestWithModuleContextsPlanBuilder{},
	}
	_, reject := exec.ExecuteEntrypointStage(req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	stOut := exec.GetOutcomes()[0]
	if len(stOut.Groups) != 2 {
		t.Error("some hook groups have not been processed")
	}

	ctx1 := exec.InvocationCtx.ModuleContextFor("module-1")
	if ctx1.Ctx["some-ctx-1"] != "some-ctx-1" {
		t.Error("context for module-1 not created")
	}

	ctx2 := exec.InvocationCtx.ModuleContextFor("module-2")
	if ctx2.Ctx["some-ctx-2"] != "some-ctx-2" {
		t.Error("context for module-2 not created")
	}
}

type mockModuleContextEntrypointHook1 struct{}

func (e mockModuleContextEntrypointHook1) HandleEntrypointHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-1": "some-ctx-1"}
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

type mockModuleContextEntrypointHook2 struct{}

func (e mockModuleContextEntrypointHook2) HandleEntrypointHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-2": "some-ctx-2"}
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
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
