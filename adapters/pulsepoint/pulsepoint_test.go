package pulsepoint

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"

	"bytes"
	"context"
	"fmt"
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http/httptest"
	"time"
)

func TestOpenRTBRequest(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-12345",
		Imp: []openrtb.Imp{{
			ID: "banner-1",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					{W: 300, H: 250},
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"cp": 1234,
				"ct": 1001
			}}`),
		}, {
			ID: "test-imp-video-id",
			Video: &openrtb.Video{
				W:           640,
				H:           360,
				MIMEs:       []string{"video/mp4"},
				MinDuration: 15,
				MaxDuration: 30,
			},
			Ext: json.RawMessage(`{"bidder": {
				"cp": 1234,
				"ct": 2001
			}}`),
		}},
		Site: &openrtb.Site{
			Publisher: &openrtb.Publisher{
				Name: "publisher.com",
			},
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)
	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)

	httpReq := reqs[0]
	assert.Equal(t, "POST", httpReq.Method, "Expected a POST message. Got %s", httpReq.Method)

	var ortbRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReq.Body, &ortbRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", ortbRequest)
	}

	assert.Equal(t, request.ID, ortbRequest.ID, "Bad Request ID. Expected %s, Got %s", request.ID, ortbRequest.ID)
	assert.Equal(t, 2, len(ortbRequest.Imp), "Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(ortbRequest.Imp))
	assert.Equal(t, request.Imp[0].ID, ortbRequest.Imp[0].ID, "Bad Impression ID. Expected %s, Got %s", request.Imp[0].ID, ortbRequest.Imp[0].ID)
	assert.Equal(t, request.Imp[1].ID, ortbRequest.Imp[1].ID, "Bad Impression ID. Expected %s, Got %s", request.Imp[1].ID, ortbRequest.Imp[1].ID)
	assert.Equal(t, "1001", ortbRequest.Imp[0].TagID, "Bad Tag ID. Expected 1001, Got %s", ortbRequest.Imp[0].TagID)
	assert.Equal(t, "2001", ortbRequest.Imp[1].TagID, "Bad Tag ID. Expected 2001, Got %s", ortbRequest.Imp[1].TagID)
	assert.Equal(t, "1234", ortbRequest.Site.Publisher.ID, "Bad Publisher ID. Expected 1234, Got %s", ortbRequest.Site.Publisher.ID)
}

func TestOpenRTBRequestNoPubProvided(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-12345",
		Imp: []openrtb.Imp{{
			ID: "banner-1",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					{W: 300, H: 250},
				},
			},
			Ext: json.RawMessage(`{"bidder": {
				"cp": 1234,
				"ct": 1001
			}}`),
		},
		},
		App: &openrtb.App{
			ID: "com.pulsepoint.app",
		},
	}

	reqs, errs := bidder.MakeRequests(request, &adapters.ExtraRequestInfo{})

	assert.Empty(t, errs, "Got unexpected errors while building HTTP requests: %v", errs)
	assert.Equal(t, 1, len(reqs), "Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)

	httpReq := reqs[0]
	assert.Equal(t, "POST", httpReq.Method, "Expected a POST message. Got %s", httpReq.Method)

	var ortbRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReq.Body, &ortbRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", ortbRequest)
	}

	assert.Equal(t, request.ID, ortbRequest.ID, "Bad Request ID. Expected %s, Got %s", request.ID, ortbRequest.ID)
	assert.Equal(t, 1, len(ortbRequest.Imp), "Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(ortbRequest.Imp))
	assert.Equal(t, "1234", ortbRequest.App.Publisher.ID, "Bad Publisher ID. Expected 1234, Got %s", ortbRequest.App.Publisher.ID)
}

func TestMakeBids(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-1000",
		Imp: []openrtb.Imp{{
			ID: "imp-123",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{
					{W: 300, H: 250},
				},
			},
		},
		},
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"request-1000","seatbid":[{"bid":[{"id":"1234567890","impid":"imp-123","price":2,"crid":"4122982","adm":"some ad","h":50,"w":320}]}]}`),
	}

	openrtbResponse, errs := bidder.MakeBids(request, nil, httpResp)
	assert.NotNil(t, openrtbResponse, "Expected not empty response")
	assert.Equal(t, 1, len(openrtbResponse.Bids), "Expected 1 bid. Got %d", len(openrtbResponse.Bids))
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
	assert.Equal(t, openrtb_ext.BidTypeBanner, openrtbResponse.Bids[0].BidType, "Expected bid type %s. Got %s", openrtb_ext.BidTypeBanner, openrtbResponse.Bids[0].BidType)
}

