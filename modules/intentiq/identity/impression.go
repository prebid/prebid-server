package identity

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"

	"github.com/prebid/prebid-server/v4/hooks/hookstage"
)

// HandleAuctionResponseHook reports each winning bid to the IntentIQ impression API and records
// whole-flow latency. The bid response is never modified.
//
// Faithful port of Java IntentiqIdentityAuctionResponseHook. Unlike the Java hook — which reaches the
// bid request via AuctionContext — the Go auction-response payload exposes only the BidResponse, so
// the request-derived report fields (vrref/prebidAuctionId/ip/ua) come from the flowContext the enrich
// hook stashed in the module context.
func (m *Module) HandleAuctionResponseHook(
	_ context.Context,
	miCtx hookstage.ModuleInvocationContext,
	payload hookstage.AuctionResponsePayload,
) (hookstage.HookResult[hookstage.AuctionResponsePayload], error) {
	var result hookstage.HookResult[hookstage.AuctionResponsePayload]

	cfg := m.cfg.resolve(miCtx.AccountConfig)
	dpi := cfg.PartnerID

	fc, ok := getFlowContext(miCtx.ModuleContext)
	// Whole-flow latency: enrich entry -> bid release, recorded once per auction regardless of
	// whether an impression report is configured/sent.
	if ok {
		m.metrics.FlowLatency(time.Since(fc.start), dpi)
	}

	if cfg.ReportsEndpoint == "" || payload.BidResponse == nil {
		return result, nil
	}

	bidResponse := payload.BidResponse
	currency := bidResponse.Cur
	if currency == "" {
		currency = defaultCurrency
	}

	for i := range bidResponse.SeatBid {
		seatBid := bidResponse.SeatBid[i]
		for j := range seatBid.Bid {
			m.report(cfg, seatBid.Seat, seatBid.Bid[j], currency, fc, ok)
		}
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
	timeout := cfg.timeout()

	// Fire-and-forget: use a fresh background context (the hook's request context is cancelled once
	// the response is returned) bounded by the configured timeout. Recover so a stray panic in the
	// detached goroutine can never take down the server.
	go func() {
		defer func() { _ = recover() }()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reportURL, nil)
		if err != nil {
			m.metrics.ImpressionError(dpi)
			return
		}
		resp, err := m.httpClient.Do(req)
		if err != nil {
			m.metrics.ImpressionError(dpi)
			return
		}
		// Drain and close so the connection can be reused.
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
		m.metrics.ImpressionReported(dpi)
	}()
}

// buildReportURL assembles the reports-endpoint URL with the fixed query params plus the url-encoded
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
