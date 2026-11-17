package panxo

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v4/adapters"
	"github.com/prebid/prebid-server/v4/adapters/adapterstest"
	"github.com/prebid/prebid-server/v4/config"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestJsonSamples(t *testing.T) {
	bidder, buildErr := Builder(
		openrtb_ext.BidderPanxo,
		config.Adapter{Endpoint: "https://panxo-sys.com/openrtb/2.6/bid"},
		config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"},
	)

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error: %v", buildErr)
	}

	adapterstest.RunJSONBidderTest(t, "panxotest", bidder)
}

func TestMakeRequestsGroupsPropertyKeysWithoutMutatingInput(t *testing.T) {
	bidder, err := Builder(
		openrtb_ext.BidderPanxo,
		config.Adapter{Endpoint: "https://panxo-sys.com/openrtb/2.6/bid"},
		config.Server{},
	)
	if err != nil {
		t.Fatalf("Builder returned unexpected error: %v", err)
	}

	originalExt := json.RawMessage(`{"bidder":{"propertyKey":"property one"},"gpid":"slot-1"}`)
	request := &openrtb2.BidRequest{
		ID: "mixed-properties",
		Imp: []openrtb2.Imp{
			{ID: "imp-1", Ext: originalExt},
			{ID: "imp-2", Ext: json.RawMessage(`{"bidder":{"propertyKey":"property-two"},"tid":"transaction-2"}`)},
			{ID: "imp-3", Ext: json.RawMessage(`{"bidder":{"propertyKey":"property one"},"gpid":"slot-3"}`)},
		},
	}

	requests, makeRequestErrs := bidder.MakeRequests(request, nil)
	if len(makeRequestErrs) != 0 {
		t.Fatalf("MakeRequests returned unexpected errors: %v", makeRequestErrs)
	}
	if len(requests) != 2 {
		t.Fatalf("expected 2 grouped requests, got %d", len(requests))
	}

	if requests[0].Uri != "https://panxo-sys.com/openrtb/2.6/bid?key=property+one&source=prebid-server" {
		t.Errorf("unexpected first URI: %s", requests[0].Uri)
	}
	if strings.Join(requests[0].ImpIDs, ",") != "imp-1,imp-3" {
		t.Errorf("unexpected first request imp IDs: %v", requests[0].ImpIDs)
	}
	if requests[1].Uri != "https://panxo-sys.com/openrtb/2.6/bid?key=property-two&source=prebid-server" {
		t.Errorf("unexpected second URI: %s", requests[1].Uri)
	}
	if strings.Join(requests[1].ImpIDs, ",") != "imp-2" {
		t.Errorf("unexpected second request imp IDs: %v", requests[1].ImpIDs)
	}

	var firstBody openrtb2.BidRequest
	if err := json.Unmarshal(requests[0].Body, &firstBody); err != nil {
		t.Fatalf("could not decode first request: %v", err)
	}
	if len(firstBody.Imp) != 2 || strings.Contains(string(firstBody.Imp[0].Ext), "bidder") {
		t.Errorf("first request was not grouped and sanitized: %s", requests[0].Body)
	}
	for _, imp := range firstBody.Imp {
		if len(imp.Ext) != 0 {
			t.Errorf("impression extension forwarded: %s", imp.Ext)
		}
	}
	if string(request.Imp[0].Ext) != string(originalExt) {
		t.Errorf("MakeRequests mutated the input extension: %s", request.Imp[0].Ext)
	}
}

func TestMakeBidsTypesBannerVideoAndNative(t *testing.T) {
	bidder, err := Builder(
		openrtb_ext.BidderPanxo,
		config.Adapter{Endpoint: "https://panxo-sys.com/openrtb/2.6/bid"},
		config.Server{},
	)
	if err != nil {
		t.Fatalf("Builder returned unexpected error: %v", err)
	}

	banner := &openrtb2.Banner{}
	video := &openrtb2.Video{}
	native := &openrtb2.Native{Request: `{"assets":[{"id":1}]}`}
	tests := []struct {
		name        string
		imp         openrtb2.Imp
		mtype       openrtb2.MarkupType
		expected    openrtb_ext.BidType
		expectError bool
	}{
		{name: "explicit banner", imp: openrtb2.Imp{ID: "imp-1", Banner: banner}, mtype: openrtb2.MarkupBanner, expected: openrtb_ext.BidTypeBanner},
		{name: "explicit video", imp: openrtb2.Imp{ID: "imp-1", Video: video}, mtype: openrtb2.MarkupVideo, expected: openrtb_ext.BidTypeVideo},
		{name: "explicit native", imp: openrtb2.Imp{ID: "imp-1", Native: native}, mtype: openrtb2.MarkupNative, expected: openrtb_ext.BidTypeNative},
		{name: "infer banner", imp: openrtb2.Imp{ID: "imp-1", Banner: banner}, expected: openrtb_ext.BidTypeBanner},
		{name: "infer video", imp: openrtb2.Imp{ID: "imp-1", Video: video}, expected: openrtb_ext.BidTypeVideo},
		{name: "infer native", imp: openrtb2.Imp{ID: "imp-1", Native: native}, expected: openrtb_ext.BidTypeNative},
		{name: "reject ambiguous omitted mtype", imp: openrtb2.Imp{ID: "imp-1", Banner: banner, Video: video}, expectError: true},
		{name: "reject mismatched type", imp: openrtb2.Imp{ID: "imp-1", Banner: banner}, mtype: openrtb2.MarkupVideo, expectError: true},
		{name: "reject audio", imp: openrtb2.Imp{ID: "imp-1", Banner: banner}, mtype: openrtb2.MarkupAudio, expectError: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			responseBody, marshalErr := json.Marshal(openrtb2.BidResponse{
				ID:  "request-1",
				Cur: "EUR",
				SeatBid: []openrtb2.SeatBid{{
					Bid: []openrtb2.Bid{{
						ID:    "bid-1",
						ImpID: "imp-1",
						Price: 1.5,
						MType: test.mtype,
					}},
				}},
			})
			if marshalErr != nil {
				t.Fatalf("could not encode response: %v", marshalErr)
			}

			bidResponse, responseErrs := bidder.MakeBids(
				&openrtb2.BidRequest{ID: "request-1", Imp: []openrtb2.Imp{test.imp}},
				&adapters.RequestData{ImpIDs: []string{"imp-1"}},
				&adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody},
			)

			if test.expectError {
				if len(responseErrs) != 1 {
					t.Fatalf("expected one response error, got %v", responseErrs)
				}
				if bidResponse == nil || len(bidResponse.Bids) != 0 {
					t.Fatalf("expected the invalid bid to be dropped, got %#v", bidResponse)
				}
				return
			}

			if len(responseErrs) != 0 {
				t.Fatalf("unexpected response errors: %v", responseErrs)
			}
			if bidResponse == nil || len(bidResponse.Bids) != 1 {
				t.Fatalf("expected one bid, got %#v", bidResponse)
			}
			if bidResponse.Bids[0].BidType != test.expected {
				t.Errorf("expected bid type %s, got %s", test.expected, bidResponse.Bids[0].BidType)
			}
			if bidResponse.Currency != "EUR" {
				t.Errorf("expected response currency EUR, got %s", bidResponse.Currency)
			}
		})
	}
}

