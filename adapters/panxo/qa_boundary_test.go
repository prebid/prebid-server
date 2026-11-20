package panxo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func qaBidder(t *testing.T) adapters.Bidder {
	t.Helper()
	bidder, err := Builder(openrtb_ext.BidderPanxo, config.Adapter{Endpoint: "https://reviewer.invalid/openrtb/2.6/bid"}, config.Server{})
	if err != nil {
		t.Fatal(err)
	}
	return bidder
}

func TestQAPanxoInventoryNamespace(t *testing.T) {
	cases := []struct {
		name string
		ext  string
		want string
	}{
		{"true", `{"inventory":{"placement":"ordinary"},"data":{"documentLang":"ca","panxo":{"ai":true,"uid":"IDENTITY-uid","history":[{"url":"IDENTITY-history"}],"memberships":["IDENTITY-member"],"links":{"account":"IDENTITY-account"},"source":"broad"}}}`, `{"inventory":{"placement":"ordinary"},"data":{"documentLang":"ca","panxo":{"ai":true}}}`},
		{"false", `{"data":{"panxo":{"ai":false,"uid":"IDENTITY-uid"}}}`, `{"data":{"panxo":{"ai":false}}}`},
		{"absent", `{"inventory":{"placement":"ordinary"}}`, `{"inventory":{"placement":"ordinary"}}`},
		{"null panxo", `{"data":{"panxo":null,"documentLang":"ca"}}`, `{"data":{"documentLang":"ca"}}`},
		{"string panxo", `{"data":{"panxo":"IDENTITY-string"}}`, `{"data":{}}`},
		{"array panxo", `{"data":{"panxo":[{"ai":true,"uid":"IDENTITY-array"}]}}`, `{"data":{}}`},
		{"number panxo", `{"data":{"panxo":7}}`, `{"data":{}}`},
		{"boolean panxo", `{"data":{"panxo":true}}`, `{"data":{}}`},
		{"absent ai", `{"data":{"panxo":{"uid":"IDENTITY-only"}}}`, `{"data":{}}`},
		{"string ai", `{"data":{"panxo":{"ai":"true","uid":"IDENTITY-string-ai"}}}`, `{"data":{}}`},
		{"number ai", `{"data":{"panxo":{"ai":1}}}`, `{"data":{}}`},
		{"null ai", `{"data":{"panxo":{"ai":null}}}`, `{"data":{}}`},
		{"array ai", `{"data":{"panxo":{"ai":[true]}}}`, `{"data":{}}`},
		{"object ai", `{"data":{"panxo":{"ai":{"uid":"IDENTITY-object-ai"}}}}`, `{"data":{}}`},
		{"null data", `{"data":null,"inventory":"ordinary"}`, `{"inventory":"ordinary"}`},
		{"string data", `{"data":"IDENTITY-data","inventory":"ordinary"}`, `{"inventory":"ordinary"}`},
		{"array data", `{"data":[{"panxo":{"ai":true,"uid":"IDENTITY-array"}}],"inventory":"ordinary"}`, `{"inventory":"ordinary"}`},
	}
	for _, channel := range []string{"site", "app"} {
		for _, tc := range cases {
			t.Run(channel+"/"+tc.name, func(t *testing.T) {
				request := &openrtb2.BidRequest{ID: "auction", Imp: []openrtb2.Imp{
					{ID: "a", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"propertyKey":"one"}}`)},
					{ID: "b", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"propertyKey":"two"}}`)},
				}}
				if channel == "site" {
					request.Site = &openrtb2.Site{Domain: "publisher.example", Ext: json.RawMessage(tc.ext)}
				} else {
					request.App = &openrtb2.App{Bundle: "app.publisher.example", Ext: json.RawMessage(tc.ext)}
				}
				before, err := json.Marshal(request)
				if err != nil {
					t.Fatal(err)
				}
				var want map[string]any
				if err := json.Unmarshal([]byte(tc.want), &want); err != nil {
					t.Fatal(err)
				}
				for iteration := 0; iteration < 2; iteration++ {
					requests, errs := qaBidder(t).MakeRequests(request, nil)
					if len(errs) != 0 || len(requests) != 2 {
						t.Fatalf("requests=%d errors=%v", len(requests), errs)
					}
					for _, outbound := range requests {
						var body openrtb2.BidRequest
						if err := json.Unmarshal(outbound.Body, &body); err != nil {
							t.Fatal(err)
						}
						var ext json.RawMessage
						if channel == "site" {
							if body.Site == nil || body.Site.Domain != request.Site.Domain || body.App != nil {
								t.Fatal("site inventory changed")
							}
							ext = body.Site.Ext
						} else {
							if body.App == nil || body.App.Bundle != request.App.Bundle || body.Site != nil {
								t.Fatal("app inventory changed")
							}
							ext = body.App.Ext
						}
						var got map[string]any
						if err := json.Unmarshal(ext, &got); err != nil {
							t.Fatal(err)
						}
						if !reflect.DeepEqual(got, want) || strings.Contains(string(outbound.Body), "IDENTITY-") {
							t.Fatalf("unexpected inventory context: %s", ext)
						}
						outbound.Body[0] = '['
					}
					after, _ := json.Marshal(request)
					if string(after) != string(before) {
						t.Fatal("Panxo normalization changed caller-owned context")
					}
				}
			})
		}
	}
}

