package hookexecution

import (
	"context"
	"time"

	"github.com/prebid/prebid-server/hooks/hookstage"
)

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

type mockUpdateBodyHook struct{}

func (e mockUpdateBodyHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
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

func (e mockUpdateBodyHook) HandleRawAuctionHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.RawAuctionPayload) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
	c := &hookstage.ChangeSet[hookstage.RawAuctionPayload]{}
	c.AddMutation(
		func(payload hookstage.RawAuctionPayload) (hookstage.RawAuctionPayload, error) {
			payload = []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationUpdate, "body", "foo",
	).AddMutation(
		func(payload hookstage.RawAuctionPayload) (hookstage.RawAuctionPayload, error) {
			payload = []byte(`{"last_name": "Doe", "foo": "bar"}`)
			return payload, nil
		}, hookstage.MutationDelete, "body", "name",
	)

	return hookstage.HookResult[hookstage.RawAuctionPayload]{ChangeSet: c}, nil
}

type mockRejectHook struct{}

func (e mockRejectHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleRawAuctionHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.RawAuctionPayload) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionPayload]{Reject: true}, nil
}

type mockTimeoutHook struct{}

func (e mockTimeoutHook) HandleEntrypointHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
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

func (e mockTimeoutHook) HandleRawAuctionHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.RawAuctionPayload) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.RawAuctionPayload]{}
	c.AddMutation(func(payload hookstage.RawAuctionPayload) (hookstage.RawAuctionPayload, error) {
		payload = []byte(`{"last_name": "Doe", "foo": "bar", "address": "A st."}`)
		return payload, nil
	}, hookstage.MutationUpdate, "param", "address")

	return hookstage.HookResult[hookstage.RawAuctionPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleAuctionResponseHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := &hookstage.ChangeSet[hookstage.AuctionResponsePayload]{}
	c.AddMutation(func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
		payload.BidResponse.Ext = []byte("another-byte")
		return payload, nil
	}, hookstage.MutationUpdate, "auctionResponse", "bidResponse.Ext")

	return hookstage.HookResult[hookstage.AuctionResponsePayload]{ChangeSet: c}, nil
}

type mockModuleContextHook1 struct{}

func (e mockModuleContextHook1) HandleEntrypointHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-1": "some-ctx-1"}
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

func (e mockModuleContextHook1) HandleRawAuctionHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.RawAuctionPayload) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-1": "some-ctx-1"}
	return hookstage.HookResult[hookstage.RawAuctionPayload]{}, nil
}

func (e mockModuleContextHook1) HandleAuctionResponseHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-1": "some-ctx-1"}
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, nil
}

type mockModuleContextHook2 struct{}

func (e mockModuleContextHook2) HandleEntrypointHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-2": "some-ctx-2"}
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, nil
}

func (e mockModuleContextHook2) HandleRawAuctionHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.RawAuctionPayload) (hookstage.HookResult[hookstage.RawAuctionPayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-2": "some-ctx-2"}
	return hookstage.HookResult[hookstage.RawAuctionPayload]{}, nil
}

func (e mockModuleContextHook2) HandleAuctionResponseHook(_ context.Context, mctx *hookstage.ModuleContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	mctx.Ctx = map[string]interface{}{"some-ctx-2": "some-ctx-2"}
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, nil
}

type mockUpdateBidResponseHook struct{}

func (e mockUpdateBidResponseHook) HandleAuctionResponseHook(_ context.Context, _ *hookstage.ModuleContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	c := &hookstage.ChangeSet[hookstage.AuctionResponsePayload]{}
	c.AddMutation(
		func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			payload.BidResponse.CustomData = payload.BidResponse.CustomData + "more-data"
			return payload, nil
		}, hookstage.MutationUpdate, "auctionResponse", "bidResponse.custom-data")

	return hookstage.HookResult[hookstage.AuctionResponsePayload]{ChangeSet: c}, nil
}
