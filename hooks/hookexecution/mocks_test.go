package hookexecution

import (
	"context"
	"errors"
	"time"

	"github.com/prebid/prebid-server/hooks/hookstage"
)

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

type mockUpdateBodyHook struct{}

func (e mockUpdateBodyHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
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

func (e mockUpdateBodyHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	c := &hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(
		func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			payload = []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationUpdate, "body", "foo",
	).AddMutation(
		func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			payload = []byte(`{"last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationDelete, "body", "name",
	)

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: c}, nil
}

type mockRejectHook struct{}

func (e mockRejectHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleProcessedAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.ProcessedAuctionRequestPayload) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{Reject: true}, nil
}

type mockTimeoutHook struct{}

func (e mockTimeoutHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
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

func (e mockTimeoutHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
		payload = []byte(`{"last_name": "Doe", "foo": "bar", "address": "A st."}`)
		return payload, nil
	}, hookstage.MutationUpdate, "param", "address")

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleProcessedAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.ProcessedAuctionRequestPayload) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.ProcessedAuctionRequestPayload]{}
	c.AddMutation(func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
		payload.BidRequest.User.CustomData = "some-custom-data"
		return payload, nil
	}, hookstage.MutationUpdate, "bidRequest", "user.customData")

	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{ChangeSet: c}, nil
}

type mockModuleContextHook struct {
	key, val string
}

func (e mockModuleContextHook) HandleEntrypointHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

func (e mockModuleContextHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

func (e mockModuleContextHook) HandleProcessedAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.ProcessedAuctionRequestPayload) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

type mockFailureHook struct{}

func (h mockFailureHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, FailureError{Message: "attribute not found"}
}

func (h mockFailureHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, FailureError{Message: "attribute not found"}
}

type mockErrorHook struct{}

func (h mockErrorHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, errors.New("unexpected error")
}

func (h mockErrorHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, errors.New("unexpected error")
}

type mockFailedMutationHook struct{}

func (h mockFailedMutationHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	changeSet := &hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	changeSet.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: changeSet}, nil
}

func (h mockFailedMutationHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	changeSet := &hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	changeSet.AddMutation(func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: changeSet}, nil
}

type mockUpdateBidRequestHook struct{}

func (e mockUpdateBidRequestHook) HandleProcessedAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.ProcessedAuctionRequestPayload) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	c := &hookstage.ChangeSet[hookstage.ProcessedAuctionRequestPayload]{}
	c.AddMutation(
		func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
			payload.BidRequest.User.Yob = 2000
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "user.yob",
	).AddMutation(
		func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
			payload.BidRequest.User.Consent = "true"
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "user.consent",
	)

	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{ChangeSet: c}, nil
}
