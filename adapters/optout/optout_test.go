package optout

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
)

const endpoint = "https://bid.example.test/bid"

func parseRequest(t *testing.T, raw string) *openrtb2.BidRequest {
	t.Helper()
	var req openrtb2.BidRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatal(err)
	}
	return &req
}

func standardRequest(t *testing.T) *openrtb2.BidRequest {
	t.Helper()
	return parseRequest(t, `{"id":"auction","imp":[{"id":"imp-1","banner":{"w":300,"h":250},"ext":{"bidder":{"publisher":"152","adSlot":"home"}}}],"regs":{"gdpr":1},"site":{"page":"https://example.test/story"}}`)
}

func makeOutgoing(t *testing.T) (*openrtb2.BidRequest, *adapters.RequestData) {
	t.Helper()
	req := standardRequest(t)
	requests, errs := (&adapter{endpoint: endpoint}).MakeRequests(req, nil)
	if len(errs) != 0 || len(requests) != 1 {
		t.Fatalf("MakeRequests: requests=%d errs=%v", len(requests), errs)
	}
	return req, requests[0]
}

func response(status int, body string) *adapters.ResponseData {
	return &adapters.ResponseData{StatusCode: status, Body: []byte(body)}
}

func bidJSON(id, currency string, price float64) string {
	return fmt.Sprintf(`{"requestId":%q,"cpm":%g,"currency":%q,"width":300,"height":250,"ad":"<div>ok</div>","creativeId":"creative","ttl":45}`, id, price, currency)
}