func TestQARejectsMalformedInventoryExtensionsWithoutEchoingContents(t *testing.T) {
	for _, channel := range []string{"site", "app"} {
		for _, raw := range []string{`"IDENTITY-string"`, `[{"panxo":"IDENTITY-array"}]`, `true`, `7`, `{"broken":`} {
			t.Run(channel+"/"+raw, func(t *testing.T) {
				request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "a", Ext: json.RawMessage(`{"bidder":{"propertyKey":"one"}}`)}}}
				if channel == "site" {
					request.Site = &openrtb2.Site{Ext: json.RawMessage(raw)}
				} else {
					request.App = &openrtb2.App{Ext: json.RawMessage(raw)}
				}
				requests, errs := qaBidder(t).MakeRequests(request, nil)
				if len(requests) != 0 || len(errs) != 1 {
					t.Fatalf("requests=%d errors=%v", len(requests), errs)
				}
				var badInput *errortypes.BadInput
				if !errors.As(errs[0], &badInput) || errs[0].Error() != "invalid inventory.ext" {
					t.Fatalf("expected bounded BadInput, got %v", errs[0])
				}
				if channel == "site" && string(request.Site.Ext) != raw || channel == "app" && string(request.App.Ext) != raw {
					t.Fatal("error path changed caller extension")
				}
			})
		}
	}
}

