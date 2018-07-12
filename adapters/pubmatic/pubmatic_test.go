package pubmatic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"
)

func CompareStringValue(val1 string, val2 string, t *testing.T) {
	if val1 != val2 {
		t.Fatalf(fmt.Sprintf("Expected = %s , Actual = %s", val2, val1))
	}
}

func DummyPubMaticServer(w http.ResponseWriter, r *http.Request) {
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

	resp := openrtb.BidResponse{
		ID:    breq.ID,
		BidID: "bidResponse_ID",
		Cur:   "USD",
		SeatBid: []openrtb.SeatBid{
			{
				Seat: "pubmatic",
				Bid:  make([]openrtb.Bid, 0),
			},
		},
	}
	rand.Seed(int64(time.Now().UnixNano()))
	var bids []openrtb.Bid

	for i, imp := range breq.Imp {
		bids = append(bids, openrtb.Bid{
			ID:     fmt.Sprintf("SeatID_%d", i),
			ImpID:  imp.ID,
			Price:  float64(int(rand.Float64()*1000)) / 100,
			AdID:   fmt.Sprintf("adID-%d", i),
			AdM:    "AdContent",
			CrID:   fmt.Sprintf("creative-%d", i),
			W:      *imp.Banner.W,
			H:      *imp.Banner.H,
			DealID: fmt.Sprintf("DealID_%d", i),
		})
	}
	resp.SeatBid[0].Bid = bids

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestPubmaticInvalidCall(t *testing.T) {

	an := NewPubmaticAdapter(adapters.DefaultHTTPAdapterConfig, "blah")

	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestPubmaticTimeout(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-time.After(2 * time.Millisecond)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 120,
						H: 240,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240\"}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("No timeout received for timed out request: %v", err)
	}
}

func TestPubmaticInvalidJson(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Blah")
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)
	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 120,
						H: 240,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240\"}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestPubmaticInvalidStatusCode(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send 404
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)
	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 120,
						H: 240,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240\"}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestPubmaticInvalidInputParameters(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)
	ctx := context.Background()

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				BidID:      "bidid",
				Sizes: []openrtb.Format{
					{
						W: 120,
						H: 240,
					},
				},
			},
		},
	}

	pbReq.IsDebug = true
	inValidPubmaticParams := []json.RawMessage{
		// Invalid Request JSON
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240\""),
		// Missing adSlot in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\"}"),
		// Missing publisher ID
		json.RawMessage("{\"adSlot\": \"slot@120x240\"}"),
		// Missing slot name  in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"@120x240\"}"),
		// Invalid adSize in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120-240\"}"),
		// Missing impression width and height in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@\"}"),
		// Missing height  in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120\"}"),
		// Missing width  in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@x120\"}"),
		// Incorrect width param  in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@valx120\"}"),
		// Incorrect height param  in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120xval\"}"),
		// Empty slot name in AdUnits.Params,
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \" @120x240\"}"),
		// Empty width in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@ x240\"}"),
		// Empty height in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x \"}"),
		// Empty height in AdUnits.Params
		json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \" @120x \"}"),
		// Invalid Keywords
		json.RawMessage(`{"publisherId": "640",	"adSlot": "slot1@336x280","keywords":{"pmZoneId":1},"wrapper":{"version":2,"profile":595}}`),
		// Invalid Wrapper ext
		json.RawMessage(`{"publisherId": "640",	"adSlot": "slot1@336x280","keywords":{"pmZoneId":"Zone1,Zone2"},"wrapper":{"version":"2","profile":595}}`),
	}

	for _, param := range inValidPubmaticParams {
		pbBidder.AdUnits[0].Params = param
		_, err := an.Call(ctx, &pbReq, &pbBidder)
		if err == nil {
			t.Fatalf("Should get errors for params = %v", string(param))
		}
	}

}

func TestPubmaticBasicResponse_MandatoryParams(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)
	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				BidID:      "bidid",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 336,
						H: 280,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"640\", \"adSlot\": \"slot1@336x280\"}"),
			},
		},
	}
	pbReq.IsDebug = true
	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 1 {
		t.Fatalf("Should have received one bid")
	}
}

func TestPubmaticBasicResponse_AllParams(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)
	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				BidID:      "bidid",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 336,
						H: 280,
					},
				},
				Params: json.RawMessage(`{"publisherId": "640",
							"adSlot": "slot1@336x280",
							"keywords":{
									"pmZoneId": "Zone1,Zone2"
									},
							"wrapper":
									{"version":2,
									"profile":595}
									}`),
			},
		},
	}
	pbReq.IsDebug = true
	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 1 {
		t.Fatalf("Should have received one bid")
	}
}

func TestPubmaticMultiImpressionResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)

	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode1",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				BidID:      "bidid",
				Sizes: []openrtb.Format{
					{
						W: 336,
						H: 280,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"640\", \"adSlot\": \"slot1@336x280\"}"),
			},
			{
				Code:       "unitCode1",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				BidID:      "bidid",
				Sizes: []openrtb.Format{
					{
						W: 800,
						H: 200,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"640\", \"adSlot\": \"slot1@800x200\"}"),
			},
		},
	}
	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 2 {
		t.Fatalf("Should have received two bids")
	}
}

func TestPubmaticMultiAdUnitResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)

	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode1",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				BidID:      "bidid",
				Sizes: []openrtb.Format{
					{
						W: 336,
						H: 280,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"640\", \"adSlot\": \"slot1@336x280\"}"),
			},
			{
				Code:       "unitCode2",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				BidID:      "bidid",
				Sizes: []openrtb.Format{
					{
						W: 800,
						H: 200,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"640\", \"adSlot\": \"slot1@800x200\"}"),
			},
		},
	}
	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 2 {
		t.Fatalf("Should have received one bid")
	}

}

func TestPubmaticMobileResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)

	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				BidID:      "bidid",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 336,
						H: 280,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"640\", \"adSlot\": \"slot1@336x280\"}"),
			},
		},
	}

	pbReq.App = &openrtb.App{
		ID:   "com.test",
		Name: "testApp",
	}

	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 1 {
		t.Fatalf("Should have received one bid")
	}
}
func TestPubmaticInvalidLookupBidIDParameter(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)

	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 120,
						H: 240,
					},
				},
			},
		},
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240\"}")
	_, err := an.Call(ctx, &pbReq, &pbBidder)

	CompareStringValue(err.Error(), "Unknown ad unit code 'unitCode'", t)
}

func TestPubmaticAdSlotParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL)

	ctx := context.Background()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:       "unitCode",
				BidID:      "bidid",
				MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
				Sizes: []openrtb.Format{
					{
						W: 120,
						H: 240,
					},
				},
			},
		},
	}
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \" slot@120x240\"}")
	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot @120x240\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240 \"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@ 120x240\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@220 x240\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x 240\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240:1\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x 240:1\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240 :1\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240: 1\"}")
	bids, err = an.Call(ctx, &pbReq, &pbBidder)
	if err != nil && len(bids) != 1 {
		t.Fatalf("Should not return err")
	}
}

func TestPubmaticSampleRequest(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	pbReq := pbs.PBSRequest{
		AdUnits: make([]pbs.AdUnit, 1),
	}
	pbReq.AdUnits[0] = pbs.AdUnit{
		Code: "adUnit_1",
		Sizes: []openrtb.Format{
			{
				W: 100,
				H: 120,
			},
		},
		Bids: []pbs.Bids{
			{
				BidderCode: "pubmatic",
				BidID:      "BidID",
				Params:     json.RawMessage("{\"publisherId\": \"640\", \"adSlot\": \"slot1@100x120\"}"),
			},
		},
	}

	pbReq.IsDebug = true

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(pbReq)
	if err != nil {
		t.Fatalf("Error when serializing request")
	}

	httpReq := httptest.NewRequest("POST", server.URL, body)
	httpReq.Header.Add("Referer", "http://test.com/sports")
	pc := usersync.ParsePBSCookieFromRequest(httpReq, &config.HostCookie{})
	pc.TrySync("pubmatic", "12345")
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "", 90*24*time.Hour)
	httpReq.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()
	hcs := config.HostCookie{}

	_, err = pbs.ParsePBSRequest(httpReq, &config.AuctionTimeouts{
		Default: 2000,
		Max:     2000,
	}, cacheClient, &hcs)
	if err != nil {
		t.Fatalf("Error when parsing request: %v", err)
	}
}

