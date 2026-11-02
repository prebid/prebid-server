package adelerate

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/errortypes"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(openrtb_ext.BidderAdelerate, config.Adapter{
		Endpoint: "https://pbs.bidelerate.com/auction"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "adeleratetest", bidder)
}

func TestGetMediaTypeForBid(t *testing.T) {
	tests := []struct {
		name             string
		bid              openrtb2.Bid
		expectedBidType  openrtb_ext.BidType
		expectedErrorMsg string
	}{
		{
			name:            "banner",
			bid:             openrtb2.Bid{ID: "banner-bid", MType: openrtb2.MarkupBanner},
			expectedBidType: openrtb_ext.BidTypeBanner,
		},
		{
			name:            "video",
			bid:             openrtb2.Bid{ID: "video-bid", MType: openrtb2.MarkupVideo},
			expectedBidType: openrtb_ext.BidTypeVideo,
		},
		{
			name:            "native",
			bid:             openrtb2.Bid{ID: "native-bid", MType: openrtb2.MarkupNative},
			expectedBidType: openrtb_ext.BidTypeNative,
		},
		{
			name:             "missing mtype",
			bid:              openrtb2.Bid{ID: "missing-mtype-bid"},
			expectedErrorMsg: "bid missing-mtype-bid missing supported mtype or ext.prebid.type",
		},
		{
			name:             "unsupported mtype",
			bid:              openrtb2.Bid{ID: "unsupported-mtype-bid", MType: 99},
			expectedErrorMsg: "unsupported mtype 99 for bid unsupported-mtype-bid",
		},
		{name: "banner from PBS extension", bid: openrtb2.Bid{Ext: json.RawMessage(`{"prebid":{"type":"banner"}}`)}, expectedBidType: openrtb_ext.BidTypeBanner},
		{name: "video from PBS extension", bid: openrtb2.Bid{Ext: json.RawMessage(`{"prebid":{"type":"video"}}`)}, expectedBidType: openrtb_ext.BidTypeVideo},
		{name: "native from PBS extension", bid: openrtb2.Bid{Ext: json.RawMessage(`{"prebid":{"type":"native"}}`)}, expectedBidType: openrtb_ext.BidTypeNative},
		{name: "mtype takes precedence", bid: openrtb2.Bid{MType: openrtb2.MarkupBanner, Ext: json.RawMessage(`{"prebid":{"type":"video"}}`)}, expectedBidType: openrtb_ext.BidTypeBanner},
		{name: "unsupported mtype cannot fall back", bid: openrtb2.Bid{ID: "bad", MType: 99, Ext: json.RawMessage(`{"prebid":{"type":"banner"}}`)}, expectedErrorMsg: "unsupported mtype 99 for bid bad"},
		{name: "unsupported PBS type", bid: openrtb2.Bid{ID: "bad", Ext: json.RawMessage(`{"prebid":{"type":"audio"}}`)}, expectedErrorMsg: "bid bad missing supported mtype or ext.prebid.type"},
		{name: "malformed PBS extension", bid: openrtb2.Bid{ID: "bad", Ext: json.RawMessage(`{`)}, expectedErrorMsg: "bid bad missing supported mtype or ext.prebid.type"},
		{name: "empty PBS extension", bid: openrtb2.Bid{ID: "bad", Ext: json.RawMessage(`{"prebid":{}}`)}, expectedErrorMsg: "bid bad missing supported mtype or ext.prebid.type"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bidType, err := getMediaTypeForBid(test.bid)
			if test.expectedErrorMsg == "" {
				if err != nil {
					t.Fatalf("getMediaTypeForBid returned unexpected error: %v", err)
				}
				if bidType != test.expectedBidType {
					t.Errorf("getMediaTypeForBid returned bid type %q, expected %q", bidType, test.expectedBidType)
				}
				return
			}

			if err == nil {
				t.Fatal("getMediaTypeForBid expected an error, got nil")
			}
			if err.Error() != test.expectedErrorMsg {
				t.Errorf("getMediaTypeForBid returned error %q, expected %q", err.Error(), test.expectedErrorMsg)
			}

			var badServerResponse *errortypes.BadServerResponse
			if !errors.As(err, &badServerResponse) {
				t.Errorf("getMediaTypeForBid returned error type %T, expected *errortypes.BadServerResponse", err)
			}
		})
	}
}

