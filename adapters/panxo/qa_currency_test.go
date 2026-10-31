package panxo

import (
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/currency"
	"github.com/prebid/prebid-server/v4/errortypes"
)

func qaCurrencyRates(rate float64) *adapters.ExtraRequestInfo {
	info := adapters.NewExtraRequestInfo(currency.NewRates(map[string]map[string]float64{
		"EUR": {"USD": rate},
	}))
	return &info
}

func TestQACurrencyConvertsTransportAndDealFloorsWithoutMutatingInput(t *testing.T) {
	var request openrtb2.BidRequest
	if err := json.Unmarshal([]byte(`{
		"id":"currency-auction","cur":["EUR","GBP"],
		"imp":[
			{"id":"foreign","bidfloor":1.25,"bidfloorcur":"EUR",
			 "banner":{"w":300,"h":250},
			 "pmp":{"private_auction":1,"deals":[{"id":"deal","bidfloor":2.5,"bidfloorcur":"EUR","at":1,"wseat":["buyer"]}]},
			 "ext":{"bidder":{"propertyKey":"one"}}},
			{"id":"zero","bidfloor":0,"bidfloorcur":"GBP",
			 "pmp":{"deals":[{"id":"free","bidfloor":0,"bidfloorcur":"GBP"}]},
			 "ext":{"bidder":{"propertyKey":"two"}}},
			{"id":"usd","bidfloor":3,"bidfloorcur":"USD","ext":{"bidder":{"propertyKey":"one"}}}
		]
	}`), &request); err != nil {
		t.Fatal(err)
	}
	before, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	shared := request
	for iteration := 0; iteration < 2; iteration++ {
		requests, errs := qaBidder(t).MakeRequests(&request, qaCurrencyRates(2))
		if len(errs) != 0 || len(requests) != 2 {
			t.Fatalf("requests=%d errors=%v", len(requests), errs)
		}
		var first, second openrtb2.BidRequest
		if err := json.Unmarshal(requests[0].Body, &first); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(requests[1].Body, &second); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(first.Cur, []string{"USD"}) || !reflect.DeepEqual(second.Cur, []string{"USD"}) {
			t.Fatalf("outbound currencies are not USD: %v, %v", first.Cur, second.Cur)
		}
		if !reflect.DeepEqual(requests[0].ImpIDs, []string{"foreign", "usd"}) ||
			!reflect.DeepEqual(requests[1].ImpIDs, []string{"zero"}) {
			t.Fatalf("property grouping changed: %v, %v", requests[0].ImpIDs, requests[1].ImpIDs)
		}
		if first.Imp[0].BidFloor != 2.5 || first.Imp[0].BidFloorCur != "USD" ||
			first.Imp[1].BidFloor != 3 || first.Imp[1].BidFloorCur != "USD" {
			t.Fatalf("floor amounts and currency are not aligned: %#v", first.Imp)
		}
		deal := first.Imp[0].PMP.Deals[0]
		if deal.ID != "deal" || deal.BidFloor != 5 || deal.BidFloorCur != "USD" ||
			deal.AT != 1 || !reflect.DeepEqual(deal.WSeat, []string{"buyer"}) ||
			!reflect.DeepEqual(first.Imp[0].PMP.PrivateAuction, request.Imp[0].PMP.PrivateAuction) {
			t.Fatalf("deal amount, currency or metadata changed incorrectly: %#v", first.Imp[0].PMP)
		}
		if second.Imp[0].BidFloor != 0 || second.Imp[0].BidFloorCur != "USD" ||
			second.Imp[0].PMP.Deals[0].BidFloor != 0 || second.Imp[0].PMP.Deals[0].BidFloorCur != "USD" {
			t.Fatalf("zero floors require no GBP conversion: %#v", second.Imp[0])
		}
		for _, original := range []*openrtb2.BidRequest{&request, &shared} {
			after, err := json.Marshal(original)
			if err != nil || string(after) != string(before) {
				t.Fatal("conversion mutated the publisher request or another bidder's shared pointers")
			}
		}
	}
}

