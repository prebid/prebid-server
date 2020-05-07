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

	"github.com/PubMatic-OpenWrap/openrtb"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters"
	"github.com/PubMatic-OpenWrap/prebid-server/adapters/adapterstest"
	"github.com/PubMatic-OpenWrap/prebid-server/cache/dummycache"
	"github.com/PubMatic-OpenWrap/prebid-server/config"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"github.com/PubMatic-OpenWrap/prebid-server/pbs"
	"github.com/PubMatic-OpenWrap/prebid-server/usersync"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "pubmatictest", NewPubmaticBidder(nil, "https://hbopenbid.pubmatic.com/translator?source=prebid-server"))
}

// ----------------------------------------------------------------------------
// Code below this line tests the legacy, non-openrtb code flow. It can be deleted after we
// clean up the existing code and make everything openrtb.

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
		bid := openrtb.Bid{
			ID:     fmt.Sprintf("SeatID_%d", i),
			ImpID:  imp.ID,
			Price:  float64(int(rand.Float64()*1000)) / 100,
			AdID:   fmt.Sprintf("adID-%d", i),
			AdM:    "AdContent",
			CrID:   fmt.Sprintf("creative-%d", i),
			W:      *imp.Banner.W,
			H:      *imp.Banner.H,
			DealID: fmt.Sprintf("DealID_%d", i),
		}
		bid.Ext = json.RawMessage("{\"buyid\": \"testBuyId\"}")

		bids = append(bids, bid)
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

	pc.SetCookieOnResponse(fakewriter, false, "", &config.HostCookie{Domain: ""}, 90*24*time.Hour)
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

func TestGetBidTypeVideo(t *testing.T) {
	extJSON := `{"BidType":1}`
	extrm := json.RawMessage(extJSON)
	actualBidTypeValue := getBidType(extrm)
	if actualBidTypeValue != openrtb_ext.BidTypeVideo {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeVideo, actualBidTypeValue)
	}
}

func TestGetBidTypeForMissingBidTypeExt(t *testing.T) {
	extJSON := `{}`
	extrm := json.RawMessage(extJSON)
	actualBidTypeValue := getBidType(extrm)
	// banner is the default bid type when no bidType key is present in the bid.ext
	if actualBidTypeValue != "banner" {
		t.Errorf("Expected Bid Type value was: banner, actual value is: %v", actualBidTypeValue)
	}
}

func TestGetBidTypeBanner(t *testing.T) {
	extJSON := `{"BidType":0}`
	extrm := json.RawMessage(extJSON)
	actualBidTypeValue := getBidType(extrm)
	if actualBidTypeValue != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeBanner, actualBidTypeValue)
	}
}

func TestGetBidTypeNative(t *testing.T) {
	extJSON := `{"BidType":2}`
	extrm := json.RawMessage(extJSON)
	actualBidTypeValue := getBidType(extrm)
	if actualBidTypeValue != openrtb_ext.BidTypeNative {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeNative, actualBidTypeValue)
	}
}

func TestGetBidTypeForUnsupportedCode(t *testing.T) {
	extJSON := `{"BidType":99}`
	extrm := json.RawMessage(extJSON)
	actualBidTypeValue := getBidType(extrm)
	if actualBidTypeValue != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected Bid Type value was: %v, actual value is: %v", openrtb_ext.BidTypeBanner, actualBidTypeValue)
	}
}

func TestAppendAdServerTargetingForEmptyExt(t *testing.T) {
	pbid := pbs.PBSBid{}
	ext := json.RawMessage(`{}`)
	appendTargetingKey(ext, &pbid)
	// banner is the default bid type when no bidType key is present in the bid.ext
	if pbid.AdServerTargeting != nil {
		t.Errorf("It should not contained AdserverTageting")
	}
}

func TestAppendAdServerTargetingForValidExt(t *testing.T) {
	pbid := pbs.PBSBid{}
	ext := json.RawMessage(`{"buyid":"testBuyId"}`)
	appendTargetingKey(ext, &pbid)
	// banner is the default bid type when no bidType key is present in the bid.ext
	if pbid.AdServerTargeting == nil {
		t.Error("It should have AdserverTageting")
		t.FailNow()
	}
	if pbid.AdServerTargeting != nil && pbid.AdServerTargeting["hb_buyid_pubmatic"] == "" {
		t.Errorf("It should have AdserverTageting and have hb_buyid_pubmatic")
	}
	if pbid.AdServerTargeting["hb_buyid_pubmatic"] != "testBuyId" {
		t.Errorf("It should have value testBuyId")
	}
}