func TestQAGroupsEscapeAndDoNotShareHeaders(t *testing.T) {
	key := "à/slot?query=x&source=attacker+#%"
	ext := func(k string) json.RawMessage {
		b, err := json.Marshal(map[string]any{"bidder": map[string]string{"propertyKey": k}, "gpid": "slot"})
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	request := &openrtb2.BidRequest{ID: "auction", Imp: []openrtb2.Imp{
		{ID: "a", Banner: &openrtb2.Banner{}, Ext: ext(key)},
		{ID: "b", Banner: &openrtb2.Banner{}, Ext: ext("other")},
		{ID: "c", Banner: &openrtb2.Banner{}, Ext: ext(key)},
	}}
	before, _ := json.Marshal(request)
	bidder := qaBidder(t)
	for iteration := 0; iteration < 3; iteration++ {
		requests, errs := bidder.MakeRequests(request, nil)
		if len(errs) != 0 || len(requests) != 2 {
			t.Fatalf("requests=%d errs=%v", len(requests), errs)
		}
		if !reflect.DeepEqual(requests[0].ImpIDs, []string{"a", "c"}) || !reflect.DeepEqual(requests[1].ImpIDs, []string{"b"}) {
			t.Fatalf("unstable grouping: %#v %#v", requests[0].ImpIDs, requests[1].ImpIDs)
		}
		parsed, err := url.Parse(requests[0].Uri)
		if err != nil {
			t.Fatal(err)
		}
		q := parsed.Query()
		if len(q) != 2 || q.Get("key") != key || q.Get("source") != "prebid-server" || parsed.Fragment != "" {
			t.Fatalf("query escaped incorrectly: %s", requests[0].Uri)
		}
		requests[0].Headers.Set("Content-Type", "changed")
		if requests[1].Headers.Get("Content-Type") != "application/json;charset=utf-8" {
			t.Fatal("groups share mutable headers")
		}
		var body openrtb2.BidRequest
		if err := json.Unmarshal(requests[0].Body, &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Imp) != 2 || body.Imp[0].ID != "a" || body.Imp[1].ID != "c" {
			t.Fatal("group body and ImpIDs differ")
		}
	}
	after, _ := json.Marshal(request)
	if string(before) != string(after) {
		t.Fatal("grouping modified original request")
	}
}

func TestQACompleteRequestImmutability(t *testing.T) {
	var req openrtb2.BidRequest
	if err := json.Unmarshal([]byte(`{"id":"auction","cur":["EUR"],"imp":[{"id":"one","banner":{"format":[{"w":300,"h":250}]},"pmp":{"deals":[{"id":"d","bidfloor":2.5,"bidfloorcur":"EUR"}]},"ext":{"bidder":{"propertyKey":"p"},"context":{"nested":[1,2]}}}],"app":{"bundle":"test.example"},"device":{"devicetype":3,"ext":{"context":{"x":1}}},"regs":{"gpp":"synthetic","gpp_sid":[7]},"source":{"ext":{"schain":{"ver":"1.0","complete":1,"nodes":[]}}},"ext":{"context":{"number":7}}}`), &req); err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(req)
	bidder := qaBidder(t)
	requests, errs := bidder.MakeRequests(&req, qaCurrencyRates(2))
	if len(errs) != 0 || len(requests) != 1 {
		t.Fatalf("%v", errs)
	}
	requests[0].Body[0] = '['
	requests[0].ImpIDs[0] = "mutated"
	after, _ := json.Marshal(req)
	if string(before) != string(after) {
		t.Fatal("output mutations affected original request")
	}
}

func TestQAMixedResponsesPreserveValidDealsAndMetadata(t *testing.T) {
	request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{
		{ID: "banner", Banner: &openrtb2.Banner{}},
		{ID: "video", Video: &openrtb2.Video{}},
		{ID: "other-property", Native: &openrtb2.Native{}},
	}}
	validBanner := openrtb2.Bid{ID: "b", ImpID: "banner", Price: 0, DealID: "guaranteed", MType: openrtb2.MarkupBanner, AdM: "<div/>", NURL: "https://example.invalid/win", BURL: "https://example.invalid/bill", Ext: json.RawMessage(`{"prebid":{"meta":{"networkName":"Example","futureField":{"x":1}}},"dsa":{"behalf":"buyer","paid":"buyer"}}`)}
	validVideo := openrtb2.Bid{ID: "v", ImpID: "video", Price: 3.25, DealID: "video-deal", MType: openrtb2.MarkupVideo, AdM: "<VAST/>"}
	body, err := json.Marshal(openrtb2.BidResponse{Cur: "EUR", SeatBid: []openrtb2.SeatBid{
		{Seat: "a", Bid: []openrtb2.Bid{validBanner, {ID: "bad-type", ImpID: "banner", MType: openrtb2.MarkupVideo}}},
		{Seat: "b", Bid: []openrtb2.Bid{{ID: "cross-property", ImpID: "other-property", MType: openrtb2.MarkupNative}, validVideo, {ID: "unknown", ImpID: "not-requested", MType: openrtb2.MarkupBanner}}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	result, errs := qaBidder(t).MakeBids(request, &adapters.RequestData{ImpIDs: []string{"banner", "video"}}, &adapters.ResponseData{StatusCode: 200, Body: body})
	if len(errs) != 3 {
		t.Fatalf("wanted 3 isolated response errors, got %v", errs)
	}
	for _, err := range errs {
		var bad *errortypes.BadServerResponse
		if !errors.As(err, &bad) {
			t.Errorf("wrong error type: %T", err)
		}
	}
	if result == nil || len(result.Bids) != 2 || result.Currency != "EUR" {
		t.Fatalf("wrong accepted responses: %#v", result)
	}
	for i, want := range []openrtb2.Bid{validBanner, validVideo} {
		if !reflect.DeepEqual(*result.Bids[i].Bid, want) {
			t.Errorf("valid bid %d changed: %#v", i, result.Bids[i].Bid)
		}
	}
	result.Bids[0].Bid.ID = "changed"
	if result.Bids[1].Bid.ID != "v" {
		t.Fatal("distinct accepted bids alias each other")
	}
}

func TestQADefaultCurrencyAndNoMedia(t *testing.T) {
	for _, cur := range []string{"", "EUR"} {
		t.Run("currency-"+cur, func(t *testing.T) {
			body, _ := json.Marshal(openrtb2.BidResponse{Cur: cur, SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{ID: "bid", ImpID: "imp", MType: 1}}}}})
			result, errs := qaBidder(t).MakeBids(&openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp", Banner: &openrtb2.Banner{}}}}, nil, &adapters.ResponseData{StatusCode: 200, Body: body})
			want := cur
			if want == "" {
				want = "USD"
			}
			if len(errs) != 0 || result == nil || result.Currency != want {
				t.Fatalf("currency=%#v errs=%v", result, errs)
			}
		})
	}
	t.Run("no-media-no-mtype", func(t *testing.T) {
		result, errs := qaBidder(t).MakeBids(&openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp"}}}, nil, &adapters.ResponseData{StatusCode: 200, Body: []byte(`{"seatbid":[{"bid":[{"id":"bid","impid":"imp"}]}]}`)})
		if len(errs) != 1 || result == nil || len(result.Bids) != 0 {
			t.Fatalf("ambiguous bid accepted: %#v %v", result, errs)
		}
	})
}

func TestQAStatusHandlingDoesNotParseErrorBody(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusBadRequest, http.StatusUnauthorized, http.StatusTooManyRequests, http.StatusInternalServerError, http.StatusBadGateway} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			result, errs := qaBidder(t).MakeBids(nil, nil, &adapters.ResponseData{StatusCode: status, Body: []byte("not JSON")})
			if status == http.StatusNoContent {
				if result != nil || len(errs) != 0 {
					t.Fatalf("204 not treated as no bid: %v", errs)
				}
				return
			}
			if result != nil || len(errs) != 1 {
				t.Fatalf("status %d result=%#v errs=%v", status, result, errs)
			}
			if status == http.StatusBadRequest {
				var bad *errortypes.BadInput
				if !errors.As(errs[0], &bad) {
					t.Fatalf("want BadInput got %T", errs[0])
				}
			} else {
				var bad *errortypes.BadServerResponse
				if !errors.As(errs[0], &bad) {
					t.Fatalf("want BadServerResponse got %T", errs[0])
				}
			}
		})
	}
}

