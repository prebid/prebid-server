package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/panxo"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/currency"
	"github.com/prebid/prebid-server/v4/experiment/adscert"
	"github.com/prebid/prebid-server/v4/hooks/hookexecution"
	metricsConfig "github.com/prebid/prebid-server/v4/metrics/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

type qaNoNetworkTransport struct{}

func (qaNoNetworkTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, fmt.Errorf("QA forbids all HTTP calls")
}

type qaCurrencyTransport func(*http.Request) (*http.Response, error)

func (transport qaCurrencyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

func TestQAPanxoCurrencyThroughExchange(t *testing.T) {
	for _, available := range []bool{true, false} {
		t.Run(fmt.Sprintf("conversion-available-%t", available), func(t *testing.T) {
			impl, err := panxo.Builder(openrtb_ext.BidderPanxo, config.Adapter{Endpoint: "https://reviewer.invalid/bid"}, config.Server{})
			if err != nil {
				t.Fatal(err)
			}
			rates := map[string]map[string]float64{}
			floorCurrency, expectedWireFloor := "USD", 1.25
			if available {
				// Explicitly synthetic: one EUR buys two USD in this fixture.
				rates["EUR"] = map[string]float64{"USD": 2}
				floorCurrency, expectedWireFloor = "EUR", 2.5
			}
			conversions := currency.NewRates(rates)
			info := adapters.NewExtraRequestInfo(conversions)
			width, height := int64(300), int64(250)
			request := &openrtb2.BidRequest{
				ID: "currency-auction", Cur: []string{"EUR"},
				Site: &openrtb2.Site{Domain: "publisher.invalid"},
				Imp: []openrtb2.Imp{{
					ID: "imp", BidFloor: 1.25, BidFloorCur: floorCurrency,
					Banner: &openrtb2.Banner{W: &width, H: &height},
					Ext:    json.RawMessage(`{"bidder":{"propertyKey":"property"}}`),
				}},
			}
			before, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			responseBody := []byte(`{"id":"currency-auction","cur":"USD","seatbid":[{"bid":[{
				"id":"bid","impid":"imp","price":5,"mtype":1,"adm":"synthetic-markup",
				"nurl":"https://notice.invalid/win?price=${AUCTION_PRICE}"
			}]}]}`)
			observedBodies := make(chan []byte, 1)
			client := &http.Client{Transport: qaCurrencyTransport(func(outbound *http.Request) (*http.Response, error) {
				if outbound.URL.Host != "reviewer.invalid" || outbound.URL.Query().Get("key") != "property" {
					return nil, fmt.Errorf("unexpected destination in inert QA transport")
				}
				body, err := io.ReadAll(outbound.Body)
				if err != nil {
					return nil, err
				}
				select {
				case observedBodies <- body:
				default:
					return nil, fmt.Errorf("unexpected additional request in inert QA transport")
				}
				return &http.Response{
					StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}},
					Body: io.NopCloser(bytes.NewReader(responseBody)),
				}, nil
			})}
			bidder := AdaptBidder(impl, client, &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderPanxo, nil, "")
			seatBids, _, errs := bidder.requestBid(context.Background(), BidderRequest{
				BidRequest: request, BidderName: openrtb_ext.BidderPanxo, BidderCoreName: openrtb_ext.BidderPanxo,
			}, conversions, &info, &adscert.NilSigner{},
				bidRequestOptions{bidAdjustments: map[string]float64{"panxo": 1}},
				openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{}, nil)
			select {
			case body := <-observedBodies:
				var wire openrtb2.BidRequest
				if err := json.Unmarshal(body, &wire); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(wire.Cur, []string{"USD"}) || len(wire.Imp) != 1 ||
					wire.Imp[0].BidFloorCur != "USD" || wire.Imp[0].BidFloor != expectedWireFloor {
					t.Fatalf("wrong outbound price/currency pairing: %s", body)
				}
			default:
				t.Fatal("request did not pass through MakeRequests and the inert transport")
			}
			after, err := json.Marshal(request)
			if err != nil || string(after) != string(before) {
				t.Fatal("adapter/core changed original EUR request or publisher floor")
			}
			if !available {
				if len(errs) == 0 || len(seatBids) > 0 && len(seatBids[0].Bids) > 0 {
					t.Fatalf("core accepted a USD bid without an EUR conversion: bids=%#v errors=%v", seatBids, errs)
				}
				return
			}
			if len(errs) != 0 || len(seatBids) != 1 || len(seatBids[0].Bids) != 1 {
				t.Fatalf("expected one converted bid: bids=%#v errors=%v", seatBids, errs)
			}
			bid := seatBids[0].Bids[0]
			if seatBids[0].Currency != "EUR" || bid.Bid.Price != 2.5 ||
				bid.OriginalBidCPM != 5 || bid.OriginalBidCur != "USD" {
				t.Fatalf("final/core and original/billing amounts disagree: seat=%#v bid=%#v", seatBids[0], bid)
			}
			if bid.Bid.NURL != "https://notice.invalid/win?price=${AUCTION_PRICE}" {
				t.Fatal("currency conversion changed the notice before its proper handling stage")
			}
		})
	}
}

