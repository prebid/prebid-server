package tmp

import (
	"context"

	"github.com/prebid/prebid-server/v4/hooks/hookanalytics"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/util/iterutil"
	"github.com/tidwall/sjson"
)

// HandleProcessedAuctionHook snapshots the relevant fields from the
// live BidRequest synchronously, allocates the async result holder,
// then kicks off the TMP fan-out in a background goroutine. The
// goroutine never touches the BidRequest again — deriveInputs runs on
// the caller's stack so there is no data race with concurrent hook
// stages / privacy scrubbing / other modules that continue to mutate
// the request wrapper after this hook returns.
//
// The fan-out context is rooted in context.Background(), NOT the hook
// caller's context: the framework cancels every hook's own ctx the
// moment the hook returns (hooks/hookexecution/execution.go), so a
// derived ctx would be Done before the fan-out started. The response
// hook cancels it via defer async.cancel() when the auction is ready
// to serve.
func (m *Module) HandleProcessedAuctionHook(
	_ context.Context,
	_ hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	var res hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]
	if payload.Request == nil || payload.Request.BidRequest == nil {
		return res, nil
	}

	// Snapshot everything the fan-out needs off the live request BEFORE
	// spawning the goroutine. deriveInputs is pure CPU with no I/O; the
	// snapshot is a value (map[string]any is copied by re-emission at
	// coarseGeo, identities is a fresh []IdentityToken slice, etc.) that
	// the goroutine can hold independently while the auction rebuilds
	// req.Ext / user.ext elsewhere.
	inputs := deriveInputs(&m.cfg, payload.Request.BidRequest)
	if inputs.PlacementID == "" || (inputs.Domain == "" && inputs.Bundle == "") {
		// Nothing to fan out — skip both the holder allocation and the
		// goroutine so the response hook cleanly returns without
		// waiting.
		return res, nil
	}

	fanoutCtx, cancel := context.WithCancel(context.Background())
	async := &asyncRequest{
		done:   make(chan struct{}),
		ctx:    fanoutCtx,
		cancel: cancel,
	}
	moduleCtx := hookstage.NewModuleContext()
	moduleCtx.Set(asyncKey, async)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("adcontextprotocol.tmp: panic in fan-out: %v", r)
			}
			close(async.done)
		}()
		async.result = m.fanOut(fanoutCtx, inputs)
	}()

	res.ModuleContext = moduleCtx
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

	var (
		segments []string
		errCount int
	)
	if async.result != nil {
		segments = async.result.Segments
		errCount = async.result.ErrCount
	}
	if len(segments) == 0 {
		res.AnalyticsTags = analyticsForResult(0, errCount)
		return res, nil
	}
	targetingKey := m.cfg.TargetingKey
	addToTargeting := m.cfg.AddToTargeting

	res.ChangeSet.AddMutation(
		func(payload hookstage.AuctionResponsePayload) (hookstage.AuctionResponsePayload, error) {
			if payload.BidResponse == nil {
				return payload, nil
			}
			ext := payload.BidResponse.Ext
			newExt, err := sjson.SetBytes(ext, targetingKey+".segments", segments)
			if err != nil {
				logger.Errorf("adcontextprotocol.tmp: failed to set %s.segments on response ext: %v", targetingKey, err)
			} else {
				ext = newExt
			}
			payload.BidResponse.Ext = ext

			if !addToTargeting {
				return payload, nil
			}
			// Batch the per-bid targeting update: build the (key,value)
			// pairs once outside the seatbid loop so each bid gets O(1)
			// sjson rewrites instead of O(segments).
			targetingMap := targetingMapFromSegments(segments)
			if len(targetingMap) == 0 {
				return payload, nil
			}
			for seatBid := range iterutil.SlicePointerValues(payload.BidResponse.SeatBid) {
				for bid := range iterutil.SlicePointerValues(seatBid.Bid) {
					updated, err := sjson.SetBytes(bid.Ext, "prebid.targeting", targetingMap)
					if err != nil {
						logger.Errorf("adcontextprotocol.tmp: bid targeting set: %v", err)
						continue
					}
					bid.Ext = updated
				}
			}
			return payload, nil
		},
		hookstage.MutationUpdate,
		"ext",
	)

	res.AnalyticsTags = analyticsForResult(len(segments), errCount)
	return res, nil
}

// analyticsForResult surfaces both success and failure signal so the
// module cannot silently report Success when every provider errored.
func analyticsForResult(segments, errCount int) hookanalytics.Analytics {
	status := hookanalytics.ActivityStatusSuccess
	resultStatus := hookanalytics.ResultStatusAllow
	if segments == 0 && errCount > 0 {
		status = hookanalytics.ActivityStatusError
		resultStatus = hookanalytics.ResultStatusError
	}
	return hookanalytics.Analytics{
		Activities: []hookanalytics.Activity{{
			Name:   "adcontextprotocol.tmp.fanout",
			Status: status,
			Results: []hookanalytics.Result{{
				Status: resultStatus,
				Values: map[string]any{
					"segments":  segments,
					"err_count": errCount,
				},
			}},
		}},
	}
}

// targetingMapFromSegments converts the "key=value" segment slice into
// a flat map suitable for a single sjson.SetBytes into
// ext.prebid.targeting. Duplicate keys keep the last-wins value.
func targetingMapFromSegments(segments []string) map[string]string {
	out := make(map[string]string, len(segments))
	for _, s := range segments {
		kv := splitKV(s)
		if kv == nil {
			continue
		}
		out[kv[0]] = kv[1]
	}
	return out
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