func TestMakeRequestsPublisherAccounts(t *testing.T) {
	for _, app := range []bool{false, true} {
		t.Run(fmt.Sprintf("app=%t", app), func(t *testing.T) {
			request := &openrtb2.BidRequest{
				ID:  "request",
				Ext: json.RawMessage(`{"prebid":{"storedrequest":{"id":"outer"},"cache":{"bids":{}},"targeting":{},"aliases":{"outer":"adelerate"}},"custom":"keep"}`),
				Imp: []openrtb2.Imp{
					{ID: "one", Secure: openrtb2.Int8Ptr(0), Ext: json.RawMessage(`{"bidder":{"placementId":"placement-one","publisherId":"account-one"},"prebid":{"bidder":{"adelerate":{}},"storedrequest":{"id":"outer"}},"context":{"data":{"key":"value"}}}`)},
					{ID: "two", Secure: openrtb2.Int8Ptr(1), Ext: json.RawMessage(`{"bidder":{"placementId":"placement-two","publisherId":"account-two"}}`)},
					{ID: "three", Ext: json.RawMessage(`{"bidder":{"placementId":"placement-three","publisherId":"account-one"}}`)},
					{ID: "bad", Ext: json.RawMessage(`{"bidder":{"placementId":"missing-publisher"}}`)},
				},
			}
			publisher := &openrtb2.Publisher{ID: "outer-publisher", Ext: json.RawMessage(`{"prebid":{"parentAccount":"outer-account"}}`)}
			if app {
				request.App = &openrtb2.App{ID: "app-id", Bundle: "app.bundle", Publisher: publisher}
			} else {
				request.Site = &openrtb2.Site{Domain: "example.com", Publisher: publisher}
			}
			before, _ := json.Marshal(request)
			bidder := &adapter{endpoint: "https://example.com/auction"}
			requests, errs := bidder.MakeRequests(request, nil)
			if len(errs) != 1 || len(requests) != 2 {
				t.Fatalf("expected two account requests and one invalid impression, got %d requests, %v", len(requests), errs)
			}
			for i, req := range requests {
				var outgoing openrtb2.BidRequest
				if err := json.Unmarshal(req.Body, &outgoing); err != nil {
					t.Fatal(err)
				}
				expectedIDs := [][]string{{"one", "three"}, {"two"}}[i]
				if !reflect.DeepEqual(req.ImpIDs, expectedIDs) {
					t.Fatalf("unexpected impression groups: %v", req.ImpIDs)
				}
				outPublisher := outgoing.Site
				var account *openrtb2.Publisher
				if app {
					account = outgoing.App.Publisher
					if outgoing.App.Bundle != "app.bundle" {
						t.Fatal("lost app context")
					}
				} else {
					account = outPublisher.Publisher
					if outPublisher.Domain != "example.com" {
						t.Fatal("lost site context")
					}
				}
				if account.ID != []string{"account-one", "account-two"}[i] || len(account.Ext) != 0 {
					t.Fatalf("wrong PBS account: %+v", account)
				}
				if string(outgoing.Ext) != `{"custom":"keep"}` {
					t.Fatalf("outer PBS config leaked: %s", outgoing.Ext)
				}
				for _, imp := range outgoing.Imp {
					var ext map[string]json.RawMessage
					if err := json.Unmarshal(imp.Ext, &ext); err != nil {
						t.Fatal(err)
					}
					if _, exists := ext["bidder"]; exists {
						t.Fatal("internal bidder wrapper leaked")
					}
					var prebid openrtb_ext.ExtImpPrebid
					if err := json.Unmarshal(ext["prebid"], &prebid); err != nil {
						t.Fatal(err)
					}
					if prebid.StoredRequest == nil || prebid.StoredRequest.ID != "placement-"+imp.ID || len(prebid.Bidder) != 0 {
						t.Fatalf("wrong stored impression: %s", imp.Ext)
					}
					if *imp.Secure != map[string]int8{"one": 0, "two": 1, "three": 1}[imp.ID] {
						t.Fatal("publisher secure signal changed")
					}
					if imp.ID == "one" && len(ext["context"]) == 0 {
						t.Fatal("lost impression context")
					}
				}
			}
			after, _ := json.Marshal(request)
			if string(before) != string(after) {
				t.Fatal("MakeRequests mutated the original auction")
			}
		})
	}
}
