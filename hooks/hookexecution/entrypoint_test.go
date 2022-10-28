package hookexecution

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/hooks"
	"github.com/prebid/prebid-server/hooks/hookstage"
	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteEntrypointStage_DoesNotChangeRequestForEmptyPlan(t *testing.T) {
	plan := hooks.Plan[hookstage.Entrypoint]{}

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	stRes, newBody, reject := ExecuteEntrypointStage(&invocation.InvocationContext{}, plan, req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	if len(stRes.GroupsResults) != 0 {
		t.Error("unexpected non-empty stage result from empty plan")
	}

	if bytes.Compare(body, newBody) != 0 {
		t.Error("request body should not change")
	}
}

func TestExecuteEntrypointStage_CanApplyHookMutations(t *testing.T) {
	plan := hooks.Plan[hookstage.Entrypoint]{
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

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	stRes, newBody, reject := ExecuteEntrypointStage(&invocation.InvocationContext{}, plan, req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	if len(stRes.GroupsResults) != 2 {
		t.Error("some hook groups have not been processed")
	}

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

func (e mockUpdateHeaderEntrypointHook) HandleEntrypointHook(_ context.Context, _ *invocation.ModuleContext, _ hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	c := &invocation.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		payload.Request.Header.Add("foo", "bar")
		return payload, nil
	}, invocation.MutationUpdate, "header", "foo")

	return invocation.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateQueryEntrypointHook struct{}

func (e mockUpdateQueryEntrypointHook) HandleEntrypointHook(_ context.Context, _ *invocation.ModuleContext, _ hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	c := &invocation.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("foo", "baz")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, invocation.MutationUpdate, "param", "foo")

	return invocation.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateBodyEntrypointHook struct{}

func (e mockUpdateBodyEntrypointHook) HandleEntrypointHook(_ context.Context, _ *invocation.ModuleContext, _ hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	c := &invocation.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(
		func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			payload.Body = []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, invocation.MutationUpdate, "body", "foo",
	).AddMutation(
		func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
			payload.Body = []byte(`{"last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, invocation.MutationDelete, "body", "name",
	)

	return invocation.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

func TestExecuteEntrypointStage_CanRejectHook(t *testing.T) {
	plan := hooks.Plan[hookstage.Entrypoint]{
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

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	require.NoError(t, err, "Unexpected error creating http request: %s", err)

	stRes, newBody, reject := ExecuteEntrypointStage(&invocation.InvocationContext{}, plan, req, body)
	require.NotNil(t, reject, "Unexpected successful execution of entrypoint hook")
	require.Equal(t, reject, &RejectError{}, "Unexpected reject returned from entrypoint hook")
	assert.Len(t, stRes.GroupsResults, 2, "some hook groups have not been processed")
	assert.Equal(t, body, newBody, "request body shouldn't change if request rejected")
}

type mockRejectEntrypointHook struct{}

func (e mockRejectEntrypointHook) HandleEntrypointHook(_ context.Context, _ *invocation.ModuleContext, _ hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	return invocation.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
}

func TestExecuteEntrypointStage_CanTimeoutOneOfHooks(t *testing.T) {
	plan := hooks.Plan[hookstage.Entrypoint]{
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

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	stRes, newBody, reject := ExecuteEntrypointStage(&invocation.InvocationContext{}, plan, req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	if len(stRes.GroupsResults) != 2 {
		t.Error("some hook groups have not been processed")
	}

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

func (e mockTimeoutEntrypointHook) HandleEntrypointHook(_ context.Context, _ *invocation.ModuleContext, _ hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := &invocation.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("bar", "foo")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, invocation.MutationUpdate, "param", "bar")

	return invocation.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

func TestExecuteEntrypointStage_ModuleContextsAreCreated(t *testing.T) {
	plan := hooks.Plan[hookstage.Entrypoint]{
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

	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	iCtx := invocation.InvocationContext{}
	stRes, _, reject := ExecuteEntrypointStage(&iCtx, plan, req, body)
	require.Nil(t, reject, "Unexpected stage reject")

	if len(stRes.GroupsResults) != 2 {
		t.Error("some hook groups have not been processed")
	}

	ctx1 := iCtx.ModuleContextFor("module-1")
	if ctx1.Ctx["some-ctx-1"] != "some-ctx-1" {
		t.Error("context for module-1 not created")
	}

	ctx2 := iCtx.ModuleContextFor("module-2")
	if ctx2.Ctx["some-ctx-2"] != "some-ctx-2" {
		t.Error("context for module-2 not created")
	}
}

type mockModuleContextEntrypointHook1 struct{}

func (e mockModuleContextEntrypointHook1) HandleEntrypointHook(_ context.Context, mctx *invocation.ModuleContext, _ hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-1": "some-ctx-1"}
	return invocation.HookResult[hookstage.EntrypointPayload]{}, nil
}

type mockModuleContextEntrypointHook2 struct{}

func (e mockModuleContextEntrypointHook2) HandleEntrypointHook(_ context.Context, mctx *invocation.ModuleContext, _ hookstage.EntrypointPayload) (invocation.HookResult[hookstage.EntrypointPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-2": "some-ctx-2"}
	return invocation.HookResult[hookstage.EntrypointPayload]{}, nil
}