func TestQAStoredResponsesWithoutImpressions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		mtype    openrtb2.MarkupType
		wantType openrtb_ext.BidType
	}{
		{"banner", openrtb2.MarkupBanner, openrtb_ext.BidTypeBanner},
		{"video", openrtb2.MarkupVideo, openrtb_ext.BidTypeVideo},
		{"native", openrtb2.MarkupNative, openrtb_ext.BidTypeNative},
		{"missing mtype", 0, ""},
		{"unsupported audio", openrtb2.MarkupAudio, ""},
		{"unknown mtype", 99, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(openrtb2.BidResponse{
				ID: "stored-response", Cur: "USD",
				SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{
					ID: "stored-bid", ImpID: "stored-imp", Price: 1.25,
					MType: tc.mtype, AdM: "stored-markup",
				}}}},
			})
			if err != nil {
				t.Fatal(err)
			}
			result, errs := qaBidder(t).MakeBids(
				&openrtb2.BidRequest{ID: "live-auction"},
				&adapters.RequestData{Method: http.MethodPost},
				&adapters.ResponseData{StatusCode: http.StatusOK, Body: body},
			)
			if tc.wantType == "" {
				var badResponse *errortypes.BadServerResponse
				if len(errs) != 1 || !errors.As(errs[0], &badResponse) || result == nil || len(result.Bids) != 0 {
					t.Fatalf("expected invalid stored response rejection: %#v %v", result, errs)
				}
				return
			}
			if len(errs) != 0 || result == nil || len(result.Bids) != 1 {
				t.Fatalf("expected one stored bid: %#v %v", result, errs)
			}
			bid := result.Bids[0]
			if bid.BidType != tc.wantType || bid.Bid.ImpID != "stored-imp" {
				t.Fatalf("expected %s with original stored impid, got %#v", tc.wantType, bid)
			}
		})
	}
}

