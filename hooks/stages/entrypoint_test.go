package stages

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/hooks/invocation"
)

func TestExecuteEntrypointHook_CannotModifyPayloadInsideHookCallMethod(t *testing.T) {
	entrypointHooks = []EntrypointHook{mockNopEntrypointHook{}, mockNopEntrypointHook{}}

	ctx := context.TODO()
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	newBody, err := ExecuteEntrypointHook(ctx, invocation.Context{Endpoint: "auction", Timeout: time.Second}, req, body)
	if err != nil {
		t.Fatalf("Unexpected error executing entrypoint hook: %s", err)
	}

	if bytes.Compare(body, newBody) != 0 {
		t.Errorf("Body changed inside hook.Call method, expected: %s, got: %s", string(body), string(newBody))
	}
}

type mockNopEntrypointHook struct{}

func (e mockNopEntrypointHook) Code() string {
	return "MockedNopEntrypointHook"
}

func (e mockNopEntrypointHook) Call(ctx context.Context, invocationContext invocation.Context, payload ImmutableEntrypointPayload) (invocation.HookResult[MutableEntrypointPayload], error) {
	_ = jsonparser.Delete(payload.Body, "name")
	_, _ = jsonparser.Set(payload.Body, []byte(`"bar"`), "foo")

	return invocation.HookResult[MutableEntrypointPayload]{Action: invocation.Nop}, nil
}

func TestExecuteEntrypointHook_CanApplyHookMutations(t *testing.T) {
	entrypointHooks = []EntrypointHook{mockUpdateEntrypointHook{}, mockUpdateEntrypointHook{}}

	ctx := context.TODO()
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	newBody, err := ExecuteEntrypointHook(ctx, invocation.Context{Endpoint: "auction", Timeout: time.Second}, req, body)
	if err != nil {
		t.Fatalf("Unexpected error executing entrypoint hook: %s", err)
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
		t.Errorf("query params not changed inside hook.Call method")
	}
}

type mockUpdateEntrypointHook struct{}

func (e mockUpdateEntrypointHook) Code() string {
	return "MockedUpdateEntrypointHook"
}

func (e mockUpdateEntrypointHook) Call(_ context.Context, _ invocation.Context, payload ImmutableEntrypointPayload) (invocation.HookResult[MutableEntrypointPayload], error) {
	muts := []invocation.Mutation[MutableEntrypointPayload]{
		invocation.NewMutation(func(payload MutableEntrypointPayload) (MutableEntrypointPayload, error) {
			payload.Header.Add("foo", "bar")
			return payload, nil
		}, invocation.MutationUpdate, "header", "foo"),

		invocation.NewMutation(func(payload MutableEntrypointPayload) (MutableEntrypointPayload, error) {
			payload.Params.Add("foo", "baz")
			return payload, nil
		}, invocation.MutationUpdate, "param", "foo"),

		invocation.NewMutation(func(payload MutableEntrypointPayload) (MutableEntrypointPayload, error) {
			body, err := jsonparser.Set(payload.Body, []byte(`"bar"`), "foo")
			if err != nil {
				return payload, fmt.Errorf("failed to set body foo key: %s", err)
			}
			payload.Body = body

			return payload, nil
		}, invocation.MutationUpdate, "body", "foo"),

		invocation.NewMutation(func(payload MutableEntrypointPayload) (MutableEntrypointPayload, error) {
			payload.Body = jsonparser.Delete(payload.Body, "name")
			return payload, nil
		}, invocation.MutationDelete, "body", "name"),
	}

	return invocation.HookResult[MutableEntrypointPayload]{Action: invocation.Update, Mutations: muts}, nil
}

func TestExecuteEntrypointHook_CanRejectHook(t *testing.T) {
	entrypointHooks = []EntrypointHook{mockRejectEntrypointHook{}, mockUpdateEntrypointHook{}}

	ctx := context.TODO()
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	newBody, err := ExecuteEntrypointHook(ctx, invocation.Context{Endpoint: "auction", Timeout: time.Second}, req, body)
	if err != nil {
		t.Fatalf("Unexpected error executing entrypoint hook: %s", err)
	}

	if bytes.Compare(body, newBody) != 0 {
		t.Error("request body shouldn't change if request rejected")
	}

	if req.Header.Get("foo") != "" {
		t.Error("headers shouldn't change if request rejected")
	}

	if req.URL.Query().Get("foo") != "" {
		t.Errorf("query params shouldn't change if request rejected")
	}
}

type mockRejectEntrypointHook struct{}

func (e mockRejectEntrypointHook) Code() string {
	return "MockedRejectEntrypointHook"
}

func (e mockRejectEntrypointHook) Call(_ context.Context, _ invocation.Context, _ ImmutableEntrypointPayload) (invocation.HookResult[MutableEntrypointPayload], error) {
	return invocation.HookResult[MutableEntrypointPayload]{Action: invocation.Reject}, nil
}
