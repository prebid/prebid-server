package adapters

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
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/pbs"
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
			W:      imp.Banner.W,
			H:      imp.Banner.H,
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

	an := NewPubmaticAdapter(DefaultHTTPAdapterConfig, "blah", "localhost")

	s := an.Name()
	if s == "" {
		t.Fatal("Missing name")
	}

	ctx := context.TODO()
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

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")
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

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")
	ctx := context.TODO()
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

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")
	ctx := context.TODO()
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

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, "http://localhost/test", "localhost")
	ctx := context.TODO()
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

	// Invalid Request JSON
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x240\"")
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Missing adSlot in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Missing publisher ID
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"adSlot\": \"slot@120x240\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Missing slot name  in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"@120x240\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Invalid adSize in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120-240\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Missing impression width and height in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Missing height  in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Missing width  in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@x120\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Incorrect width param  in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@valx120\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Incorrect height param  in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120xval\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Empty slot name in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \" @120x240\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Empty width in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@ x240\"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Empty height in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"slot@120x \"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

	// Empty height in AdUnits.Params
	pbBidder.AdUnits[0].Params = json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \" @120x \"}")
	_, err = an.Call(ctx, &pbReq, &pbBidder)
	CompareStringValue(err.Error(), "Incorrect adSlot / Publisher param", t)

}

func TestPubmaticBasicResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")
	ctx := context.TODO()
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

func TestPubmaticMultiImpressionResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")

	ctx := context.TODO()
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
	if len(bids) != 1 {
		t.Fatalf("Should have received one bid")
	}
}

func TestPubmaticMultiAdUnitResponse(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")

	ctx := context.TODO()
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

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")

	ctx := context.TODO()
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

func TestPubmaticUserSyncInfo(t *testing.T) {

	an := NewPubmaticAdapter(DefaultHTTPAdapterConfig, "pubmaticUrl", "localhost")
	if an.usersyncInfo.URL != "//ads.pubmatic.com/AdServer/js/user_sync.html?predirect=localhost%2Fsetuid%3Fbidder%3Dpubmatic%26uid%3D" {
		t.Fatalf("should have matched")
	}
	if an.usersyncInfo.Type != "iframe" {
		t.Fatalf("should be iframe")
	}
	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}

func TestPubmaticInvalidLookupBidIDParameter(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(DummyPubMaticServer))
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")

	ctx := context.TODO()
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

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")

	ctx := context.TODO()
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
	pc := pbs.ParsePBSCookieFromRequest(httpReq)
	pc.TrySync("pubmatic", "12345")
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "")
	httpReq.Header.Add("Cookie", fakewriter.Header().Get("Set-Cookie"))

	cacheClient, _ := dummycache.New()

	_, err = pbs.ParsePBSRequest(httpReq, cacheClient)
	if err != nil {
		t.Fatalf("Error when parsing request: %v", err)
	}
}