func TestMakeBidsRejectsImpressionOutsideGroupedRequest(t *testing.T) {
	bidder, err := Builder(
		openrtb_ext.BidderPanxo,
		config.Adapter{Endpoint: "https://panxo-sys.com/openrtb/2.6/bid"},
		config.Server{},
	)
	if err != nil {
		t.Fatalf("Builder returned unexpected error: %v", err)
	}

	responseBody := []byte(`{"id":"request-1","seatbid":[{"bid":[{"id":"bid-2","impid":"imp-2","price":1.5,"mtype":1}]}]}`)
	bidResponse, responseErrs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "request-1", Imp: []openrtb2.Imp{
			{ID: "imp-1", Banner: &openrtb2.Banner{}},
			{ID: "imp-2", Banner: &openrtb2.Banner{}},
		}},
		&adapters.RequestData{ImpIDs: []string{"imp-1"}},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody},
	)

	if len(responseErrs) != 1 {
		t.Fatalf("expected one response error, got %v", responseErrs)
	}
	if bidResponse == nil || len(bidResponse.Bids) != 0 {
		t.Fatalf("expected the out-of-group bid to be dropped, got %#v", bidResponse)
	}
}

func TestMakeBidsPreservesOpenRTBResponseMetadata(t *testing.T) {
	bidder, err := Builder(
		openrtb_ext.BidderPanxo,
		config.Adapter{Endpoint: "https://panxo-sys.com/openrtb/2.6/bid"},
		config.Server{},
	)
	if err != nil {
		t.Fatalf("Builder returned unexpected error: %v", err)
	}

	responseBody := []byte(`{
		"id":"request-1",
		"cur":"USD",
		"seatbid":[{"bid":[{
			"id":"bid-1",
			"impid":"imp-1",
			"price":2.5,
			"mtype":1,
			"ext":{
				"prebid":{"meta":{"networkId":85,"networkName":"Index Exchange","futureField":{"enabled":true}}},
				"dsa":{"behalf":"buyer","paid":"buyer","adrender":1},
				"futureExtension":{"version":2}
			}
		}]}]
	}`)
	bidResponse, responseErrs := bidder.MakeBids(
		&openrtb2.BidRequest{ID: "request-1", Imp: []openrtb2.Imp{{ID: "imp-1", Banner: &openrtb2.Banner{}}}},
		&adapters.RequestData{ImpIDs: []string{"imp-1"}},
		&adapters.ResponseData{StatusCode: http.StatusOK, Body: responseBody},
	)

	if len(responseErrs) != 0 {
		t.Fatalf("unexpected response errors: %v", responseErrs)
	}
	if bidResponse == nil || len(bidResponse.Bids) != 1 {
		t.Fatalf("expected one bid, got %#v", bidResponse)
	}

	var bidExt map[string]any
	if err := json.Unmarshal(bidResponse.Bids[0].Bid.Ext, &bidExt); err != nil {
		t.Fatalf("could not decode preserved bid extension: %v", err)
	}
	prebidExt, ok := bidExt["prebid"].(map[string]any)
	if !ok {
		t.Fatalf("missing preserved ext.prebid metadata: %#v", bidExt)
	}
	meta, ok := prebidExt["meta"].(map[string]any)
	if !ok || meta["networkName"] != "Index Exchange" {
		t.Errorf("unexpected preserved metadata: %#v", prebidExt)
	}
	if _, ok := bidExt["dsa"].(map[string]any); !ok {
		t.Errorf("missing preserved DSA extension: %#v", bidExt)
	}
	if _, ok := bidExt["futureExtension"].(map[string]any); !ok {
		t.Errorf("missing future response extension: %#v", bidExt)
	}
}
