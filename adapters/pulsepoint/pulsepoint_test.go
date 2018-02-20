package pulsepoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
)

/**
 * Verify adapter names are setup correctly.
 */
func TestPulsePointAdapterNames(t *testing.T) {
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	adapters.VerifyStringValue(adapter.Name(), "pulsepoint", t)
	adapters.VerifyStringValue(adapter.FamilyName(), "pulsepoint", t)
}

/**
 * Test required parameters not sent
 */
func TestPulsePointRequiredBidParameters(t *testing.T) {
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	ctx := context.TODO()
	req := SampleRequest(1, t)
	bidder := req.Bidders[0]
	// remove "ct" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"cp\": 2001, \"cf\": \"728X90\"}")
	_, errTag := adapter.Call(ctx, req, bidder)
	adapters.VerifyStringValue(errTag.Error(), "Missing TagId param ct", t)
	// remove "cp" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cf\": \"728X90\"}")
	_, errPub := adapter.Call(ctx, req, bidder)
	adapters.VerifyStringValue(errPub.Error(), "Missing PublisherId param cp", t)
	// remove "cf" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"cp\": 2001, \"ct\": 1001}")
	_, errSize := adapter.Call(ctx, req, bidder)
	adapters.VerifyStringValue(errSize.Error(), "Missing AdSize param cf", t)
	// invalid width parameter value for cf
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cp\": 2001, \"cf\": \"aXb\"}")
	_, errWidth := adapter.Call(ctx, req, bidder)
	adapters.VerifyStringValue(errWidth.Error(), "Invalid Width param a", t)
	// invalid parameter values for cf
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cp\": 2001, \"cf\": \"12Xb\"}")
	_, errHeight := adapter.Call(ctx, req, bidder)
	adapters.VerifyStringValue(errHeight.Error(), "Invalid Height param b", t)
	// invalid parameter values for cf
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cp\": 2001, \"cf\": \"12-20\"}")
	_, errAdSizeValue := adapter.Call(ctx, req, bidder)
	adapters.VerifyStringValue(errAdSizeValue.Error(), "Invalid AdSize param 12-20", t)
}

/**
 * Verify the openrtb request sent to Pulsepoint endpoint.
 * Ensure the ct, cp, cf params are transformed and sent alright.
 */
func TestPulsePointOpenRTBRequest(t *testing.T) {
	service := CreateService(adapters.BidOnTags(""))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	adapter.Call(ctx, req, bidder)
	fmt.Println(service.LastBidRequest)
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 1, t)
	adapters.VerifyStringValue(service.LastBidRequest.Imp[0].TagID, "1001", t)
	adapters.VerifyStringValue(service.LastBidRequest.Site.Publisher.ID, "2001", t)
	adapters.VerifyBannerSize(service.LastBidRequest.Imp[0].Banner, 728, 90, t)
}

/**
 * Verify bidding behavior.
 */
func TestPulsePointBiddingBehavior(t *testing.T) {
	// setup server endpoint to return bid.
	server := CreateService(adapters.BidOnTags("1001")).Server
	ctx := context.TODO()
	req := SampleRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// number of bids should be 1
	adapters.VerifyIntValue(len(bids), 1, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapters.VerifyStringValue(bids[0].BidderCode, "pulsepoint", t)
	adapters.VerifyStringValue(bids[0].Adm, "<div>This is an Ad</div>", t)
	adapters.VerifyStringValue(bids[0].Creative_id, "Cr-234", t)
	adapters.VerifyIntValue(int(bids[0].Width), 728, t)
	adapters.VerifyIntValue(int(bids[0].Height), 90, t)
	adapters.VerifyIntValue(int(bids[0].Price*100), 210, t)
}

/**
 * Verify bidding behavior on multiple impressions, some impressions make a bid
 */
func TestPulsePointMultiImpPartialBidding(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapters.BidOnTags("1001"))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapters.VerifyIntValue(len(bids), 1, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
}

/**
 * Verify bidding behavior on multiple impressions, all impressions passed back.
 */
func TestPulsePointMultiImpPassback(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapters.BidOnTags(""))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapters.VerifyIntValue(len(bids), 0, t)
}

