package identity

import (
	"context"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/enrichment"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/metrics"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// HandleProcessedAuctionHook enriches user.eids with IntentIQ-resolved ids. Faithful port of the Java
// IntentiqIdentityProcessedAuctionRequestHook: it resolves identity (from the alias cache or a live
// S2S call) and, on a hit, appends the resolved eids to user.eids. It is fail-open — any resolution
// error leaves the request untouched. Flow state for the impression hook is always stashed in the
// returned ModuleContext.
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], err error) {
	start := time.Now()
	cfg := m.cfg.resolve(miCtx.AccountConfig)
	dpi := cfg.PartnerID

	rw := payload.Request
	var req *openrtb2.BidRequest
	if rw != nil {
		req = rw.BidRequest
	}

	tr := &flowTrace{collect: cfg.TraceEnabled && requestTraceOptIn(req)}

	// Baseline flow context: request-derived fields known regardless of the resolution outcome.
	fc := flowContext{start: start}
	if req != nil {
		fc.auctionID = req.ID
		fc.ref = resolveRef(req)
		if device := req.Device; device != nil {
			fc.ip = device.IP
			if fc.ip == "" {
				fc.ip = device.IPv6
			}
			fc.ua = device.UA
		}
	}

	defer func() {
		fc.trace = tr.msgs
		fc.traceEnabled = tr.enabled()
		result.ModuleContext = setFlowContext(fc)
		result.DebugMessages = tr.msgs
	}()

	m.metrics.Requests(dpi)

	cacheOn := m.cache != nil && cfg.Cache.Enabled
	if tr.enabled() {
		tr.tracef("[enrich] start — partner=%s, cache=%s; request signals: %s", dpi, onOff(cacheOn), requestSignals(req))
	}

	if !notBlank(cfg.APIEndpoint) {
		m.metrics.NotEnriched(metrics.ReasonNoEndpoint, dpi)
		if tr.enabled() {
			tr.tracef("[enrich] skipped: no api_endpoint configured (no-op); enrich hook took %s total", fmtDur(time.Since(start)))
		}
		return result, nil
	}
	if req == nil {
		if tr.enabled() {
			tr.tracef("[enrich] skipped: nil bid request; enrich hook took %s total", fmtDur(time.Since(start)))
		}
		return result, nil
	}

	res, resErr := m.resolveEids(ctx, cfg, rw, tr)
	if resErr != nil {
		reason, status := enrichment.ErrorLabels(resErr)
		m.metrics.APIError(dpi, reason, status)
		if tr.enabled() {
			tr.tracef("[enrich] resolution error (fail-open, request unchanged): reason=%s status=%s: %v; enrich hook took %s total",
				reason, orNone(status), resErr, fmtDur(time.Since(start)))
		}
		return result, nil
	}

	fc.abTestUUID = res.abTestUUID
	fc.terminationCause = res.terminationCause

	if len(res.eids) == 0 {
		m.metrics.NotEnriched(res.notEnrichedReason, dpi)
		if tr.enabled() {
			tr.tracef("[enrich] result: 0 eids resolved (reason=%s) → user.eids unchanged; enrich hook took %s total",
				orNone(res.notEnrichedReason), fmtDur(time.Since(start)))
		}
		return result, nil
	}

	m.metrics.Enriched(dpi)
	if tr.enabled() {
		tr.tracef("[enrich] result: enriched user.eids += %d eid(s) [%s]; enrich hook took %s total",
			len(res.eids), eidsDetail(res.eids), fmtDur(time.Since(start)))
	}

	eids := res.eids
	result.ChangeSet.AddMutation(
		func(p hookstage.ProcessedAuctionRequestPayload) (hookstage.ProcessedAuctionRequestPayload, error) {
			enrichUserEids(p.Request, eids)
			return p, nil
		},
		hookstage.MutationUpdate, "bidrequest", "user", "eids",
	)

	return result, nil
}

