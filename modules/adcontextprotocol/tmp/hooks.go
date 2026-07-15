package tmp

import (
	"context"

	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/util/iterutil"
	"github.com/tidwall/sjson"
)

// HandleEntrypointHook allocates the per-auction async request holder along
// with a cancelable context that survives across hook stages. The response
// hook cancels it on the way out so an in-flight fan-out does not outlive the
// auction.
func (m *Module) HandleEntrypointHook(
	ctx context.Context,
	_ hookstage.ModuleInvocationContext,
	_ hookstage.EntrypointPayload,
) (hookstage.HookResult[hookstage.EntrypointPayload], error) {
	fanoutCtx, cancel := context.WithCancel(ctx)
	moduleContext := hookstage.NewModuleContext()
	moduleContext.Set(asyncKey, &asyncRequest{
		done:   make(chan struct{}),
		ctx:    fanoutCtx,
		cancel: cancel,
	})
	return hookstage.HookResult[hookstage.EntrypointPayload]{ModuleContext: moduleContext}, nil
}

// HandleProcessedAuctionHook kicks off the TMP fan-out in the background. The
// auction continues immediately; results are collected in the response hook.
func (m *Module) HandleProcessedAuctionHook(
	_ context.Context,
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
		async.result = m.fanOut(async.ctx, bidRequest)
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
	// Cancelling here releases the fan-out goroutine if it is still running
	// past the response window — no orphan goroutines beyond the auction.
	defer async.cancel()
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
			payload.BidResponse.Ext = ext

			// Per-bid targeting is where GAM et al actually read keys, so
			// mirror the response-level segments onto each bid's ext when
			// enabled.
			if addToTargeting {
				for seatBid := range iterutil.SlicePointerValues(payload.BidResponse.SeatBid) {
					for bid := range iterutil.SlicePointerValues(seatBid.Bid) {
						bidExt := bid.Ext
						for _, s := range segments {
							kv := splitKV(s)
							if kv == nil {
								continue
							}
							updated, err := sjson.SetBytes(bidExt, "prebid.targeting."+kv[0], kv[1])
							if err != nil {
								logger.Errorf("adcontextprotocol.tmp: bid targeting set: %v", err)
								continue
							}
							bidExt = updated
						}
						bid.Ext = bidExt
					}
				}
			}
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

// splitKV splits "key=value" on the first '=' and rejects empty keys.
func splitKV(s string) []string {
	for i := range len(s) {
		if s[i] == '=' {
			if i == 0 {
				return nil
			}
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