func TestMakeBidsVideo(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-1001",
		Imp: []openrtb.Imp{{
			ID: "imp-234",
			Video: &openrtb.Video{
				W: 640,
				H: 360,
			},
		},
		},
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"request-1001","seatbid":[{"bid":[{"id":"1234567890","impid":"imp-234","price":2,"crid":"4122982","adm":"<vast></vast>"}]}]}`),
	}

	openrtbResponse, errs := bidder.MakeBids(request, nil, httpResp)
	assert.NotNil(t, openrtbResponse, "Expected not empty response")
	assert.Equal(t, 1, len(openrtbResponse.Bids), "Expected 1 bid. Got %d", len(openrtbResponse.Bids))
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
	assert.Equal(t, openrtb_ext.BidTypeVideo, openrtbResponse.Bids[0].BidType, "Expected bid type %s. Got %s", openrtb_ext.BidTypeVideo, openrtbResponse.Bids[0].BidType)
}

func TestUnknownImpId(t *testing.T) {
	bidder := new(PulsePointAdapter)

	request := &openrtb.BidRequest{
		ID: "request-1001",
		Imp: []openrtb.Imp{{
			ID: "imp-234",
			Video: &openrtb.Video{
				W: 640,
				H: 360,
			},
		},
		},
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"id":"request-1001","seatbid":[{"bid":[{"id":"1234567890","impid":"imp-345","price":2,"crid":"4122982","adm":"<vast></vast>"}]}]}`),
	}

	openrtbResponse, errs := bidder.MakeBids(request, nil, httpResp)
	assert.NotNil(t, openrtbResponse, "Expected not empty response")
	assert.Equal(t, 0, len(openrtbResponse.Bids), "Expected 1 bid. Got %d", len(openrtbResponse.Bids))
	assert.Empty(t, errs, "Expected 0 errors. Got %d", len(errs))
}

/////////////////////////////////
// Legacy implementation: Start
/////////////////////////////////

/**
 * Verify adapter names are setup correctly.
 */
func TestPulsePointAdapterNames(t *testing.T) {
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	adapterstest.VerifyStringValue(adapter.Name(), "pulsepoint", t)
}

/**
 * Test required parameters not sent
 */
func TestPulsePointRequiredBidParameters(t *testing.T) {
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	ctx := context.TODO()
	req := SampleRequest(1, t)
	bidder := req.Bidders[0]
	// remove "ct" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"cp\": 2001, \"cf\": \"728X90\"}")
	_, errTag := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errTag.Error(), "Missing TagId param ct", t)
	// remove "cp" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cf\": \"728X90\"}")
	_, errPub := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errPub.Error(), "Missing PublisherId param cp", t)
	// remove "cf" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"cp\": 2001, \"ct\": 1001}")
	_, errSize := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errSize.Error(), "Missing AdSize param cf", t)
	// invalid width parameter value for cf
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cp\": 2001, \"cf\": \"aXb\"}")
	_, errWidth := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errWidth.Error(), "Invalid Width param a", t)
	// invalid parameter values for cf
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cp\": 2001, \"cf\": \"12Xb\"}")
	_, errHeight := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errHeight.Error(), "Invalid Height param b", t)
	// invalid parameter values for cf
	bidder.AdUnits[0].Params = json.RawMessage("{\"ct\": 1001, \"cp\": 2001, \"cf\": \"12-20\"}")
	_, errAdSizeValue := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errAdSizeValue.Error(), "Invalid AdSize param 12-20", t)
}

/**
 * Verify the openrtb request sent to Pulsepoint endpoint.
 * Ensure the ct, cp, cf params are transformed and sent alright.
 */
func TestPulsePointOpenRTBRequest(t *testing.T) {
	service := CreateService(adapterstest.BidOnTags(""))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	adapter.Call(ctx, req, bidder)
	fmt.Println(service.LastBidRequest)
	adapterstest.VerifyIntValue(len(service.LastBidRequest.Imp), 1, t)
	adapterstest.VerifyStringValue(service.LastBidRequest.Imp[0].TagID, "1001", t)
	adapterstest.VerifyStringValue(service.LastBidRequest.Site.Publisher.ID, "2001", t)
	adapterstest.VerifyBannerSize(service.LastBidRequest.Imp[0].Banner, 728, 90, t)
}

