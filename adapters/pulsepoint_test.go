package adapters

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
	"net/http"
	"net/http/httptest"
	"testing"
)

/**
 * Verify adapter names are setup correctly.
 */
func TestPulsePointAdapterNames(t *testing.T) {
	adapter := NewPulsePointAdapter(DefaultHTTPAdapterConfig, "http://localhost/bid", "http://localhost")
	VerifyStringValue(adapter.Name(), "PulsePoint", t)
	VerifyStringValue(adapter.FamilyName(), "pulsepoint", t)
}

/**
 * Verifies the user sync parameters.
 */
func TestPulsePointUserSyncInfo(t *testing.T) {
	adapter := NewPulsePointAdapter(DefaultHTTPAdapterConfig, "http://localhost/bid", "http://localhost")
	VerifyStringValue(adapter.GetUsersyncInfo().Type, "redirect", t)
	VerifyStringValue(adapter.GetUsersyncInfo().URL, "https://bh.contextweb.com/rtset?pid=561205&ev=1&rurl=http%3A%2F%2Flocalhost%2Fsetuid%3Fbidder%3Dpulsepoint%26uid%3D%25%25VGUID%25%25", t)
}

/**
 * Test required parameters not sent
 */
func TestPulsePointRequiredBidParameters(t *testing.T) {
	adapter := NewPulsePointAdapter(DefaultHTTPAdapterConfig, "http://localhost/bid", "http://localhost")
	ctx := context.TODO()
	req := SampleRequest()
	bidder := SampleBidder()
	// remove "ct" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"cp\": 2001, \"cf\": \"728X90\"}")
	_, errTag := adapter.Call(ctx, &req, &bidder)
	VerifyStringValue(errTag.Error(), "Missing TagId param ct", t)
	// remove "cp" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cf\": \"728X90\"}")
	_, errPub := adapter.Call(ctx, &req, &bidder)
	VerifyStringValue(errPub.Error(), "Missing PublisherId param cp", t)
	// remove "cf" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"cp\": 2001, \"ct\": 1001}")
	_, errSize := adapter.Call(ctx, &req, &bidder)
	VerifyStringValue(errSize.Error(), "Missing AdSize param cf", t)
}

/**
 * Verify the openrtb request sent to Pulsepoint endpoint.
 * Ensure the ct, cp, cf params are transformed and sent alright.
 */
func TestPulsePointOpenRTBRequest(t *testing.T) {
	var ortbRequest openrtb.BidRequest
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			// Send 204 (no-bid)
			decoder := json.NewDecoder(req.Body)
			decoder.Decode(&ortbRequest)
			defer req.Body.Close()
			w.WriteHeader(204)
		}),
	)
	defer server.Close()
	ctx := context.TODO()
	req := SampleRequest()
	bidder := SampleBidder()
	adapter := NewPulsePointAdapter(DefaultHTTPAdapterConfig, server.URL, "http://localhost")
	adapter.Call(ctx, &req, &bidder)
	VerifyStringValue(ortbRequest.Imp[0].TagID, "1001", t)
	VerifyStringValue(ortbRequest.Site.Publisher.ID, "2001", t)
	VerifyIntValue(int(ortbRequest.Imp[0].Banner.W), 728, t)
	VerifyIntValue(int(ortbRequest.Imp[0].Banner.H), 90, t)
}

/**
 * Verify bidding behavior.
 */
func TestPulsePointBiddingBehavior(t *testing.T) {
	// setup server endpoint to return bid.
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			resp := SampleBidResponse()
			js, _ := json.Marshal(resp)
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}),
	)
	defer server.Close()
	ctx := context.TODO()
	req := SampleRequest()
	bidder := SampleBidder()
	adapter := NewPulsePointAdapter(DefaultHTTPAdapterConfig, server.URL, "http://localhost")
	bids, _ := adapter.Call(ctx, &req, &bidder)
	// number of bids should be 1
	VerifyIntValue(len(bids), 1, t)
	VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	VerifyStringValue(bids[0].BidderCode, "pulsepoint", t)
	VerifyStringValue(bids[0].Adm, "<div>This is an Ad</div>", t)
	VerifyStringValue(bids[0].Creative_id, "Cr-234", t)
	VerifyIntValue(int(bids[0].Width), 728, t)
	VerifyIntValue(int(bids[0].Height), 90, t)
	VerifyIntValue(int(bids[0].Price*100), 210, t)
}

func SampleBidder() pbs.PBSBidder {
	bidder := pbs.PBSBidder{
		BidderCode: "pulsepoint",
		AdUnitCode: "div-adunit-1",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:  "div-adunit-1",
				BidID: "bid-123",
				Sizes: []openrtb.Format{
					{
						W: 120,
						H: 600,
					},
				},
				Params: json.RawMessage("{\"ct\": 1001, \"cp\": 2001, \"cf\": \"728X90\"}"),
			},
		},
	}
	return bidder
}

func SampleBidResponse() openrtb.BidResponse {
	return openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{
			{
				Bid: []openrtb.Bid{
					{
						ID:    "Bid-123",
						ImpID: "div-adunit-1",
						Price: 2.1,
						AdM:   "<div>This is an Ad</div>",
						CrID:  "Cr-234",
						W:     728,
						H:     90,
					},
				},
			},
		},
	}
}

func SampleRequest() pbs.PBSRequest {
	req := pbs.PBSRequest{
		AccountID: "1",
		AdUnits: []pbs.AdUnit{
			{
				Code: "div-adunit-1",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Bids: []pbs.Bids{
					{
						BidderCode: "pulsepoint",
						BidID:      "Bid-123",
						Params:     json.RawMessage("{\"cp\": 1001, \"cp\": 2001, \"cf\": \"728X90\"}"),
					},
				},
			},
		},
	}
	return req
}

/**
 * Helper function to assert string equals.
 */
func VerifyStringValue(value string, expected string, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%s expected, got %s", expected, value))
	}
}

/**
 * Helper function to assert Int equals.
 */
func VerifyIntValue(value int, expected int, t *testing.T) {
	if value != expected {
		t.Fatalf(fmt.Sprintf("%d expected, got %d", expected, value))
	}
}
