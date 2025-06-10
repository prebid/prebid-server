package hookexecution

import (
	"context"
	"errors"
	"time"

	"github.com/prebid/prebid-server/v3/hooks/hookstage"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
)

type mockUpdateHeaderEntrypointHook struct{}

func (e mockUpdateHeaderEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		payload.Request.Header.Add("foo", "bar")
		return payload, nil
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

type mockUpdateQueryEntrypointHook struct{}

func (e mockUpdateQueryEntrypointHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	c := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
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
	c := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
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
	c := hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(
		func(_ hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			return []byte(`{"name": "John", "last_name": "Doe", "foo": "bar"}`), nil
		}, hookstage.MutationUpdate, "body", "foo",
	).AddMutation(
		func(_ hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
			return []byte(`{"last_name": "Doe", "foo": "bar"}`), nil
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

func (e mockRejectHook) HandleBidderRequestHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	return hookstage.HookResult[hookstage.BidderRequestPayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleRawBidderResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawBidderResponsePayload) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	return hookstage.HookResult[hookstage.RawBidderResponsePayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{Reject: true}, nil
}

func (e mockRejectHook) HandleAuctionResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{Reject: true}, nil
}

type mockTimeoutHook struct{}

func (e mockTimeoutHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	time.Sleep(20 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	c.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		params := payload.Request.URL.Query()
		params.Add("bar", "foo")
		payload.Request.URL.RawQuery = params.Encode()
		return payload, nil
	}, hookstage.MutationUpdate, "param", "bar")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	time.Sleep(20 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	c.AddMutation(func(_ hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
		return []byte(`{"last_name": "Doe", "foo": "bar", "address": "A st."}`), nil
	}, hookstage.MutationUpdate, "param", "address")

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleProcessedAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.ProcessedAuctionRequestPayload) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	time.Sleep(20 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.ProcessedAuctionRequestPayload]{}
	c.AddMutation(func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
		payload.Request.User.CustomData = "some-custom-data"
		return payload, nil
	}, hookstage.MutationUpdate, "bidRequest", "user.customData")

	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleBidderRequestHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	time.Sleep(20 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.BidderRequestPayload]{}
	c.AddMutation(func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
		payload.Request.User.CustomData = "some-custom-data"
		return payload, nil
	}, hookstage.MutationUpdate, "bidRequest", "user.customData")

	return hookstage.HookResult[hookstage.BidderRequestPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleRawBidderResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawBidderResponsePayload) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	time.Sleep(20 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.RawBidderResponsePayload]{}
	c.AddMutation(func(payload hookstage.RawBidderResponsePayload) (hookstage.RawBidderResponsePayload, error) {
		payload.BidderResponse.Bids[0].BidMeta = &openrtb_ext.ExtBidPrebidMeta{AdapterCode: "new-code"}
		return payload, nil
	}, hookstage.MutationUpdate, "bidderResponse", "bidMeta.AdapterCode")

	return hookstage.HookResult[hookstage.RawBidderResponsePayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	time.Sleep(20 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.AllProcessedBidResponsesPayload]{}
	c.AddMutation(func(payload hookstage.AllProcessedBidResponsesPayload) (hookstage.AllProcessedBidResponsesPayload, error) {
		payload.Responses["some-bidder"].Bids[0].BidMeta = &openrtb_ext.ExtBidPrebidMeta{AdapterCode: "new-code"}
		return payload, nil
	}, hookstage.MutationUpdate, "processedBidderResponse", "bidMeta.AdapterCode")

	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ChangeSet: c}, nil
}

func (e mockTimeoutHook) HandleAuctionResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	time.Sleep(2 * time.Millisecond)
	c := hookstage.ChangeSet[hookstage.AuctionResponsePayload]{}
	c.AddMutation(func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
		payload.BidResponse.Ext = []byte("another-byte")
		return payload, nil
	}, hookstage.MutationUpdate, "auctionResponse", "bidResponse.Ext")

	return hookstage.HookResult[hookstage.AuctionResponsePayload]{ChangeSet: c}, nil
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

func (e mockModuleContextHook) HandleBidderRequestHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.BidderRequestPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

func (e mockModuleContextHook) HandleRawBidderResponseHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawBidderResponsePayload) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.RawBidderResponsePayload]{ModuleContext: miCtx.ModuleContext}, nil
}

func (e mockModuleContextHook) HandleAllProcessedBidResponsesHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ModuleContext: miCtx.ModuleContext}, nil
}

func (e mockModuleContextHook) HandleAuctionResponseHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	miCtx.ModuleContext = map[string]interface{}{e.key: e.val}
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{ModuleContext: miCtx.ModuleContext}, nil
}

type mockFailureHook struct{}

func (h mockFailureHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, FailureError{Message: "attribute not found"}
}

func (h mockFailureHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, FailureError{Message: "attribute not found"}
}

func (h mockFailureHook) HandleBidderRequestHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	return hookstage.HookResult[hookstage.BidderRequestPayload]{}, FailureError{Message: "attribute not found"}
}

func (h mockFailureHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}, FailureError{Message: "attribute not found"}
}

type mockErrorHook struct{}

func (h mockErrorHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	return hookstage.HookResult[hookstage.EntrypointPayload]{}, errors.New("unexpected error")
}

func (h mockErrorHook) HandleRawAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{}, errors.New("unexpected error")
}

func (h mockErrorHook) HandleBidderRequestHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	return hookstage.HookResult[hookstage.BidderRequestPayload]{}, errors.New("unexpected error")
}

func (h mockErrorHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{}, errors.New("unexpected error")
}

func (h mockErrorHook) HandleAuctionResponseHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	return hookstage.HookResult[hookstage.AuctionResponsePayload]{}, errors.New("unexpected error")
}

