package tmp

import (
	"context"

	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/tidwall/sjson"
)

// HandleEntrypointHook allocates the per-auction async request holder.
func (m *Module) HandleEntrypointHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	moduleContext := hookstage.NewModuleContext()
	moduleContext.Set(asyncKey, &asyncRequest{done: make(chan struct{})})
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: moduleContext}, nil
}

// HandleProcessedAuctionHook kicks off the TMP fan-out in the background. The
// auction continues immediately; results are collected in the response hook.
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	var res hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]

	async, ok := m.loadAsync(miCtx)
	if !ok {
		return res, nil
	}
	if payload.Request == nil || payload.Request.BidRequest == nil {
		close(async.done)
		return res, nil
	}
	bidRequest := payload.Request.BidRequest

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("adcontextprotocol.tmp: panic in fan-out: %v", r)
			}
			close(async.done)
		}()
		async.result = m.fanOut(ctx, bidRequest)
	}()
	return res, nil
}

// HandleAuctionResponseHook joins fan-out results with the bid response.
func (m *Module) HandleAuctionResponseHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	_ hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	var res hookstage.HookResult[hookstage.AuctionResponsePayload]

	async, ok := m.loadAsync(miCtx)
	if !ok {
		return res, nil
	}
	select {
	case <-async.done:
	case <-ctx.Done():
		return res, nil
	}
	if async.result == nil || len(async.result.Segments) == 0 {
		return res, nil
	}
	segments := async.result.Segments
	targetingKey := m.cfg.TargetingKey
	addToTargeting := m.cfg.AddToTargeting

	res.ChangeSet.AddMutation(
		func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			ext := payload.BidResponse.Ext
			newExt, err := sjson.SetBytes(ext, targetingKey+".segments", segments)
			if err != nil {
				logger.Errorf("adcontextprotocol.tmp: failed to set %s.segments on response ext: %v", targetingKey, err)
			} else {
				ext = newExt
			}
			if addToTargeting {
				for _, s := range segments {
					kv := splitKV(s)
					if kv == nil {
						continue
					}
					newExt, err := sjson.SetBytes(ext, "prebid.targeting."+kv[0], kv[1])
					if err != nil {
						logger.Errorf("adcontextprotocol.tmp: targeting set: %v", err)
						continue
					}
					ext = newExt
				}
			}
			payload.BidResponse.Ext = ext
			return payload, nil
		},
		hookstage.MutationUpdate,
		"ext",
	)

	res.AnalyticsTags = hookanalytics.Analytics{
		Activities: []hookanalytics.Activity{{
			Name:   "adcontextprotocol.tmp.fanout",
			Status: hookanalytics.ActivityStatusSuccess,
			Results: []hookanalytics.Result{{
				Status: hookanalytics.ResultStatusAllow,
				Values: map[string]any{"segments": len(segments)},
			}},
		}},
	}
	return res, nil
}

func (m *Module) loadAsync(miCtx hookstage.ModuleInvocationContext) (*asyncRequest, bool) {
	v, ok := miCtx.ModuleContext.Get(asyncKey)
	if !ok {
		return nil, false
	}
	a, ok := v.(*asyncRequest)
	return a, ok
}

// splitKV splits "key=value" once; returns nil for malformed input.
func splitKV(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
