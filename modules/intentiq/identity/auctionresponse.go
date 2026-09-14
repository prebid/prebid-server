package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
	"github.com/prebid/prebid-server/v4/logger"
)

// HandleAuctionResponseHook reports each winning bid to the IntentIQ impression API. The bid
// response is never modified.
//
// Faithful port of Java IntentiqIdentityAuctionResponseHook. Unlike the Java hook — which reaches the
// bid request via AuctionContext — the Go auction-response payload exposes only the BidResponse, so
// the request-derived report fields (vrref/prebidAuctionId/ip/ua) come from the flowContext the enrich
// hook stashed in the module context.
func (m *Module) HandleAuctionResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (result hookstage.HookResult[hookstage.AuctionResponsePayload], err error) {
	start := time.Now()
	cfg := m.cfg.resolve(miCtx.AccountConfig)

	fc, ok := getFlowContext(miCtx.ModuleContext)

	tr := &flowTrace{collect: ok && fc.traceEnabled}

	defer func() {
		result.DebugMessages = tr.msgs
		if !tr.enabled() {
			return
		}
		combined := make([]string, 0, len(fc.trace)+len(tr.msgs))
		combined = append(combined, fc.trace...)
		combined = append(combined, tr.msgs...)
		if len(combined) == 0 {
			return
		}
		result.ChangeSet.AddMutation(
			setIdentityTraceMutation(combined),
			hookstage.MutationUpdate, "ext", "trace", traceExtKey,
		)
	}()

	if tr.enabled() {
		if ok {
			tr.tracef("[response] start — flow latency (enrich → response) %s", fmtDur(time.Since(fc.start)))
		} else {
			tr.tracef("[response] start — no enrich flow context (enrich hook did not run)")
		}
	}

	if cfg.ReportsEndpoint == "" {
		if tr.enabled() {
			tr.tracef("[response] no reports_endpoint configured — reporting skipped; response hook took %s", fmtDur(time.Since(start)))
		}
		return result, nil
	}
	if payload.BidResponse == nil {
		if tr.enabled() {
			tr.tracef("[response] nil bid response — reporting skipped; response hook took %s", fmtDur(time.Since(start)))
		}
		return result, nil
	}

	bidResponse := payload.BidResponse
	currency := bidResponse.Cur
	if currency == "" {
		currency = defaultCurrency
	}

	n := 0
	for i := range bidResponse.SeatBid {
		seatBid := bidResponse.SeatBid[i]
		for j := range seatBid.Bid {
			m.report(cfg, seatBid.Seat, seatBid.Bid[j], currency, fc, ok)
			n++
		}
	}
	if tr.enabled() {
		tr.tracef("[response] queued %d impression report(s) → %s (fire-and-forget); tc=%s carried from enrich",
			n, cfg.ReportsEndpoint, tcStr(fc.terminationCause))
		tr.tracef("[response] response hook took %s", fmtDur(time.Since(start)))
	}

	return result, nil
}