func TestQAStoredResponseExceptionDoesNotWeakenLiveValidation(t *testing.T) {
	request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "live", Banner: &openrtb2.Banner{}}}}
	for _, tc := range []struct {
		name    string
		data    *adapters.RequestData
		impID   string
		mtype   openrtb2.MarkupType
		wantBid bool
	}{
		{"stored explicit native ignores unrelated live media", &adapters.RequestData{}, "live", openrtb2.MarkupNative, true},
		{"stored explicit banner can have old impid", &adapters.RequestData{}, "stored", openrtb2.MarkupBanner, true},
		{"HTTP without ImpIDs still checks request membership", &adapters.RequestData{Uri: "https://reviewer.invalid/bid"}, "stored", openrtb2.MarkupBanner, false},
		{"HTTP with ImpIDs rejects cross property", &adapters.RequestData{Uri: "https://reviewer.invalid/bid", ImpIDs: []string{"other"}}, "live", openrtb2.MarkupBanner, false},
		{"empty URI with ImpIDs is not stored", &adapters.RequestData{ImpIDs: []string{"other"}}, "live", openrtb2.MarkupBanner, false},
		{"stored omitted mtype cannot use unrelated live media", &adapters.RequestData{}, "live", 0, false},
		{"stored audio remains unsupported", &adapters.RequestData{}, "stored", openrtb2.MarkupAudio, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(openrtb2.BidResponse{SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{ID: "bid", ImpID: tc.impID, MType: tc.mtype}}}}})
			result, errs := qaBidder(t).MakeBids(request, tc.data, &adapters.ResponseData{StatusCode: 200, Body: body})
			if tc.wantBid {
				if len(errs) != 0 || result == nil || len(result.Bids) != 1 {
					t.Fatalf("expected stored bid: %#v %v", result, errs)
				}
			} else {
				if len(errs) != 1 || result == nil || len(result.Bids) != 0 {
					t.Fatalf("expected rejection: %#v %v", result, errs)
				}
			}
		})
	}
}

