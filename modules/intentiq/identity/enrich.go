package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/modules/intentiq/identity/cache"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/prebid/prebid-server/v4/util/jsonutil"
)

// terminationCauseMetricMax bounds the tc values recorded as a metric (0 <= tc < 200); larger
// values are IntentIQ diagnostic codes not tracked as a business counter.
const terminationCauseMetricMax = 200

// HandleProcessedAuctionHook enriches user.eids with IntentIQ-resolved ids. Faithful port of the Java
// IntentiqIdentityProcessedAuctionRequestHook: it resolves identity (from the alias cache or a live
// S2S call) and, on a hit, appends the resolved eids to user.eids. It is fail-open — any resolution
// error leaves the request untouched. Flow state for the impression hook is always stashed in the
// returned ModuleContext.
func (m *Module) HandleProcessedAuctionHook(
	ctx context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.ProcessedAuctionRequestPayload,
) (hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload], error) {
	start := time.Now()
	cfg := m.cfg.resolve(miCtx.AccountConfig)
	dpi := cfg.PartnerID

	rw := payload.Request
	var req *openrtb2.BidRequest
	if rw != nil {
		req = rw.BidRequest
	}

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

	var result hookstage.HookResult[hookstage.ProcessedAuctionRequestPayload]

	if !notBlank(cfg.APIEndpoint) {
		m.metrics.SkipNoEndpoint(dpi)
		result.ModuleContext = setFlowContext(fc)
		return result, nil
	}
	if req == nil {
		result.ModuleContext = setFlowContext(fc)
		return result, nil
	}

	res, err := m.resolveEids(ctx, cfg, rw)
	if err != nil {
		// Fail open: proceed without enrichment (abTestUuid/tc unknown on error).
		m.metrics.APIError(dpi)
		result.ModuleContext = setFlowContext(fc)
		return result, nil
	}

	fc.abTestUUID = res.abTestUUID
	fc.terminationCause = res.terminationCause
	result.ModuleContext = setFlowContext(fc)

	if len(res.eids) == 0 {
		m.metrics.EidsNone(dpi)
		return result, nil
	}

	m.metrics.Enriched(dpi)
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
func (m *Module) resolveEids(ctx context.Context, cfg Config, rw *openrtb_ext.RequestWrapper) (resolution, error) {
	dpi := cfg.PartnerID
	req := rw.BidRequest

	cacheEnabled := m.cache != nil && cfg.Cache.Enabled
	var keys []cache.Key
	if cacheEnabled {
		keys = m.keyExtractor.CandidateKeys(req)
	}
	if len(keys) == 0 {
		resp, err := m.fetch(ctx, cfg, rw)
		if err != nil {
			return resolution{}, err
		}
		return resolution{eids: resp.eids(), abTestUUID: resp.AbTestUUID, terminationCause: resp.Tc}, nil
	}

	// Cache counters are broken down by key type: the type of the key that actually matched for
	// Hit/Negative/InProgress, and the request's primary (highest-priority) candidate type for a full
	// Miss, where no key matched.
	primaryType := keys[0].Type.Token()
	res := m.cache.Get(ctx, keys)
	switch res.State {
	case cache.Hit:
		m.metrics.CacheHit(res.Layer.Token(), res.KeyType.Token(), dpi)
		return resolution{eids: res.Eids}, nil
	case cache.Negative:
		// A negative sentinel is a cached miss (id known-unresolvable): counts toward cache.miss, and
		// the negative-specific counter distinguishes it from a true miss.
		typ := res.KeyType.Token()
		m.metrics.CacheMiss(typ, dpi)
		m.metrics.CacheNegativeHit(res.Layer.Token(), typ, dpi)
		return resolution{}, nil
	case cache.InProgress:
		// A resolution call for this id is already in flight; skip a duplicate and don't enrich.
		m.metrics.CacheInProgress(res.Layer.Token(), res.KeyType.Token(), dpi)
		return resolution{}, nil
	default: // cache.Miss
		m.metrics.CacheMiss(primaryType, dpi)
		m.cache.PutInProgress(ctx, keys)
		resp, err := m.fetch(ctx, cfg, rw)
		if err != nil {
			return resolution{}, err
		}
		eids := resp.eids()
		if len(eids) > 0 {
			m.cache.Put(ctx, keys, eids, resp.cttl())
		} else {
			m.cache.PutNegative(ctx, keys, resp.cttl())
		}
		return resolution{eids: eids, abTestUUID: resp.AbTestUUID, terminationCause: resp.Tc}, nil
	}
}

// fetch performs the identity-resolution S2S GET (with the gdpr-consent header when present) under a
// per-request timeout, parses the response leniently, and records the API metrics. APIError is left
// to the caller's fail-open path so it is counted once per failed resolution (as in Java).
func (m *Module) fetch(ctx context.Context, cfg Config, rw *openrtb_ext.RequestWrapper) (iiqResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, cfg.timeout())
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, resolveURL(cfg, rw), nil)
	if err != nil {
		return iiqResponse{}, err
	}
	if consent := resolveConsent(rw); consent != "" {
		httpReq.Header.Set(gdprConsentHeader, consent)
	}

	start := time.Now()
	resp, err := m.httpClient.Do(httpReq)
	m.metrics.APILatency(time.Since(start), cfg.PartnerID)
	if err != nil {
		return iiqResponse{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return iiqResponse{}, err
	}

	var parsed iiqResponse
	if err := jsonutil.Unmarshal(body, &parsed); err != nil {
		return iiqResponse{}, err
	}

	m.metrics.APISuccess(cfg.PartnerID)
	if parsed.Tc != nil && *parsed.Tc >= 0 && *parsed.Tc < terminationCauseMetricMax {
		m.metrics.TerminationCause(*parsed.Tc, cfg.PartnerID)
	}
	return parsed, nil
}

// eids leniently extracts the resolved eids from the response data. IntentIQ returns data as an
// object on a hit but as an empty string ("") on an empty/invalid response, so a non-object data is
// treated as absent rather than failing the parse.
func (r iiqResponse) eids() []openrtb2.EID {
	data := bytes.TrimSpace(r.Data)
	if len(data) == 0 || data[0] != '{' {
		return nil
	}
	var d struct {
		Eids []openrtb2.EID `json:"eids"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return nil
	}
	return d.Eids
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
