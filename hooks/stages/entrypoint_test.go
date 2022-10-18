package stages

import (
	"bytes"
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/buger/jsonparser"
	"github.com/prebid/prebid-server/hooks/invocation"
)

func TestExecuteEntrypointStage_CanApplyHookMutations(t *testing.T) {
	entrypointHooks = []EntrypointHook{mockUpdateEntrypointHook{}, mockUpdateEntrypointHook{}}

	ctx := context.Background()
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	newBody, err := ExecuteEntrypointStage(ctx, invocation.Context{Endpoint: "auction", Timeout: time.Second}, req, body)
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
		t.Error("header not changed inside hook.HandleEntrypointHook method")
	}

	if req.URL.Query().Get("foo") == "" {
		t.Errorf("query params not changed inside hook.HandleEntrypointHook method")
	}
}

type mockUpdateEntrypointHook struct{}

func (e mockUpdateEntrypointHook) Code() string {
	return "MockedUpdateEntrypointHook"
}

func (e mockUpdateEntrypointHook) HandleEntrypointHook(_ context.Context, _ invocation.Context, payload EntrypointPayload) (invocation.HookResult[EntrypointPayload], error) {
	muts := []invocation.Mutation[EntrypointPayload]{
		invocation.NewMutation(func(payload EntrypointPayload) (EntrypointPayload, error) {
			payload.Request.Header.Add("foo", "bar")
			return payload, nil
		}, invocation.MutationUpdate, "header", "foo"),

		invocation.NewMutation(func(payload EntrypointPayload) (EntrypointPayload, error) {
			params := payload.Request.URL.Query()
			params.Add("foo", "baz")
			payload.Request.URL.RawQuery = params.Encode()
			return payload, nil
		}, invocation.MutationUpdate, "param", "foo"),

		invocation.NewMutation(func(payload EntrypointPayload) (EntrypointPayload, error) {
			payload.Body = []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, invocation.MutationUpdate, "body", "foo"),

		invocation.NewMutation(func(payload EntrypointPayload) (EntrypointPayload, error) {
			payload.Body = []byte(`{"last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, invocation.MutationDelete, "body", "name"),
	}

	return invocation.HookResult[EntrypointPayload]{Mutations: muts}, nil
}

func TestExecuteEntrypointStage_CanRejectHook(t *testing.T) {
	entrypointHooks = []EntrypointHook{mockRejectEntrypointHook{}, mockUpdateEntrypointHook{}}

	ctx := context.Background()
	body := []byte(`{"name": "John", "last_name": "Doe"}`)
	reader := bytes.NewReader(body)
	req, err := http.NewRequest(http.MethodPost, "https://prebid.com/openrtb2/auction", reader)
	if err != nil {
		t.Fatalf("Unexpected error creating http request: %s", err)
	}

	newBody, err := ExecuteEntrypointStage(ctx, invocation.Context{Endpoint: "auction", Timeout: time.Second}, req, body)
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

func (e mockRejectEntrypointHook) HandleEntrypointHook(_ context.Context, _ invocation.Context, _ EntrypointPayload) (invocation.HookResult[EntrypointPayload], error) {
	return invocation.HookResult[EntrypointPayload]{Reject: true}, nil
}