// report builds the rdata payload for a single bid and fires a fire-and-forget GET to the reports
// endpoint. The bid response is never touched.
func (m *Module) report(cfg Config, bidderCode string, bid openrtb2.Bid, currency string, fc flowContext, haveFC bool) {
	rdata := newOrderedMap()
	rdata.put("bidderCode", bidderCode)
	rdata.put("partnerId", cfg.PartnerID)
	rdata.put("cpm", bid.Price)
	rdata.put("currency", currency)
	appendOriginalBid(rdata, bid.Ext)
	rdata.put("placementId", bid.ImpID)
	rdata.put("biddingPlatformId", biddingPlatformOpenRTB)

	if haveFC {
		putIfPresent(rdata, "vrref", fc.ref)
		putIfPresent(rdata, "prebidAuctionId", fc.auctionID)
		putIfPresent(rdata, "partnerAuctionId", fc.auctionID)
		putIfPresent(rdata, "abTestUuid", fc.abTestUUID)
		if fc.terminationCause != nil {
			rdata.put("terminationCause", *fc.terminationCause)
		}
		putIfPresent(rdata, "ip", fc.ip)
		putIfPresent(rdata, "ua", fc.ua)
	}

	reportURL := buildReportURL(cfg, rdata)
	dpi := cfg.PartnerID
	timeout := cfg.GetTimeout()

	// Fire-and-forget: use a fresh background context (the hook's request context is cancelled once
	// the response is returned) bounded by the configured timeout. Recover so a stray panic in the
	// detached goroutine can never take down the server.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Errorf("intentiq-identity: panic in impression report (dpi=%s): %v\n%s", dpi, r, debug.Stack())
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reportURL, nil)
		if err != nil {
			m.metrics.ImpressionError(dpi)
			logger.Warnf("intentiq-identity: impression report failed (dpi=%s): %v", dpi, err)
			return
		}
		resp, err := m.httpClient.Do(req)
		if err != nil {
			m.metrics.ImpressionError(dpi)
			logger.Warnf("intentiq-identity: impression report failed (dpi=%s): %v", dpi, err)
			return
		}
		// Drain and close so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		m.metrics.ImpressionReported(dpi)
	}()
}

// buildReportURL assembles the reports_endpoint URL with the fixed query params plus the url-encoded
// dpi and rdata JSON, mirroring the Java reportUrl/encodeComponent.
func buildReportURL(cfg Config, rdata *orderedMap) string {
	sep := "?"
	if strings.Contains(cfg.ReportsEndpoint, "?") {
		sep = "&"
	}
	rdataJSON, _ := rdata.MarshalJSON()

	var b strings.Builder
	b.WriteString(cfg.ReportsEndpoint)
	b.WriteString(sep)
	b.WriteString("at=45")
	b.WriteString("&rtype=1")
	b.WriteString("&source=" + sourcePBSGo)
	b.WriteString("&dpi=" + encodeComponent(cfg.PartnerID))
	b.WriteString("&rdata=" + encodeComponent(string(rdataJSON)))
	return b.String()
}

// appendOriginalBid pulls origbidcpm (numeric) and origbidcur (non-blank string) from the bid ext,
// adding originalCpm/originalCurrency to rdata. A non-numeric origbidcpm or unparseable ext is
// silently skipped, matching the Java isNumber()/isNotBlank() guards.
func appendOriginalBid(rdata *orderedMap, ext json.RawMessage) {
	if len(ext) == 0 {
		return
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(ext, &fields); err != nil {
		return
	}
	if raw, ok := fields["origbidcpm"]; ok {
		var num json.Number
		if err := json.Unmarshal(raw, &num); err == nil && num != "" {
			rdata.put("originalCpm", num)
		}
	}
	if raw, ok := fields["origbidcur"]; ok {
		var cur string
		if err := json.Unmarshal(raw, &cur); err == nil && strings.TrimSpace(cur) != "" {
			rdata.put("originalCurrency", cur)
		}
	}
}

// putIfPresent adds key only when value is non-blank (StringUtils.isNotBlank parity). Uses the shared
// notBlank helper from params.go.
func putIfPresent(rdata *orderedMap, key, value string) {
	if notBlank(value) {
		rdata.put(key, value)
	}
}

// orderedMap is a minimal insertion-ordered string-keyed map that marshals to a JSON object in
// insertion order, matching the Java LinkedHashMap-backed rdata so the produced JSON key order is
// identical.
type orderedMap struct {
	keys   []string
	values map[string]any
}

func newOrderedMap() *orderedMap {
	return &orderedMap{values: make(map[string]any)}
}

// put sets key to value, appending to the order the first time the key is seen (later puts overwrite
// the value but keep the original position, like LinkedHashMap).
func (o *orderedMap) put(key string, value any) {
	if _, exists := o.values[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.values[key] = value
}

// MarshalJSON renders the object with keys in insertion order.
func (o *orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		keyJSON, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyJSON)
		buf.WriteByte(':')
		valJSON, err := json.Marshal(o.values[k])
		if err != nil {
			return nil, err
		}
		buf.Write(valJSON)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