func TestQAPanxoStoredBidResponses(t *testing.T) {
	for _, mtype := range []openrtb2.MarkupType{0, openrtb2.MarkupBanner, openrtb2.MarkupVideo, openrtb2.MarkupNative} {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("mtype-%d-replace-%t", mtype, replace), func(t *testing.T) {
				impl, err := panxo.Builder(openrtb_ext.BidderPanxo, config.Adapter{Endpoint: "https://reviewer.invalid/bid"}, config.Server{})
				if err != nil {
					t.Fatal(err)
				}
				client := &http.Client{Transport: qaNoNetworkTransport{}}
				bidder := AdaptBidder(impl, client, &config.Configuration{}, &metricsConfig.NilMetricsEngine{}, openrtb_ext.BidderPanxo, nil, "")
				converter := currency.NewRateConverter(client, time.Second, "", 0)
				body, _ := json.Marshal(openrtb2.BidResponse{ID: "stored-response", Cur: "USD", SeatBid: []openrtb2.SeatBid{{Bid: []openrtb2.Bid{{ID: "stored-bid", ImpID: "stored-imp", Price: 1.25, MType: mtype, AdM: "stored-markup"}}}}})
				seatBids, _, errs := bidder.requestBid(context.Background(), BidderRequest{
					BidRequest: &openrtb2.BidRequest{ID: "live-auction", Imp: nil, Site: &openrtb2.Site{Domain: "publisher.invalid"}},
					BidderName: openrtb_ext.BidderPanxo, BidderCoreName: openrtb_ext.BidderPanxo,
					BidderStoredResponses: map[string]json.RawMessage{"live-imp": body},
					ImpReplaceImpId:       map[string]bool{"live-imp": replace},
				}, converter.Rates(), &adapters.ExtraRequestInfo{}, &adscert.NilSigner{}, bidRequestOptions{bidAdjustments: map[string]float64{"panxo": 1}}, openrtb_ext.ExtAlternateBidderCodes{}, &hookexecution.EmptyHookExecutor{}, nil)
				if mtype == 0 {
					if len(errs) == 0 || (len(seatBids) > 0 && len(seatBids[0].Bids) > 0) {
						t.Fatalf("stored response without mtype cannot infer absent impression media: bids=%#v errs=%v", seatBids, errs)
					}
					return
				}
				if len(errs) > 0 {
					t.Errorf("stored response rejected: %v", errs)
				}
				if len(seatBids) != 1 || len(seatBids[0].Bids) != 1 {
					t.Fatalf("want one stored bid, got %#v", seatBids)
				}
				wantImp := "stored-imp"
				if replace {
					wantImp = "live-imp"
				}
				if got := seatBids[0].Bids[0].Bid.ImpID; got != wantImp {
					t.Errorf("want imp %s got %s", wantImp, got)
				}
			})
		}
	}
}