func TestQANamedIdentityRemovedFromEveryProperty(t *testing.T) {
	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(`{
		"id":"auction","cur":["EUR"],
		"imp":[
			{"id":"banner","banner":{"format":[{"w":300,"h":250}]},"bidfloor":1.25,"bidfloorcur":"EUR","pmp":{"deals":[{"id":"deal","bidfloor":2.5,"bidfloorcur":"EUR"}]},"ext":{"bidder":{"propertyKey":"p"},"gpid":"slot-banner"}},
			{"id":"video","video":{"mimes":["video/mp4"],"protocols":[2,3],"w":1920,"h":1080},"ext":{"bidder":{"propertyKey":"q"}}},
			{"id":"native","native":{"request":"{\"assets\":[{\"id\":1,\"title\":{\"len\":80}}]}"},"ext":{"bidder":{"propertyKey":"p"}}}
		],
		"user":{"id":"IDENTITY-id","buyeruid":"IDENTITY-buyer","yob":1987,"gender":"IDENTITY-gender","keywords":"IDENTITY-keywords","kwarray":["IDENTITY-kwarray"],"customdata":"IDENTITY-custom","geo":{"lat":45.12345,"lon":1.54321},"data":[{"id":"IDENTITY-data","segment":[{"id":"IDENTITY-membership"}]}],"eids":[{"source":"IDENTITY-top-eid","uids":[{"id":"IDENTITY-top-uid"}]}],"consent":"privacy-main","ext":{"consent":"privacy-legacy","eids":[{"source":"IDENTITY-ext-eid","uids":[{"id":"IDENTITY-ext-uid"}]}],"data":{"panxo_uid":"IDENTITY-panxo"},"future":"IDENTITY-future"}},
		"device":{"ifa":"IDENTITY-ifa","ip":"192.0.2.9","ipv6":"2001:db8::9","didsha1":"IDENTITY-didsha1","didmd5":"IDENTITY-didmd5","dpidsha1":"IDENTITY-dpidsha1","dpidmd5":"IDENTITY-dpidmd5","macsha1":"IDENTITY-macsha1","macmd5":"IDENTITY-macmd5","ua":"test-agent","devicetype":3,"language":"es","dnt":1,"lmt":1,"ext":{"panxo":{"uid":"IDENTITY-device-panxo","v1":"IDENTITY-session"},"vpw":1920,"vph":1080}},
		"regs":{"gdpr":1,"gpp":"privacy-gpp","gpp_sid":[7],"ext":{"gdpr":1,"us_privacy":"1YNN"}},
		"source":{"tid":"auction-transaction","ext":{"schain":{"ver":"1.0","complete":1,"nodes":[{"asi":"publisher.example","sid":"seller","hp":1}]}}}
	}`), &request); err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	// Core may share User and Device pointers with another bidder's request.
	otherBidderRequest := request
	expectedPMP := *request.Imp[0].PMP
	expectedPMP.Deals = append([]openrtb2.Deal(nil), request.Imp[0].PMP.Deals...)
	expectedPMP.Deals[0].BidFloor = 5
	expectedPMP.Deals[0].BidFloorCur = "USD"
	for iteration := 0; iteration < 2; iteration++ {
		requests, errs := qaBidder(t).MakeRequests(&request, qaCurrencyRates(2))
		if len(errs) != 0 || len(requests) != 2 {
			t.Fatalf("requests=%d errors=%v", len(requests), errs)
		}
		for group, outbound := range requests {
			for _, forbidden := range []string{"IDENTITY-", "192.0.2.9", "2001:db8::9", "45.12345", "1.54321"} {
				if strings.Contains(string(outbound.Body), forbidden) {
					t.Errorf("property group %d retains named identity %q", group, forbidden)
				}
			}
			var body openrtb2.BidRequest
			if err := json.Unmarshal(outbound.Body, &body); err != nil {
				t.Fatal(err)
			}
			if body.User == nil || body.User.Consent != "privacy-main" || string(body.User.Ext) != `{"consent":"privacy-legacy"}` {
				t.Fatalf("privacy consent not preserved in group %d: %#v", group, body.User)
			}
			if !reflect.DeepEqual(body.Regs, request.Regs) || !reflect.DeepEqual(body.Source, request.Source) || !reflect.DeepEqual(body.Cur, []string{"USD"}) {
				t.Errorf("privacy, auction transaction, supply chain or currency changed in group %d", group)
			}
			if body.Device == nil || body.Device.UA != "test-agent" || body.Device.DeviceType != 3 || body.Device.Language != "es" || body.Device.DNT == nil || *body.Device.DNT != 1 || body.Device.Lmt == nil || *body.Device.Lmt != 1 {
				t.Errorf("non-identity device context changed: %#v", body.Device)
			}
			if len(body.Device.Ext) != 0 {
				t.Errorf("device extension forwarded: %s", body.Device.Ext)
			}
			if group == 0 {
				if len(body.Imp) != 2 || body.Imp[0].ID != "banner" || body.Imp[1].ID != "native" || !reflect.DeepEqual(body.Imp[0].Banner, request.Imp[0].Banner) || !reflect.DeepEqual(body.Imp[1].Native, request.Imp[2].Native) || !reflect.DeepEqual(body.Imp[0].PMP, &expectedPMP) || body.Imp[0].BidFloor != 2.5 || body.Imp[0].BidFloorCur != "USD" {
					t.Errorf("Banner/Native, deals or floors changed: %#v", body.Imp)
				}
			} else if len(body.Imp) != 1 || body.Imp[0].ID != "video" || !reflect.DeepEqual(body.Imp[0].Video, request.Imp[1].Video) {
				t.Errorf("Video group changed: %#v", body.Imp)
			}
		}
		requests[0].Body[0] = '['
		for _, candidate := range []*openrtb2.BidRequest{&request, &otherBidderRequest} {
			after, err := json.Marshal(candidate)
			if err != nil || string(before) != string(after) {
				t.Fatal("sanitization or output mutation affected the publisher/other bidder input")
			}
		}
	}
}