func TestOpenRTBBidRequest(t *testing.T) {
	bidder := new(PubmaticAdapter)

	request := &openrtb.BidRequest{
		ID: "12345",
		Imp: []openrtb.Imp{{
			ID: "234",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
								"adSlot": "AdTag_Div1@300x250",
								"publisherId": "1234",
								"keywords":{
											"pmZoneID": "Zone1,Zone2",
											"preference": "sports,movies"
											},
								"wrapper":{"version":1,"profile":5123}
							}}`),
		}, {
			ID: "456",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 200,
					H: 350,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
				"adSlot": "AdTag_Div2@200x350",
				"publisherId": "1234",
				"keywords":{
							"pmZoneID": "Zone3,Zone4",
							"preference": "movies"
							}
			}}`),
		}},
		Device: &openrtb.Device{
			UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36",
		},
		User: &openrtb.User{
			ID: "testID",
		},
		Site: &openrtb.Site{
			ID: "siteID",
			Publisher: &openrtb.Publisher{
				ID: "1234",
			},
		},
	}

	reqs, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Fatalf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
	}

	httpReq := reqs[0]
	if httpReq.Method != "POST" {
		t.Errorf("Expected a POST message. Got %s", httpReq.Method)
	}

	var ortbRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReq.Body, &ortbRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", ortbRequest)
	}

	if ortbRequest.ID != request.ID {
		t.Errorf("Bad Request ID. Expected %s, Got %s", request.ID, ortbRequest.ID)
	}
	if len(ortbRequest.Imp) != len(request.Imp) {
		t.Fatalf("Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(ortbRequest.Imp))
	}

	if ortbRequest.Imp[0].ID == "234" {

		if ortbRequest.Imp[0].Banner.Format[0].W != 300 {
			t.Fatalf("Banner width does not match. Expected %d, Got %d", 300, ortbRequest.Imp[0].Banner.Format[0].W)
		}
		if ortbRequest.Imp[0].Banner.Format[0].H != 250 {
			t.Fatalf("Banner height does not match. Expected %d, Got %d", 250, ortbRequest.Imp[0].Banner.Format[0].H)
		}
		if ortbRequest.Imp[0].TagID != "AdTag_Div1" {
			t.Fatalf("Failed to Set TqagID. Expected %s, Got %s", "AdTag_Div1", ortbRequest.Imp[0].TagID)
		}

		if ortbRequest.Imp[0].Ext == nil {
			t.Fatalf("Failed to add imp.Ext into outgoing request.")
		}
	}
	if ortbRequest.Imp[1].ID == "456" {

		if ortbRequest.Imp[1].Banner.Format[0].W != 200 {
			t.Fatalf("Banner width does not match. Expected %d, Got %d", 200, ortbRequest.Imp[1].Banner.Format[0].W)
		}

		if ortbRequest.Imp[1].Banner.Format[0].H != 350 {
			t.Fatalf("Banner height does not match. Expected %d, Got %d", 350, ortbRequest.Imp[1].Banner.Format[0].H)
		}
		if ortbRequest.Imp[1].TagID != "AdTag_Div2" {
			t.Fatalf("Failed to Set TagID. Expected %s, Got %s", "AdTag_Div2", ortbRequest.Imp[1].TagID)
		}
		if ortbRequest.Imp[1].Ext == nil {
			t.Fatalf("Failed to add imp.Ext into outgoing request.")
		}
	}
	if ortbRequest.Site.Publisher.ID != "1234" {
		t.Fatalf("Failed to Publisher ID. Expected %s Actual %s", "1234", ortbRequest.Site.Publisher.ID)
	}

	if string(ortbRequest.Ext) != "{\"wrapper\":{\"version\":1,\"profile\":5123}}" {
		t.Fatalf("Failed to set  ortbRequest.Ext. Expected %s Actual %s ", "{\"wrapper\":{\"version\":8,\"profile\":593}}", string(ortbRequest.Ext))
	}
}