type mockFailedMutationHook struct{}

func (h mockFailedMutationHook) HandleEntrypointHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.EntrypointPayload) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	changeSet := hookstage.ChangeSet[hookstage.EntrypointPayload]{}
	changeSet.AddMutation(func(payload hookstage.EntrypointPayload) (hookstage.EntrypointPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.EntrypointPayload]{ChangeSet: changeSet}, nil
}

func (h mockFailedMutationHook) HandleRawAuctionHook(_ context.Context, miCtx hookstage.ModuleInvocationContext, _ hookstage.RawAuctionRequestPayload) (hookstage.HookResult[hookstage.RawAuctionRequestPayload], error) {
	changeSet := hookstage.ChangeSet[hookstage.RawAuctionRequestPayload]{}
	changeSet.AddMutation(func(payload hookstage.RawAuctionRequestPayload) (hookstage.RawAuctionRequestPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.RawAuctionRequestPayload]{ChangeSet: changeSet}, nil
}

func (h mockFailedMutationHook) HandleBidderRequestHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	changeSet := hookstage.ChangeSet[hookstage.BidderRequestPayload]{}
	changeSet.AddMutation(func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "header", "foo")

	return hookstage.HookResult[hookstage.BidderRequestPayload]{ChangeSet: changeSet}, nil
}

func (h mockFailedMutationHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	changeSet := hookstage.ChangeSet[hookstage.AllProcessedBidResponsesPayload]{}
	changeSet.AddMutation(func(payload hookstage.AllProcessedBidResponsesPayload) (hookstage.AllProcessedBidResponsesPayload, error) {
		return payload, errors.New("key not found")
	}, hookstage.MutationUpdate, "some-bidder", "bids[0]", "deal_priority")

	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ChangeSet: changeSet}, nil
}

type mockUpdateBidRequestHook struct{}

func (e mockUpdateBidRequestHook) HandleProcessedAuctionHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.ProcessedAuctionRequestPayload) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	c := hookstage.ChangeSet[hookstage.ProcessedAuctionRequestPayload]{}
	c.AddMutation(
		func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
			payload.Request.User.Yob = 2000
			userExt, err := payload.Request.GetUserExt()
			if err != nil {
				return payload, err
			}
			newPrebidExt := &openrtb_ext.ExtUserPrebid{
				BuyerUIDs: map[string]string{"some": "id"},
			}
			userExt.SetPrebid(newPrebidExt)
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "user.yob",
	).AddMutation(
		func(payload hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
			payload.Request.User.Consent = "true"
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "user.consent",
	)

	return hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]{ChangeSet: c}, nil
}

func (e mockUpdateBidRequestHook) HandleBidderRequestHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.BidderRequestPayload) (hookstage.HookResult[hookstage.BidderRequestPayload], error) {
	c := hookstage.ChangeSet[hookstage.BidderRequestPayload]{}
	c.AddMutation(
		func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
			user := ptrutil.Clone(payload.Request.User)
			user.Yob = 2000
			payload.Request.User = user
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "user.yob",
	).AddMutation(
		func(payload hookstage.BidderRequestPayload) (hookstage.BidderRequestPayload, error) {
			user := ptrutil.Clone(payload.Request.User)
			user.Consent = "true"
			payload.Request.User = user
			return payload, nil
		}, hookstage.MutationUpdate, "bidRequest", "user.consent",
	)

	return hookstage.HookResult[hookstage.BidderRequestPayload]{ChangeSet: c}, nil
}

type mockUpdateBidderResponseHook struct{}

func (e mockUpdateBidderResponseHook) HandleRawBidderResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.RawBidderResponsePayload) (hookstage.HookResult[hookstage.RawBidderResponsePayload], error) {
	c := hookstage.ChangeSet[hookstage.RawBidderResponsePayload]{}
	c.AddMutation(
		func(payload hookstage.RawBidderResponsePayload) (hookstage.RawBidderResponsePayload, error) {
			payload.BidderResponse.Bids[0].DealPriority = 10
			return payload, nil
		}, hookstage.MutationUpdate, "bidderResponse", "bid.deal-priority",
	)

	return hookstage.HookResult[hookstage.RawBidderResponsePayload]{ChangeSet: c}, nil
}

type mockUpdateBiddersResponsesHook struct{}

func (e mockUpdateBiddersResponsesHook) HandleAllProcessedBidResponsesHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AllProcessedBidResponsesPayload) (hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload], error) {
	c := hookstage.ChangeSet[hookstage.AllProcessedBidResponsesPayload]{}
	c.AddMutation(
		func(payload hookstage.AllProcessedBidResponsesPayload) (hookstage.AllProcessedBidResponsesPayload, error) {
			payload.Responses["some-bidder"].Bids[0].DealPriority = 10
			return payload, nil
		}, hookstage.MutationUpdate, "processedBidderResponse", "bid.deal-priority",
	)

	return hookstage.HookResult[hookstage.AllProcessedBidResponsesPayload]{ChangeSet: c}, nil
}

type mockUpdateBidResponseHook struct{}

func (e mockUpdateBidResponseHook) HandleAuctionResponseHook(_ context.Context, _ hookstage.ModuleInvocationContext, _ hookstage.AuctionResponsePayload) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	c := hookstage.ChangeSet[hookstage.AuctionResponsePayload]{}
	c.AddMutation(
		func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			payload.BidResponse.CustomData = "new-custom-data"
			return payload, nil
		}, hookstage.MutationUpdate, "auctionResponse", "bidResponse.custom-data")

	return hookstage.HookResult[hookstage.AuctionResponsePayload]{ChangeSet: c}, nil
}