func TestQAIdentityBoundaryHandlesNilAndEmpty(t *testing.T) {
	for _, tc := range []struct {
		name       string
		user       *openrtb2.User
		device     *openrtb2.Device
		wantUser   bool
		wantDevice bool
	}{
		{"nil", nil, nil, false, false},
		{"empty", &openrtb2.User{}, &openrtb2.Device{}, true, true},
		{"null extensions", &openrtb2.User{Ext: json.RawMessage(`null`)}, &openrtb2.Device{Ext: json.RawMessage(`null`)}, true, true},
		{"empty extensions", &openrtb2.User{Ext: json.RawMessage(`{}`)}, &openrtb2.Device{Ext: json.RawMessage(`{}`)}, true, true},
		{"identity only", &openrtb2.User{ID: "IDENTITY-user", Ext: json.RawMessage(`{"eids":[],"data":{"uid":"IDENTITY-user"}}`)}, &openrtb2.Device{IFA: "IDENTITY-device", Ext: json.RawMessage(`{"panxo":{"uid":"IDENTITY-device"}}`)}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"propertyKey":"p"}}`)}}, User: tc.user, Device: tc.device}
			before, _ := json.Marshal(request)
			requests, errs := qaBidder(t).MakeRequests(request, nil)
			if len(errs) != 0 || len(requests) != 1 {
				t.Fatalf("requests=%d errors=%v", len(requests), errs)
			}
			var body openrtb2.BidRequest
			if err := json.Unmarshal(requests[0].Body, &body); err != nil {
				t.Fatal(err)
			}
			if (body.User != nil) != tc.wantUser || (body.Device != nil) != tc.wantDevice || strings.Contains(string(requests[0].Body), "IDENTITY-") {
				t.Fatalf("unexpected nil/identity result: %s", requests[0].Body)
			}
			if body.User != nil && len(body.User.Ext) != 0 || body.Device != nil && len(body.Device.Ext) != 0 {
				t.Fatalf("empty identity extensions retained: %s", requests[0].Body)
			}
			after, _ := json.Marshal(request)
			if string(before) != string(after) {
				t.Fatal("input changed")
			}
		})
	}
}

func TestQAUserConsentAcceptsOnlyNonEmptyStrings(t *testing.T) {
	for _, value := range []string{`""`, `null`, `false`, `7`, `[]`, `{ "id": "IDENTITY-consent-object" }`, `"privacy-legacy"`} {
		t.Run(value, func(t *testing.T) {
			request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "imp", Ext: json.RawMessage(`{"bidder":{"propertyKey":"p"}}`)}}, User: &openrtb2.User{Consent: "privacy-main", Ext: json.RawMessage(`{"consent":` + value + `,"unknown":"IDENTITY-unknown"}`)}}
			before, _ := json.Marshal(request)
			requests, errs := qaBidder(t).MakeRequests(request, nil)
			if len(errs) != 0 || len(requests) != 1 {
				t.Fatalf("requests=%d errors=%v", len(requests), errs)
			}
			var body openrtb2.BidRequest
			if err := json.Unmarshal(requests[0].Body, &body); err != nil {
				t.Fatal(err)
			}
			if body.User == nil || body.User.Consent != "privacy-main" || strings.Contains(string(requests[0].Body), "IDENTITY-") {
				t.Fatalf("privacy consent damaged or identifier forwarded: %s", requests[0].Body)
			}
			if value == `"privacy-legacy"` {
				if string(body.User.Ext) != `{"consent":"privacy-legacy"}` {
					t.Fatalf("legacy consent missing: %s", body.User.Ext)
				}
			} else if len(body.User.Ext) != 0 {
				t.Fatalf("non-string/empty consent forwarded: %s", body.User.Ext)
			}
			after, _ := json.Marshal(request)
			if string(before) != string(after) {
				t.Fatal("input changed")
			}
		})
	}
}

func TestQAMalformedIdentityExtensionsFailWithoutEcho(t *testing.T) {
	for _, field := range []string{"user", "device"} {
		for _, raw := range []string{`"IDENTITY-secret"`, `["IDENTITY-secret"]`, `{"id":"IDENTITY-secret",`, `true`, `7`} {
			t.Run(field+"-"+raw, func(t *testing.T) {
				request := &openrtb2.BidRequest{Imp: []openrtb2.Imp{{ID: "a", Ext: json.RawMessage(`{"bidder":{"propertyKey":"p"}}`)}, {ID: "b", Ext: json.RawMessage(`{"bidder":{"propertyKey":"q"}}`)}}}
				if field == "user" {
					request.User = &openrtb2.User{Ext: json.RawMessage(raw)}
				} else {
					request.Device = &openrtb2.Device{Ext: json.RawMessage(raw)}
				}
				requests, errs := qaBidder(t).MakeRequests(request, nil)
				if len(requests) != 0 || len(errs) != 1 {
					t.Fatalf("invalid input generated requests=%d errors=%v", len(requests), errs)
				}
				var bad *errortypes.BadInput
				if !errors.As(errs[0], &bad) || bad.Message != "invalid "+field+".ext" || strings.Contains(errs[0].Error(), "IDENTITY-secret") {
					t.Fatalf("wrong or leaking diagnostic: %v", errs[0])
				}
				if field == "user" && string(request.User.Ext) != raw || field == "device" && string(request.Device.Ext) != raw {
					t.Fatal("error path changed caller extension")
				}
			})
		}
	}
}

func TestQAOmitsUnboundedExtensionsAndSiteURLs(t *testing.T) {
	for _, tc := range []struct {
		name string
		site *openrtb2.Site
	}{
		{"no site", nil},
		{"empty site", &openrtb2.Site{}},
		{"url path query and fragment", &openrtb2.Site{Domain: "publisher.example", Page: "https://publisher.example/member/account?reader=arbitrary#session", Ref: "https://referrer.example/profile/visitor?token=arbitrary#history", Keywords: "news"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := &openrtb2.BidRequest{
				ID: "auction", Site: tc.site, Cur: []string{"EUR"},
				Ext:    json.RawMessage(`{"future":{"nested":[{"publisherOwned":"any-value"}]},"prebid":{"unknown":true}}`),
				Device: &openrtb2.Device{UA: "browser", Language: "es", Ext: json.RawMessage(`{"viewport":1920,"unknown":{"nested":["arbitrary",3,true]}}`)},
				Imp: []openrtb2.Imp{
					{ID: "banner", Banner: &openrtb2.Banner{Format: []openrtb2.Format{{W: 300, H: 250}}}, BidFloor: 1.25, BidFloorCur: "EUR", Ext: json.RawMessage(`{"bidder":{"propertyKey":"one"},"gpid":"placement","tid":"arbitrary","data":{"future":[1,2,3]}}`)},
					{ID: "video", Video: &openrtb2.Video{MIMEs: []string{"video/mp4"}}, Ext: json.RawMessage(`{"bidder":{"propertyKey":"two"},"unknown":{"link":"any-member"}}`)},
				},
			}
			before, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			otherBidderRequest := *request
			for iteration := 0; iteration < 2; iteration++ {
				requests, errs := qaBidder(t).MakeRequests(request, qaCurrencyRates(2))
				if len(errs) != 0 || len(requests) != 2 {
					t.Fatalf("requests=%d errors=%v", len(requests), errs)
				}
				for group, outbound := range requests {
					var body openrtb2.BidRequest
					if err := json.Unmarshal(outbound.Body, &body); err != nil {
						t.Fatal(err)
					}
					if len(body.Ext) != 0 || body.Device == nil || len(body.Device.Ext) != 0 || len(body.Imp) != 1 || len(body.Imp[0].Ext) != 0 {
						t.Fatalf("arbitrary extensions forwarded: %s", outbound.Body)
					}
					if request.Site == nil {
						if body.Site != nil {
							t.Fatal("site invented")
						}
					} else {
						expected := *request.Site
						expected.Page = ""
						expected.Ref = ""
						if !reflect.DeepEqual(body.Site, &expected) {
							t.Fatalf("site context changed beyond URL omission: %#v", body.Site)
						}
					}
					if body.Device.UA != request.Device.UA || body.Device.Language != request.Device.Language || !reflect.DeepEqual(body.Cur, []string{"USD"}) {
						t.Fatal("remaining device context or currency changed")
					}
					if group == 0 && (body.Imp[0].BidFloor != 2.5 || body.Imp[0].BidFloorCur != "USD" || !reflect.DeepEqual(body.Imp[0].Banner, request.Imp[0].Banner)) {
						t.Fatal("banner or floor changed")
					}
					if group == 1 && !reflect.DeepEqual(body.Imp[0].Video, request.Imp[1].Video) {
						t.Fatal("video changed")
					}
					var wire map[string]json.RawMessage
					if err := json.Unmarshal(outbound.Body, &wire); err != nil {
						t.Fatal(err)
					}
					if _, present := wire["ext"]; present {
						t.Fatal("root ext key serialized")
					}
				}
				for _, input := range []*openrtb2.BidRequest{request, &otherBidderRequest} {
					after, err := json.Marshal(input)
					if err != nil || string(before) != string(after) {
						t.Fatal("publisher or another bidder input mutated")
					}
				}
			}
		})
	}
}