func TestOpenRTBBidRequest_MandatoryParams(t *testing.T) {
	bidder := new(PubmaticAdapter)

	request := &openrtb.BidRequest{
		ID: "12345",
		Imp: []openrtb.Imp{{
			ID: "234",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
								"adSlot": "AdTag_Div1@300x250",
								"publisherId": "1234"
							}}`),
		}},
		Device: &openrtb.Device{
			UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36",
		},
		User: &openrtb.User{
			ID: "testID",
		},
		Site: &openrtb.Site{
			ID: "siteID",
		},
	}

	reqs, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Fatalf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
	}

}

func TestOpenRTBBidRequest_App(t *testing.T) {
	bidder := new(PubmaticAdapter)

	request := &openrtb.BidRequest{
		ID: "12345",
		Imp: []openrtb.Imp{{
			ID: "234",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
								"adSlot": "AdTag_Div1@300x250",
								"publisherId": "1234",
								"keywords":{
											"pmZoneID": "Zone1,Zone2",
											"preference": "sports,movies"
											},
								"wrapper":{"version":1,"profile":5123}
							}}`),
		}},
		Device: &openrtb.Device{
			UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36",
		},
		User: &openrtb.User{
			ID: "testID",
		},
		App: &openrtb.App{
			ID: "appID",
		},
	}

	reqs, errs := bidder.MakeRequests(request)

	if len(errs) > 0 {
		t.Fatalf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 1 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 1)
	}

	httpReq := reqs[0]
	if httpReq.Method != "POST" {
		t.Errorf("Expected a POST message. Got %s", httpReq.Method)
	}

	var ortbRequest openrtb.BidRequest
	if err := json.Unmarshal(httpReq.Body, &ortbRequest); err != nil {
		t.Fatalf("Failed to unmarshal HTTP request: %v", ortbRequest)
	}

	if ortbRequest.ID != request.ID {
		t.Errorf("Bad Request ID. Expected %s, Got %s", request.ID, ortbRequest.ID)
	}
	if len(ortbRequest.Imp) != len(request.Imp) {
		t.Fatalf("Wrong len(request.Imp). Expected %d, Got %d", len(request.Imp), len(ortbRequest.Imp))
	}

	if ortbRequest.Imp[0].ID == "123" {

		if ortbRequest.Imp[0].Banner.Format[0].W != 300 {
			t.Fatalf("Banner width does not match. Expected %d, Got %d", 300, ortbRequest.Imp[0].Banner.Format[0].W)
		}
		if ortbRequest.Imp[0].Banner.Format[0].H != 250 {
			t.Fatalf("Banner height does not match. Expected %d, Got %d", 250, ortbRequest.Imp[0].Banner.Format[0].H)
		}
		if ortbRequest.Imp[0].BidFloor != 0.5 {
			t.Fatalf("Failed to Set BidFloor. Expected %f, Got %f", 0.5, ortbRequest.Imp[0].BidFloor)
		}
		if ortbRequest.Imp[0].TagID != "AdTag_Div1" {
			t.Fatalf("Failed to Set TqagID. Expected %s, Got %s", "AdTag_Div1", ortbRequest.Imp[0].TagID)
		}

		if ortbRequest.Imp[0].Ext == nil {
			t.Fatalf("Failed to add imp.Ext into outgoing request.")
		}

		if string(ortbRequest.Imp[0].Ext) != "\"keywords\":{\"pmZoneID\": \"Zone1,Zone2\",\"preference\": \"sports,movies\"}" {
			t.Fatalf("Failed to set  ortbRequest.Imp.Ext. Expected %s Actual %s ", "{\"wrapper\":{\"version\":1,\"profile\":5123}}", string(ortbRequest.Ext))
		}
	}

	if ortbRequest.App.Publisher.ID != "1234" {
		t.Fatalf("Failed to Publisher ID. Expected %s Actual %s", "1234", ortbRequest.Site.Publisher.ID)
	}

	if string(ortbRequest.Ext) != "{\"wrapper\":{\"version\":1,\"profile\":5123}}" {
		t.Fatalf("Failed to set  ortbRequest.Ext. Expected %s Actual %s ", "{\"wrapper\":{\"version\":1,\"profile\":5123}}", string(ortbRequest.Ext))
	}
}

var inValidPubmaticParams = []string{
	`{"bidder":{"adSlot":"AdTag_Div1@728x90","publisherId":"7890"}`,
	`{"bidder":{"publisherId":"7890"}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@728","publisherId":"7890"}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@valx728","publisherId":"7890"}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@728xval","publisherId":"7890"}}`,
	`{"bidder":{"adSlot":"AdTag_Div1","publisherId":"7890"}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@728x90:0"}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@728x90:0","publisherId":1}}`,
	`{"bidder":{"adSlot":123,"publisherId":"7890"}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@728x90","publisherId":"7890","keywords":{"pmZoneID": 1,"key": "v1,v2"}}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@728x90","publisherId":"7890","keywords":{"pmZoneID": "zone1","key": 1.2}}}`,
	`{"bidder":{"adSlot":"AdTag_Div1@728x90","publisherId":"7890","keywords":{"pmZoneID": "zone1"}, "wrapper":{"version":"1","profile":5123}}}`,
}

func TestOpenRTBBidRequest_InvalidParams(t *testing.T) {
	bidder := new(PubmaticAdapter)

	for _, param := range inValidPubmaticParams {

		request := &openrtb.BidRequest{
			ID: "12345",
			Imp: []openrtb.Imp{{
				ID: "234",
				Banner: &openrtb.Banner{
					Format: []openrtb.Format{{
						W: 300,
						H: 250,
					}},
				},
				Ext: openrtb.RawJSON(param),
			}},

			Device: &openrtb.Device{
				UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.132 Safari/537.36",
			},
			User: &openrtb.User{
				ID: "testID",
			},
			Site: &openrtb.Site{
				ID: "siteID",
			},
		}

		reqs, errs := bidder.MakeRequests(request)
		if len(errs) == 0 {
			t.Fatalf("Should get errors while Making HTTP requests for params = %v", param)
		}

		if len(reqs) != 0 {
			t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d for params = %v ", len(reqs), 0, param)
		}
	}

}
