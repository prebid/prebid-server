package foobar

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/prebid/prebid-server/hooks/invocation"
	"github.com/prebid/prebid-server/hooks/stages"
	"github.com/prebid/prebid-server/modules/acme/foobar/config"
	moduleHooks "github.com/prebid/prebid-server/modules/acme/foobar/hooks"
)

func Builder(conf json.RawMessage, client *http.Client) (interface{}, error) {
	cfg, err := config.New(conf)
	if err != nil {
		return nil, err
	}

	return Module{
		entrypointHook: moduleHooks.NewEntrypointHook(cfg),
		rawAuctionHook: moduleHooks.NewRawAuctionHook(client, cfg),
	}, nil
}

type Module struct {
	entrypointHook moduleHooks.EntrypointHook
	rawAuctionHook moduleHooks.RawAuctionHook
}

func (m Module) HandleEntrypointHook(ctx context.Context, context invocation.Context, payload stages.EntrypointPayload) (invocation.HookResult[stages.EntrypointPayload], error) {
	return m.entrypointHook.Handle(ctx, context, payload)
}

func (m Module) HandleRawAuctionHook(ctx context.Context, context invocation.Context, request stages.BidRequest) (invocation.HookResult[stages.BidRequest], error) {
	return m.rawAuctionHook.Handle(ctx, context, request)
}