// resolveEids resolves identity for the request: either directly via the S2S call (caching disabled
// or no candidate keys) or through the two-layer alias cache. Mirrors the Java resolveEids state
// machine, recording the cache business counters as it goes.
func (m *Module) resolveEids(ctx context.Context, cfg Config, rw *openrtb_ext.RequestWrapper, tr *flowTrace) (resolution, error) {
	dpi := cfg.PartnerID
	req := rw.BidRequest

	cacheEnabled := m.cache != nil && cfg.Cache.Enabled
	var keys []cache.Key
	if cacheEnabled {
		keys = m.keyExtractor.CandidateKeys(req)
	}
	if len(keys) == 0 {
		if cacheEnabled {
			tr.tracef("[enrich] no candidate first-party keys derived — direct S2S call")
		} else {
			tr.tracef("[enrich] cache disabled — direct S2S call")
		}

		resp, err := m.fetch(ctx, cfg, rw, tr)
		if err != nil {
			return resolution{}, err
		}

		return resolution{
			eids:              resp.Eids(),
			abTestUUID:        resp.AbTestUUID,
			terminationCause:  resp.Tc,
			notEnrichedReason: metrics.ReasonNoIDs,
		}, nil
	}

	var lookupStart time.Time
	if tr.enabled() {
		tr.tracef("[enrich] cache lookup over %d candidate key(s): [%s]", len(keys), keysDetail(keys))
		lookupStart = time.Now()
	}
	res := m.cache.Get(ctx, keys)

	switch res.State {

	case cache.Hit:
		m.metrics.CacheLookup(metrics.ResultHit, res.Layer.Token(), dpi)

		if tr.enabled() {
			tr.tracef("[enrich] cache HIT (layer=%s, keytype=%s) in %s — %d eid(s), no S2S call: [%s]; tc=%s, abTestUuid=%s",
				res.Layer.Token(), res.KeyType.Token(), fmtDur(time.Since(lookupStart)),
				len(res.Eids), eidsDetail(res.Eids), tcStr(res.Tc), abTestShort(res.AbTestUUID))
		}

		return resolution{
			eids:             res.Eids,
			abTestUUID:       res.AbTestUUID,
			terminationCause: res.Tc,
		}, nil

	case cache.Negative:
		m.metrics.CacheLookup(metrics.ResultMiss, res.Layer.Token(), dpi)

		if tr.enabled() {
			tr.tracef("[enrich] cache NEGATIVE (layer=%s, keytype=%s) in %s — known-unresolvable, no S2S call; tc=%s",
				res.Layer.Token(), res.KeyType.Token(), fmtDur(time.Since(lookupStart)), tcStr(res.Tc))
		}

		return resolution{
			abTestUUID:        res.AbTestUUID,
			terminationCause:  res.Tc,
			notEnrichedReason: metrics.ReasonNoIDsCached,
		}, nil

	case cache.InProgress:
		m.metrics.CacheLookup(metrics.ResultMiss, res.Layer.Token(), dpi)

		if tr.enabled() {
			tr.tracef("[enrich] cache IN-PROGRESS (layer=%s, keytype=%s) in %s — concurrent resolution already running, no S2S call",
				res.Layer.Token(), res.KeyType.Token(), fmtDur(time.Since(lookupStart)))
		}

		return resolution{notEnrichedReason: metrics.ReasonInProgress}, nil

	default: // cache.miss
		m.metrics.CacheLookup(metrics.ResultMiss, res.Layer.Token(), dpi)
		m.cache.PutInProgress(ctx, keys)

		if tr.enabled() {
			tr.tracef("[enrich] cache MISS (keytype=%s) in %s — in-progress marker set, calling IIQ",
				keys[0].Type.Token(), fmtDur(time.Since(lookupStart)))
		}

		resp, err := m.fetch(ctx, cfg, rw, tr)
		if err != nil {
			return resolution{}, err
		}

		eids := resp.Eids()
		if len(eids) > 0 {
			m.cache.Put(ctx, keys, eids, resp.AbTestUUID, resp.Tc, resp.CTTL())

			if tr.enabled() {
				tr.tracef("[enrich] cached %d eid(s) under all keys: [%s]", len(eids), eidsDetail(eids))
			}
		} else {
			m.cache.PutNegative(ctx, keys, resp.AbTestUUID, resp.Tc, resp.CTTL())

			tr.tracef("[enrich] cached negative sentinel under all keys (0 eids) — id unresolvable until TTL expiry")
		}

		return resolution{
			eids:              eids,
			abTestUUID:        resp.AbTestUUID,
			terminationCause:  resp.Tc,
			notEnrichedReason: metrics.ReasonNoIDs,
		}, nil
	}
}

// fetch performs the identity-resolution S2S call under a per-request timeout and records the API
// metrics. The error is left to the caller's fail-open path so it is counted once per failed
// resolution (as in Java).
func (m *Module) fetch(ctx context.Context, cfg Config, rw *openrtb_ext.RequestWrapper, tr *flowTrace) (enrichment.Response, error) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.GetTimeout())
	defer cancel()

	url := resolveURL(cfg, rw)
	consent := resolveConsent(rw)

	if tr.enabled() {
		ip := ""
		if rw.BidRequest != nil && rw.BidRequest.Device != nil {
			if ip = rw.BidRequest.Device.IP; ip == "" {
				ip = rw.BidRequest.Device.IPv6
			}
		}
		tr.tracef("[enrich] → IIQ resolve GET (dpi=%s, ip=%s, gdpr-consent=%s, timeout=%s)",
			cfg.PartnerID, ip, presentAbsent(consent != ""), fmtDur(cfg.GetTimeout()))
		tr.tracef("[enrich]   url=%s", url)
	}

	start := time.Now()
	resp, err := m.api.Get(reqCtx, url, consent)

	dur := time.Since(start)
	m.metrics.APILatency(dur, cfg.PartnerID)

	if err != nil {
		reason, status := enrichment.ErrorLabels(err)
		elapsed, statusStr := fmtDur(dur), orNone(status)

		logger.Warnf("intentiq-identity: resolution failed after %s: reason=%s status=%s: %v",
			elapsed, reason, statusStr, err)
		tr.tracef("[enrich] ← IIQ %s error after %s (status=%s): %v", reason, elapsed, statusStr, err)

		return enrichment.Response{}, err
	}

	m.metrics.APISuccess(cfg.PartnerID)

	if tr.enabled() {
		tr.tracef("[enrich] ← IIQ %d in %s (GET latency) — eids=%d, tc=%s, cttl=%s, abTestUuid=%s",
			resp.Status, fmtDur(dur), len(resp.Eids()), tcStr(resp.Tc), cttlStr(resp), abTestShort(resp.AbTestUUID))
	}
	return resp, nil
}

// enrichUserEids appends resolved eids to req.User.EIDs, creating User when absent (mirrors Java).
func enrichUserEids(rw *openrtb_ext.RequestWrapper, resolved []openrtb2.EID) {
	if rw == nil || rw.BidRequest == nil {
		return
	}
	if rw.User == nil {
		rw.User = &openrtb2.User{}
	}
	merged := make([]openrtb2.EID, 0, len(rw.User.EIDs)+len(resolved))
	merged = append(merged, rw.User.EIDs...)
	merged = append(merged, resolved...)
	rw.User.EIDs = merged
}

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}
