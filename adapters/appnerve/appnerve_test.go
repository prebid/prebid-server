package appnerve

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJSONSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "appnervetest", &adapter{endpoint: "https://exchange.appnerve.net/openrtb2/auction"})
}

func TestBuilderEndpoint(t *testing.T) {
	if _, err := Builder(openrtb_ext.BidderAppnerve, config.Adapter{Endpoint: "://bad"}, config.Server{}); err == nil {
		t.Fatal("accepted an invalid endpoint")
	}
	bidder, err := Builder(openrtb_ext.BidderAppnerve, config.Adapter{Endpoint: "https://exchange.appnerve.net/openrtb2/auction"}, config.Server{})
	if err != nil || bidder == nil {
		t.Fatal(err)
	}
}

func testRequest() *openrtb2.BidRequest {
	return &openrtb2.BidRequest{ID: "request", TMax: 1200, Imp: []openrtb2.Imp{
		{ID: "one", Banner: &openrtb2.Banner{}, Ext: json.RawMessage(`{"bidder":{"sourceId":"source-a"},"tid":"publisher-tid","data":{"section":"news"}}`)},
		{ID: "two", Video: &openrtb2.Video{}, Ext: json.RawMessage(`{"bidder":{"placementId":"source-b"}}`)},
	}}
}

func TestMakeRequestsSplitPreserveAndValidate(t *testing.T) {
	a := &adapter{endpoint: "https://exchange.appnerve.net/openrtb2/auction"}
	original := testRequest()
	before, _ := json.Marshal(original)
	requests, errs := a.MakeRequests(original, nil)
	if len(errs) != 0 || len(requests) != 2 {
		t.Fatalf("requests=%d errors=%v", len(requests), errs)
	}
	for index, sent := range requests {
		var body openrtb2.BidRequest
		if err := json.Unmarshal(sent.Body, &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Imp) != 1 || body.TMax != original.TMax || body.Imp[0].ID != original.Imp[index].ID {
			t.Fatal("impression/deadline lost")
		}
		if sent.Method != http.MethodPost || sent.Headers.Get("Content-Type") != "application/json" {
			t.Fatal("invalid request headers")
		}
		if !reflect.DeepEqual(sent.ImpIDs, []string{body.Imp[0].ID}) {
			t.Fatal("invalid impression association")
		}
	}
	var first map[string]any
	json.Unmarshal(requests[0].Body, &first)
	ext := first["imp"].([]any)[0].(map[string]any)["ext"].(map[string]any)
	if ext["tid"] != "publisher-tid" || ext["data"] == nil || ext["bidder"] != nil {
		t.Fatal("publisher fields or bidder parameters handled incorrectly")
	}
	after, _ := json.Marshal(original)
	if string(before) != string(after) {
		t.Fatal("mutated publisher request")
	}
	original.Imp[0].Ext = json.RawMessage(`{"bidder":{}}`)
	requests, errs = a.MakeRequests(original, nil)
	if len(errs) != 1 || len(requests) != 1 {
		t.Fatal("invalid impression must not suppress valid impression")
	}
}

func TestResponses(t *testing.T) {
	a := &adapter{endpoint: "https://exchange.appnerve.net/openrtb2/auction"}
	sent, _ := a.MakeRequests(testRequest(), nil)
	cases := []struct {
		name   string
		status int
		body   string
		bids   int
		errors bool
	}{
		{"banner", 200, `{"cur":"USD","seatbid":[{"bid":[{"id":"b","impid":"one","price":1,"adm":"<div>Ad</div>"}]}]}`, 1, false},
		{"no content", 204, "", 0, false},
		{"no bid", 200, `{"seatbid":[]}`, 0, false},
		{"malformed", 200, `not-json`, 0, true},
		{"http error", 500, `{}`, 0, true},
		{"unknown imp", 200, `{"seatbid":[{"bid":[{"id":"b","impid":"other","price":1,"adm":"Ad"}]}]}`, 0, true},
		{"zero price", 200, `{"seatbid":[{"bid":[{"id":"b","impid":"one","price":0,"adm":"Ad"}]}]}`, 0, true},
		{"wrong type", 200, `{"seatbid":[{"bid":[{"id":"b","impid":"one","price":1,"adm":"Ad","ext":{"prebid":{"type":"video"}}}]}]}`, 0, true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			result, errs := a.MakeBids(testRequest(), sent[0], &adapters.ResponseData{StatusCode: item.status, Body: []byte(item.body)})
			count := 0
			if result != nil {
				count = len(result.Bids)
			}
			if count != item.bids || (len(errs) > 0) != item.errors {
				t.Fatalf("bids=%d errors=%v", count, errs)
			}
		})
	}
}

func TestMediaTypes(t *testing.T) {
	for _, kind := range []string{"banner", "video", "native", "audio"} {
		imp := openrtb2.Imp{Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}, Native: &openrtb2.Native{}, Audio: &openrtb2.Audio{}}
		bid := openrtb2.Bid{Ext: json.RawMessage(`{"prebid":{"type":"` + kind + `"}}`)}
		actual, err := bidType(bid, imp)
		if err != nil || actual != openrtb_ext.BidType(kind) {
			t.Fatalf("%s: %s %v", kind, actual, err)
		}
	}
	if _, err := bidType(openrtb2.Bid{}, openrtb2.Imp{Banner: &openrtb2.Banner{}, Video: &openrtb2.Video{}}); err == nil {
		t.Fatal("ambiguous type accepted")
	}
}