func decodeOutgoing(t *testing.T, r *adapters.RequestData) optoutRequest {
	t.Helper()
	var out optoutRequest
	if err := json.Unmarshal(r.Body, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestJSONSamples(t *testing.T) {
	bidder, err := Builder("optout", config.Adapter{Endpoint: endpoint}, config.Server{})
	if err != nil {
		t.Fatal(err)
	}
	adapterstest.RunJSONBidderTest(t, "optouttest", bidder)
}

func TestBuilder(t *testing.T) {
	if _, err := Builder("optout", config.Adapter{}, config.Server{}); err == nil {
		t.Fatal("missing endpoint accepted")
	}
	b, err := Builder("optout", config.Adapter{Endpoint: endpoint}, config.Server{})
	if err != nil || b == nil {
		t.Fatalf("valid endpoint: bidder=%v err=%v", b, err)
	}
}

func TestDecodeParams(t *testing.T) {
	for _, tc := range []struct {
		name, ext string
		valid     bool
	}{
		{"valid", `{"bidder":{"publisher":"152","adSlot":"home","customs":{"section":"sports"}}}`, true},
		{"missing bidder", `{}`, false},
		{"invalid JSON", `{`, false},
		{"invalid bidder", `{"bidder":{"publisher":3,"adSlot":"home"}}`, false},
		{"missing publisher", `{"bidder":{"adSlot":"home"}}`, false},
		{"missing adSlot", `{"bidder":{"publisher":"152"}}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodeParams(openrtb2.Imp{Ext: json.RawMessage(tc.ext)})
			if (err == nil) != tc.valid {
				t.Fatalf("err=%v, want valid=%v", err, tc.valid)
			}
		})
	}
}

func TestCustomsNormalizationAndMerge(t *testing.T) {
	input := map[string]interface{}{"s": "hello", "n": float64(42), "bool": true, "arr": []interface{}{"sports", "news"}, "nested": map[string]interface{}{"x": "y"}, "nil": nil}
	got := normalizeCustoms(input)
	for key, expected := range map[string]string{"s": "hello", "n": "42", "bool": "true", "arr": "sports,news", "nested": `{"x":"y"}`} {
		if got[key] != expected {
			t.Errorf("%s: got %q want %q", key, got[key], expected)
		}
	}
	if _, exists := got["nil"]; exists {
		t.Error("nil must be omitted")
	}
	dst := map[string]string{"section": "old", "untouched": "yes"}
	mergeCustoms(dst, map[string]interface{}{"section": "new", "ignore": nil})
	if dst["section"] != "new" || dst["untouched"] != "yes" {
		t.Fatalf("bad merge: %v", dst)
	}
	if _, exists := dst["ignore"]; exists {
		t.Error("nil merged")
	}
}

func TestConsentReading(t *testing.T) {
	for _, tc := range []struct{ name, body, want string }{
		{"modern", `{"user":{"consent":"modern","ext":{"consent":"legacy"}}}`, "modern"},
		{"legacy", `{"user":{"ext":{"consent":"legacy"}}}`, "legacy"},
		{"missing", `{}`, ""},
		{"malformed extension", `{"user":{"ext":"not an object"}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := readConsent(parseRequest(t, tc.body)); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
	if got := readConsent(nil); got != "" {
		t.Fatalf("nil request returned %q", got)
	}
}

func TestMakeRequestsReportsMalformedExtensions(t *testing.T) {
	req := parseRequest(t, `{"id":"a","imp":[{"id":"imp-1","banner":{"w":300,"h":250},"ext":{"bidder":{"publisher":"152","adSlot":"top"}}}],"regs":{"gdpr":0},"site":{"ext":{"data":"not an object"}},"app":{"ext":{"data":{"shared":"app"}}},"user":{"ext":"not an object"}}`)
	requests, errs := (&adapter{endpoint: endpoint}).MakeRequests(req, nil)
	if len(requests) != 1 || len(errs) != 2 {
		t.Fatalf("requests=%d errs=%v", len(requests), errs)
	}
	for _, err := range errs {
		if _, ok := err.(*errortypes.BadInput); !ok {
			t.Errorf("expected BadInput warning, got %T: %v", err, err)
		}
	}
	if out := decodeOutgoing(t, requests[0]); out.Customs["shared"] != "app" {
		t.Fatalf("valid app data dropped: %v", out.Customs)
	}
}

func TestPersonalDataPermission(t *testing.T) {
	for _, tc := range []struct {
		name, regs, consent string
		want                bool
	}{
		{"gdpr false", `"regs":{"gdpr":0}`, "", true},
		{"gdpr true no consent", `"regs":{"gdpr":1}`, "", false},
		{"gdpr true invalid consent", `"regs":{"gdpr":1}`, "invalid", false},
		{"unknown", `"regs":{}`, "", false},
		{"missing regs", `"id":"x"`, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := parseRequest(t, "{"+tc.regs+"}")
			if got := mayForwardPersonalData(req, tc.consent); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
	if mayForwardPersonalData(nil, "") {
		t.Error("nil request permitted")
	}
}

func TestForwardableORTB2RedactionAndNoMutation(t *testing.T) {
	req := parseRequest(t, `{"id":"auction","regs":{"gdpr":1},"device":{"ip":"203.0.113.10","ipv6":"2001:db8::10","ua":"UA"},"user":{"id":"secret","buyeruid":"secret2","ext":{"data":{"segment":"private"}}}}`)
	before, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	redacted, err := makeForwardableORTB2(req, false)
	if err != nil {
		t.Fatal(err)
	}
	if redacted.Device == nil || redacted.Device.IP != "" || redacted.Device.IPv6 != "" {
		t.Fatalf("IP not cleared: %+v", redacted.Device)
	}
	if redacted.User == nil || redacted.User.ID != "" || redacted.User.BuyerUID != "" || !strings.Contains(string(redacted.User.Ext), `"redacted":true`) {
		t.Fatalf("user not redacted: %+v", redacted.User)
	}
	after, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("modified shared request")
	}
	allowed, err := makeForwardableORTB2(req, true)
	if err != nil {
		t.Fatal(err)
	}
	if allowed.Device.IP != "203.0.113.10" || allowed.User.ID != "secret" {
		t.Fatal("data unexpectedly removed with explicit permission")
	}
}

func TestMakeRequestsOnePublisherSeparateSlotCustoms(t *testing.T) {
	req := parseRequest(t, `{"id":"auction","cur":["GBP"],"imp":[{"id":"one","banner":{"w":300,"h":250},"ext":{"bidder":{"publisher":"152","adSlot":"top","customs":{"topic":"sports"}}}},{"id":"two","banner":{"format":[{"w":320,"h":50}]},"ext":{"bidder":{"publisher":"152","adSlot":"bottom","customs":{"topic":"finance"}}}}],"site":{"page":"https://example.test","ext":{"data":{"section":"site","shared":"site"}}},"app":{"ext":{"data":{"shared":"app"}}},"regs":{"gdpr":0},"user":{"ext":{"data":{"shared":"user"}}}}`)
	requests, errs := (&adapter{endpoint: endpoint}).MakeRequests(req, nil)
	if len(errs) != 0 || len(requests) != 1 {
		t.Fatalf("requests=%d errors=%v", len(requests), errs)
	}
	r := requests[0]
	out := decodeOutgoing(t, r)
	if r.Method != http.MethodPost || r.Uri != endpoint || r.Headers.Get("Content-Type") != "application/json" {
		t.Fatalf("bad transport: %+v", r)
	}
	if out.Publisher != "152" || len(out.Slots) != 2 || !reflect.DeepEqual(r.ImpIDs, []string{"one", "two"}) {
		t.Fatalf("bad batching: %+v", out)
	}
	if out.Slots[0].Customs["topic"] != "sports" || out.Slots[1].Customs["topic"] != "finance" {
		t.Fatalf("slot customs mixed: %+v", out.Slots)
	}
	if out.Customs["section"] != "site" || out.Customs["shared"] != "user" {
		t.Fatalf("shared customs incorrect: %v", out.Customs)
	}
	if out.Currency.AdServerCurrency != "GBP" || out.GDPR != 0 || out.URL != "https://example.test" {
		t.Fatalf("bad request fields: %+v", out)
	}
}

func TestMakeRequestsRejectsMixedPublisherAndInvalidImpressions(t *testing.T) {
	req := parseRequest(t, `{"id":"auction","imp":[{"id":"one","banner":{"w":300,"h":250},"ext":{"bidder":{"publisher":"152","adSlot":"top"}}},{"id":"other","banner":{"w":300,"h":250},"ext":{"bidder":{"publisher":"999","adSlot":"top"}}},{"id":"video","video":{},"ext":{"bidder":{"publisher":"152","adSlot":"video"}}},{"id":"bad","banner":{"w":300,"h":250},"ext":{"bidder":{"publisher":"152"}}}]}`)
	requests, errs := (&adapter{endpoint: endpoint}).MakeRequests(req, nil)
	if len(requests) != 1 || len(errs) != 3 {
		t.Fatalf("requests=%d errs=%v", len(requests), errs)
	}
	out := decodeOutgoing(t, requests[0])
	if len(out.Slots) != 1 || out.Slots[0].ID != "one" {
		t.Fatalf("unexpected slots: %+v", out.Slots)
	}
}

func TestMakeRequestsEmptyAndDefaultGDPR(t *testing.T) {
	a := &adapter{endpoint: endpoint}
	if requests, errs := a.MakeRequests(nil, nil); len(requests) != 0 || len(errs) != 1 {
		t.Fatalf("nil request requests=%d errs=%v", len(requests), errs)
	}
	req := parseRequest(t, `{"id":"a","imp":[{"id":"bad","video":{}}]}`)
	if requests, _ := a.MakeRequests(req, nil); len(requests) != 0 {
		t.Fatal("made HTTP call with zero valid slots")
	}
	req = standardRequest(t)
	req.Regs = nil
	requests, errs := a.MakeRequests(req, nil)
	if len(errs) != 0 || len(requests) != 1 {
		t.Fatalf("requests=%d errs=%v", len(requests), errs)
	}
	if out := decodeOutgoing(t, requests[0]); out.GDPR != 1 {
		t.Fatalf("expected default GDPR=1, got %d", out.GDPR)
	}
}

func TestMakeRequestsPrivacyAndORTB2(t *testing.T) {
	req := parseRequest(t, `{"id":"a","imp":[{"id":"imp-1","banner":{"w":300,"h":250},"ext":{"bidder":{"publisher":"152","adSlot":"top","includeOrtb2":true}}}],"regs":{"gdpr":1},"device":{"ip":"203.0.113.7","ipv6":"2001:db8::7","ua":"Test UA","os":"Android"},"user":{"id":"secret","ext":{"data":{"segment":"sensitive"}}},"site":{"ext":{"data":{"section":"news"}}}}`)
	before, _ := json.Marshal(req)
	requests, errs := (&adapter{endpoint: endpoint}).MakeRequests(req, nil)
	if len(errs) != 0 || len(requests) != 1 {
		t.Fatalf("requests=%d errs=%v", len(requests), errs)
	}
	out := decodeOutgoing(t, requests[0])
	if out.GDPR != 1 || out.OS != "Android" || out.Customs["section"] != "news" {
		t.Fatalf("bad request: %+v", out)
	}
	if _, ok := out.Customs["segment"]; ok {
		t.Error("user first-party data leaked")
	}
	if requests[0].Headers.Get("X-Forwarded-For") != "" {
		t.Error("IP forwarded without consent")
	}
	if requests[0].Headers.Get("X-Device-User-Agent") != "Test UA" {
		t.Error("missing UA header")
	}
	var forwarded openrtb2.BidRequest
	if err := json.Unmarshal([]byte(out.ORTB2), &forwarded); err != nil {
		t.Fatal(err)
	}
	if forwarded.Device.IP != "" || forwarded.Device.IPv6 != "" || forwarded.User.ID != "" {
		t.Fatalf("unredacted OpenRTB: %+v", forwarded)
	}
	after, _ := json.Marshal(req)
	if string(before) != string(after) {
		t.Error("original request mutated")
	}
}

func TestGetAllowedSizes(t *testing.T) {
	req := parseRequest(t, `{"imp":[{"id":"1","banner":{"w":300,"h":250,"format":[{"w":300,"h":250},{"w":320,"h":50},{"w":320,"h":50}]}}]}`)
	got := getAllowedSizes(req.Imp[0])
	want := []adSize{{300, 250}, {320, 50}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v want %+v", got, want)
	}
}

func TestMakeBidsSuccessfulFormatsAndCurrency(t *testing.T) {
	req, outgoing := makeOutgoing(t)
	for _, tc := range []struct{ name, body string }{
		{"wrapped", `{"bids":[` + bidJSON("imp-1", "EUR", 1.25) + `]}`},
		{"array", `[` + bidJSON("imp-1", "EUR", 1.25) + `]`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, errs := (&adapter{}).MakeBids(req, outgoing, response(200, tc.body))
			if len(errs) != 0 || res == nil || len(res.Bids) != 1 {
				t.Fatalf("res=%+v errs=%v", res, errs)
			}
			if res.Currency != "EUR" || res.Bids[0].Bid.Price != 1.25 || res.Bids[0].Bid.ImpID != "imp-1" || res.Bids[0].Bid.Exp != 45 {
				t.Fatalf("unexpected bid: currency=%s bid=%+v", res.Currency, res.Bids[0].Bid)
			}
		})
	}
}

func TestMakeBidsResponseFailures(t *testing.T) {
	req, outgoing := makeOutgoing(t)
	for _, tc := range []struct {
		name       string
		resp       *adapters.ResponseData
		outgoing   *adapters.RequestData
		wantErrors int
		wantNil    bool
	}{
		{"nil response", nil, outgoing, 1, true},
		{"no content", response(204, ""), outgoing, 0, true},
		{"server error", response(503, "failure"), outgoing, 1, true},
		{"missing outgoing", response(200, `[]`), nil, 1, true},
		{"bad outgoing", response(200, `[]`), &adapters.RequestData{Body: []byte("{")}, 1, true},
		{"no slots", response(200, `[]`), &adapters.RequestData{Body: []byte(`{"slots":[]}`)}, 1, true},
		{"invalid response", response(200, `garbage`), outgoing, 1, true},
		{"missing bids", response(200, `{"hello":"world"}`), outgoing, 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res, errs := (&adapter{}).MakeBids(req, tc.outgoing, tc.resp)
			if len(errs) != tc.wantErrors || (res == nil) != tc.wantNil {
				t.Fatalf("res=%+v errs=%v", res, errs)
			}
		})
	}
}

func TestMakeBidsRejectsInvalidBidsAndPreservesValidBids(t *testing.T) {
	req, outgoing := makeOutgoing(t)
	invalid := []struct{ name, body string }{
		{"unknown impression", bidJSON("unknown", "EUR", 1)},
		{"missing currency", bidJSON("imp-1", "", 1)},
		{"missing markup", `{"requestId":"imp-1","cpm":1,"currency":"EUR","width":300,"height":250,"ad":""}`},
		{"zero width", `{"requestId":"imp-1","cpm":1,"currency":"EUR","width":0,"height":250,"ad":"ok"}`},
		{"negative cpm", bidJSON("imp-1", "EUR", -1)},
		{"bad ext array", `{"requestId":"imp-1","cpm":1,"currency":"EUR","width":300,"height":250,"ad":"ok","optOutExt":[]}`},
		{"bad ext string", `{"requestId":"imp-1","cpm":1,"currency":"EUR","width":300,"height":250,"ad":"ok","optOutExt":"test"}`},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"bids":[` + bidJSON("imp-1", "EUR", 2) + `,` + tc.body + `]}`
			res, errs := (&adapter{}).MakeBids(req, outgoing, response(200, body))
			if len(errs) != 1 || res == nil || len(res.Bids) != 1 || res.Currency != "EUR" {
				t.Fatalf("res=%+v errs=%v", res, errs)
			}
		})
	}
}

func TestMakeBidsMixedCurrenciesAndZeroPrice(t *testing.T) {
	req, outgoing := makeOutgoing(t)
	mixed := `{"bids":[` + bidJSON("imp-1", "EUR", 2) + `,` + bidJSON("imp-1", "USD", 3) + `]}`
	res, errs := (&adapter{}).MakeBids(req, outgoing, response(200, mixed))
	if len(errs) != 1 || len(res.Bids) != 1 || res.Currency != "EUR" {
		t.Fatalf("res=%+v errs=%v", res, errs)
	}
	zero := `{"bids":[` + bidJSON("imp-1", "EUR", 0) + `]}`
	res, errs = (&adapter{}).MakeBids(req, outgoing, response(200, zero))
	if len(errs) != 0 || len(res.Bids) != 1 || res.Bids[0].Bid.Price != 0 {
		t.Fatalf("zero bid changed: res=%+v errs=%v", res, errs)
	}
	noBids, errs := (&adapter{}).MakeBids(req, outgoing, response(200, `[]`))
	if len(errs) != 0 || noBids.Currency != defaultCurrency || len(noBids.Bids) != 0 {
		t.Fatalf("empty result=%+v errs=%v", noBids, errs)
	}
}

func TestMakeBidsExtensionAndDefaultTTL(t *testing.T) {
	req, outgoing := makeOutgoing(t)
	body := `{"bids":[{"requestId":"imp-1","cpm":1,"currency":"EUR","width":300,"height":250,"ad":"<div/>","optOutExt":{"optout":{"version":1,"key":"test"}}}]}`
	res, errs := (&adapter{}).MakeBids(req, outgoing, response(200, body))
	if len(errs) != 0 || len(res.Bids) != 1 {
		t.Fatalf("res=%+v errs=%v", res, errs)
	}
	bid := res.Bids[0].Bid
	if bid.Exp != defaultTTL || !json.Valid(bid.Ext) || !strings.Contains(string(bid.Ext), `"optout"`) {
		t.Fatalf("bad ext or TTL: %+v", bid)
	}
}