/**
 * Verify bidding behavior on multiple impressions, all impressions passed back.
 */
func TestPulsePointMultiImpAllBid(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapters.BidOnTags("1001,1002"))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapters.VerifyIntValue(len(bids), 2, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapters.VerifyStringValue(bids[1].AdUnitCode, "div-adunit-2", t)
}

/**
 * Verify bidding behavior on mobile app requests
 */
func TestMobileAppRequest(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapters.BidOnTags("1001"))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(1, t)
	req.App = &openrtb.App{
		ID:   "com.facebook.katana",
		Name: "facebook",
	}
	bidder := req.Bidders[0]
	adapter := NewPulsePointAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// one mobile app impression sent.
	// verify appropriate fields are sent to pulsepoint endpoint.
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 1, t)
	adapters.VerifyStringValue(service.LastBidRequest.App.ID, "com.facebook.katana", t)
	adapters.VerifyIntValue(len(bids), 1, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
}

/**
 * Produces a sample PBSRequest, for the impressions given.
 */
func SampleRequest(numberOfImpressions int, t *testing.T) *pbs.PBSRequest {
	// create a request object
	req := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 2),
	}
	req.AccountID = "1"
	tagId := 1001
	for i := 0; i < numberOfImpressions; i++ {
		req.AdUnits[i] = pbs.AdUnit{
			Code: fmt.Sprintf("div-adunit-%d", i+1),
			Sizes: []openrtb.Format{
				{
					W: 10,
					H: 12,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "pulsepoint",
					BidID:      fmt.Sprintf("Bid-%d", i+1),
					Params:     json.RawMessage(fmt.Sprintf("{\"ct\": %d, \"cp\": 2001, \"cf\": \"728X90\"}", tagId+i)),
				},
			},
		}
	}
	// serialize the request to json
	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	if err != nil {
		t.Fatalf("Error when serializing request")
	}
	// setup a http request
	httpReq := httptest.NewRequest("POST", CreateService(adapters.BidOnTags("")).Server.URL, body)
	httpReq.Header.Add("Referer", "http://news.pub/topnews")
	pc := pbs.ParsePBSCookieFromRequest(httpReq, &config.Cookie{})
	pc.TrySync("pulsepoint", "pulsepointUser123")
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "", 90*24*time.Hour)
	httpReq.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))
	// parse the http request
	cacheClient, _ := dummycache.New()
	hcs := pbs.HostCookieSettings{}

	parsedReq, err := pbs.ParsePBSRequest(httpReq, cacheClient, &hcs)
	if err != nil {
		t.Fatalf("Error when parsing request: %v", err)
	}
	return parsedReq
}

/**
 * Represents a mock ORTB endpoint of PulsePoint. Would return a bid
 * for TagId 1001 and passback for 1002 as the default behavior.
 */
func CreateService(tagsToBid map[string]bool) adapters.OrtbMockService {
	service := adapters.OrtbMockService{}
	var lastBidRequest openrtb.BidRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		var breq openrtb.BidRequest
		err = json.Unmarshal(body, &breq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		lastBidRequest = breq
		var bids []openrtb.Bid
		for i, imp := range breq.Imp {
			if tagsToBid[imp.TagID] {
				bids = append(bids, adapters.SampleBid(imp.Banner.W, imp.Banner.H, imp.ID, i+1))
			}
		}
		// no bids were produced, pulsepoint service returns 204
		if len(bids) == 0 {
			w.WriteHeader(204)
		} else {
			// serialize the bids to openrtb.BidResponse
			js, _ := json.Marshal(openrtb.BidResponse{
				SeatBid: []openrtb.SeatBid{
					{
						Bid: bids,
					},
				},
			})
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}
	}))
	service.Server = server
	service.LastBidRequest = &lastBidRequest
	return service
}