func TestQACurrencyDefaultsAndZeroDoNotRequireRates(t *testing.T) {
	for _, tc := range []struct {
		name  string
		cur   string
		floor float64
	}{
		{name: "implicit USD", floor: 1.25},
		{name: "explicit USD", cur: "USD", floor: 1.25},
		{name: "normalized USD", cur: " usd ", floor: 1.25},
		{name: "zero EUR", cur: "EUR"},
		{name: "zero GBP", cur: "GBP"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := &openrtb2.BidRequest{ID: "defaults", Imp: []openrtb2.Imp{{
				ID: "imp", BidFloor: tc.floor, BidFloorCur: tc.cur,
				Ext: json.RawMessage(`{"bidder":{"propertyKey":"one"}}`),
				PMP: &openrtb2.PMP{Deals: []openrtb2.Deal{{
					ID: "deal", BidFloor: tc.floor, BidFloorCur: tc.cur,
				}}},
			}}}
			requests, errs := qaBidder(t).MakeRequests(request, nil)
			if len(errs) != 0 || len(requests) != 1 {
				t.Fatalf("requests=%d errors=%v", len(requests), errs)
			}
			var body openrtb2.BidRequest
			if err := json.Unmarshal(requests[0].Body, &body); err != nil {
				t.Fatal(err)
			}
			if len(body.Cur) != 0 && !reflect.DeepEqual(body.Cur, []string{"USD"}) {
				t.Fatalf("effective request currency is not USD: %v", body.Cur)
			}
			wantCur := "USD"
			if tc.cur == "" {
				wantCur = ""
			}
			if body.Imp[0].BidFloor != tc.floor || body.Imp[0].BidFloorCur != wantCur ||
				body.Imp[0].PMP.Deals[0].BidFloor != tc.floor || body.Imp[0].PMP.Deals[0].BidFloorCur != wantCur {
				t.Fatalf("default/zero floor was not preserved in effective USD: %#v", body.Imp[0])
			}
		})
	}
}

func TestQACurrencyRejectsOnlyTheUnconvertibleImpression(t *testing.T) {
	for _, tc := range []struct {
		name  string
		info  *adapters.ExtraRequestInfo
		floor float64
		cur   string
	}{
		{name: "missing extra info", floor: 1, cur: "EUR"},
		{name: "missing rates", info: &adapters.ExtraRequestInfo{}, floor: 1, cur: "EUR"},
		{name: "missing mapping", info: qaCurrencyRates(2), floor: 1, cur: "GBP"},
		{name: "zero rate", info: qaCurrencyRates(0), floor: 1, cur: "EUR"},
		{name: "negative rate", info: qaCurrencyRates(-1), floor: 1, cur: "EUR"},
		{name: "NaN rate", info: qaCurrencyRates(math.NaN()), floor: 1, cur: "EUR"},
		{name: "infinite rate", info: qaCurrencyRates(math.Inf(1)), floor: 1, cur: "EUR"},
		{name: "overflow", info: qaCurrencyRates(math.MaxFloat64), floor: 2, cur: "EUR"},
		{name: "unknown currency", info: qaCurrencyRates(2), floor: 1, cur: "not-a-currency"},
		{name: "negative floor", floor: -1, cur: "USD"},
		{name: "NaN floor", floor: math.NaN(), cur: "USD"},
		{name: "infinite floor", floor: math.Inf(1), cur: "USD"},
	} {
		for _, placement := range []string{"impression", "deal"} {
			t.Run(tc.name+"/"+placement, func(t *testing.T) {
				bad := openrtb2.Imp{ID: "bad", Ext: json.RawMessage(`{"bidder":{"propertyKey":"one"}}`)}
				if placement == "deal" {
					bad.PMP = &openrtb2.PMP{Deals: []openrtb2.Deal{{ID: "bad-deal", BidFloor: tc.floor, BidFloorCur: tc.cur}}}
				} else {
					bad.BidFloor, bad.BidFloorCur = tc.floor, tc.cur
				}
				good := openrtb2.Imp{ID: "good", BidFloor: 1, Ext: json.RawMessage(`{"bidder":{"propertyKey":"one"}}`)}
				request := &openrtb2.BidRequest{ID: "partial", Cur: []string{"EUR"}, Imp: []openrtb2.Imp{bad, good}}
				requests, errs := qaBidder(t).MakeRequests(request, tc.info)
				var badInput *errortypes.BadInput
				if len(errs) != 1 || !errors.As(errs[0], &badInput) {
					t.Fatalf("expected one BadInput for unconvertible floor: %v", errs)
				}
				if len(requests) != 1 || !reflect.DeepEqual(requests[0].ImpIDs, []string{"good"}) {
					t.Fatalf("bad impression was retained or good sibling lost: %#v", requests)
				}
				var body openrtb2.BidRequest
				if err := json.Unmarshal(requests[0].Body, &body); err != nil {
					t.Fatal(err)
				}
				if len(body.Imp) != 1 || body.Imp[0].ID != "good" || body.Imp[0].BidFloor != 1 {
					t.Fatalf("unexpected survivor: %#v", body.Imp)
				}
				if request.Cur[0] != "EUR" || len(request.Imp) != 2 || request.Imp[0].ID != "bad" {
					t.Fatal("input mutated while omitting an invalid floor")
				}
			})
		}
	}
}

func TestQACurrencyNoEmptyRequestWhenAllImpressionsAreRejected(t *testing.T) {
	request := &openrtb2.BidRequest{ID: "unconvertible", Imp: []openrtb2.Imp{{
		ID: "imp", BidFloor: 1, BidFloorCur: "EUR",
		Ext: json.RawMessage(`{"bidder":{"propertyKey":"one"}}`),
	}}}
	requests, errs := qaBidder(t).MakeRequests(request, nil)
	if len(requests) != 0 || len(errs) != 1 {
		t.Fatalf("empty group was sent or error lost: requests=%d errors=%v", len(requests), errs)
	}
}