/**
 * Verify bidding behavior.
 */
func TestPulsePointBiddingBehavior(t *testing.T) {
	// setup server endpoint to return bid.
	server := CreateService(adapterstest.BidOnTags("1001")).Server
	ctx := context.TODO()
	req := SampleRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// number of bids should be 1
	adapterstest.VerifyIntValue(len(bids), 1, t)
	adapterstest.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapterstest.VerifyStringValue(bids[0].BidderCode, "pulsepoint", t)
	adapterstest.VerifyStringValue(bids[0].Adm, "<div>This is an Ad</div>", t)
	adapterstest.VerifyStringValue(bids[0].Creative_id, "Cr-234", t)
	adapterstest.VerifyIntValue(int(bids[0].Width), 728, t)
	adapterstest.VerifyIntValue(int(bids[0].Height), 90, t)
	adapterstest.VerifyIntValue(int(bids[0].Price*100), 210, t)
	adapterstest.VerifyStringValue(bids[0].CreativeMediaType, string(openrtb_ext.BidTypeBanner), t)
}

/**
 * Verify bidding behavior on multiple impressions, some impressions make a bid
 */
func TestPulsePointMultiImpPartialBidding(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapterstest.BidOnTags("1001"))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapterstest.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapterstest.VerifyIntValue(len(bids), 1, t)
	adapterstest.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
}

/**
 * Verify bidding behavior on multiple impressions, all impressions passed back.
 */
func TestPulsePointMultiImpPassback(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapterstest.BidOnTags(""))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapterstest.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapterstest.VerifyIntValue(len(bids), 0, t)
}

/**
 * Verify bidding behavior on multiple impressions, all impressions passed back.
 */
func TestPulsePointMultiImpAllBid(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapterstest.BidOnTags("1001,1002"))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapterstest.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapterstest.VerifyIntValue(len(bids), 2, t)
	adapterstest.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapterstest.VerifyStringValue(bids[1].AdUnitCode, "div-adunit-2", t)
}

/**
 * Verify bidding behavior on mobile app requests
 */
func TestMobileAppRequest(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateService(adapterstest.BidOnTags("1001"))
	server := service.Server
	ctx := context.TODO()
	req := SampleRequest(1, t)
	req.App = &openrtb.App{
		ID:   "com.facebook.katana",
		Name: "facebook",
	}
	bidder := req.Bidders[0]
	adapter := NewPulsePointLegacyAdapter(adapters.DefaultHTTPAdapterConfig, server.URL)
	bids, _ := adapter.Call(ctx, req, bidder)
	// one mobile app impression sent.
	// verify appropriate fields are sent to pulsepoint endpoint.
	adapterstest.VerifyIntValue(len(service.LastBidRequest.Imp), 1, t)
	adapterstest.VerifyStringValue(service.LastBidRequest.App.ID, "com.facebook.katana", t)
	adapterstest.VerifyIntValue(len(bids), 1, t)
	adapterstest.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
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
	httpReq := httptest.NewRequest("POST", CreateService(adapterstest.BidOnTags("")).Server.URL, body)
	httpReq.Header.Add("Referer", "http://news.pub/topnews")
	pc := usersync.ParsePBSCookieFromRequest(httpReq, &config.HostCookie{})
	pc.TrySync("pulsepoint", "pulsepointUser123")
	fakewriter := httptest.NewRecorder()

	pc.SetCookieOnResponse(fakewriter, false, &config.HostCookie{Domain: ""}, 90*24*time.Hour)
	httpReq.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))
	// parse the http request
	cacheClient, _ := dummycache.New()
	hcs := config.HostCookie{}

	parsedReq, err := pbs.ParsePBSRequest(httpReq, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &hcs)
	if err != nil {
		t.Fatalf("Error when parsing request: %v", err)
	}
	return parsedReq
}

/**
 * Represents a mock ORTB endpoint of PulsePoint. Would return a bid
 * for TagId 1001 and passback for 1002 as the default behavior.
 */
func CreateService(tagsToBid map[string]bool) adapterstest.OrtbMockService {
	service := adapterstest.OrtbMockService{}
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
				bids = append(bids, adapterstest.SampleBid(imp.Banner.W, imp.Banner.H, imp.ID, i+1))
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

/////////////////////////////////
// Legacy implementation: End
/////////////////////////////////
