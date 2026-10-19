package identity

import (
	"context"
	"time"

	iiqenrichment "github.com/Intent-IQ/identity-go/enrichment"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
)

// HandleProcessedAuctionHook enriches the request and deliberately fails open.
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	invocation hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], _ error) {
	start := time.Now()
	config := m.config.resolve(invocation.AccountConfig)
	var auction *openrtb2.BidRequest
	if payload.Request != nil {
		auction = payload.Request.BidRequest
	}

	tr := &flowTrace{collect: config.TraceEnabled && requestTraceOptIn(auction)}
	auctionCtx := newAuctionContext(auction)
	auctionCtx.Start = start
	defer func() {
		auctionCtx.TraceEnabled = tr.enabled()
		auctionCtx.Trace = tr.msgs
		result.ModuleContext = storeAuctionContext(auctionCtx)
		result.DebugMessages = tr.msgs
	}()

	if tr.enabled() {
		tr.tracef("[enrich] start — partner=%s, cache=%s; request signals: %s",
			config.PartnerID, onOff(config.Cache.Enabled), requestSignals(auction))
	}

	resolved, err := m.enricher.Enrich(ctx, iiqenrichment.Request{
		PartnerID: config.PartnerID, Endpoint: config.APIEndpoint, Auction: auction,
		Timeout: config.timeout(), WaitTimeout: config.waitTimeout(), CacheEnabled: config.Cache.Enabled,
	})
	if err != nil {
		logger.Debugf("intentiq identity: enrichment skipped: %v", err)
		if tr.enabled() {
			tr.tracef("[enrich] resolution error (fail-open, request unchanged): %v; enrich hook took %s total",
				err, fmtDur(time.Since(start)))
		}
		return result, nil
	}
	if resolved.Outcome == iiqenrichment.OutcomeNoEndpoint {
		if tr.enabled() {
			tr.tracef("[enrich] skipped: no api_endpoint configured (no-op); enrich hook took %s total", fmtDur(time.Since(start)))
		}
		return result, nil
	}
	if auction == nil {
		if tr.enabled() {
			tr.tracef("[enrich] skipped: nil bid request; enrich hook took %s total", fmtDur(time.Since(start)))
		}
		return result, nil
	}

	auctionCtx.ABTestUUID = resolved.ABTestUUID
	auctionCtx.TerminationCause = resolved.TerminationCause
	if tr.enabled() {
		tr.tracef("[enrich] outcome=%s via %s — eids=%d, tc=%s, abTestUuid=%s",
			resolved.Outcome, resultSource(resolved), len(resolved.EIDs),
			tcStr(resolved.TerminationCause), abTestShort(resolved.ABTestUUID))
	}
	if len(resolved.EIDs) == 0 {
		if tr.enabled() {
			tr.tracef("[enrich] result: 0 eids resolved (reason=%s) → user.eids unchanged; enrich hook took %s total",
				resolved.Outcome, fmtDur(time.Since(start)))
		}
		return result, nil
	}
	if tr.enabled() {
		tr.tracef("[enrich] result: enriched user.eids += %d eid(s) [%s]; enrich hook took %s total",
			len(resolved.EIDs), eidsDetail(resolved.EIDs), fmtDur(time.Since(start)))
	}

	eids := append([]openrtb2.EID(nil), resolved.EIDs...)
	result.ChangeSet.AddMutation(func(value hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
		if value.Request == nil || value.Request.BidRequest == nil {
			return value, nil
		}
		if value.Request.BidRequest.User == nil {
			value.Request.BidRequest.User = &openrtb2.User{}
		}
		value.Request.BidRequest.User.EIDs = append(value.Request.BidRequest.User.EIDs, eids...)
		return value, nil
	}, hookstage.MutationUpdate, "bidrequest", "user", "eids")
	return result, nil
}
