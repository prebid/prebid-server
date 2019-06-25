package ix

import (
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
	"github.com/prebid/prebid-server/pbs"
)

const url string = "http://appnexus-us-east.lb.indexww.com/bidder?p=184932"

func getAdUnit() pbs.PBSAdUnit {
	return pbs.PBSAdUnit{
		Code:       "unitCode",
		MediaTypes: []pbs.MediaType{pbs.MEDIA_TYPE_BANNER},
		BidID:      "bidid",
		Sizes: []openrtb.Format{
			{
				W: 10,
				H: 12,
			},
		},
		Params: json.RawMessage("{\"siteId\":\"12\"}"),
	}
}

func getOpenRTBBid(i openrtb.Imp) openrtb.Bid {
	return openrtb.Bid{
		ID:     fmt.Sprintf("%d", rand.Intn(1000)),
		ImpID:  i.ID,
		Price:  1.0,
		AdM:    "Content",
		CrID:   fmt.Sprintf("%d", rand.Intn(1000)),
		W:      *i.Banner.W,
		H:      *i.Banner.H,
		DealID: "5",
	}
}

func dummyIXServer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)

	var breq openrtb.BidRequest
	err = json.Unmarshal(body, &breq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	impression := breq.Imp[0]

	resp := openrtb.BidResponse{
		SeatBid: []openrtb.SeatBid{
			{
				Bid: []openrtb.Bid{
					getOpenRTBBid(impression),
				},
			},
		},
	}

	js, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

func TestIxInvalidCall(t *testing.T) {

	an := NewIxAdapter(adapters.DefaultHTTPAdapterConfig, url)
	an.URI = "blah"

	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestIxInvalidCallReqAppNil(t *testing.T) {

	an := NewIxAdapter(adapters.DefaultHTTPAdapterConfig, url)
	an.URI = "blah"

	ctx := context.TODO()
	pbReq := pbs.PBSRequest{
		App: &openrtb.App{},
	}

	pbBidder := pbs.PBSBidder{}
	_, err := an.Call(ctx, &pbReq, &pbBidder)

	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestIxInvalidCallMissingSiteID(t *testing.T) {

	an := NewIxAdapter(adapters.DefaultHTTPAdapterConfig, url)
	an.URI = "blah"

	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	adUnit := getAdUnit()
	adUnit.Params = json.RawMessage("{}")

	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			adUnit,
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for request with missing siteId")
	}
}

func TestIxTimeout(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-time.After(2 * time.Millisecond)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 0)
	defer cancel()

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			getAdUnit(),
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("Invalid timeout error received")
	}
}

func TestIxTimeoutMultipleSlots(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			body, err := ioutil.ReadAll(r.Body)

			var breq openrtb.BidRequest
			err = json.Unmarshal(body, &breq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			impression := breq.Imp[0]

			resp := openrtb.BidResponse{
				SeatBid: []openrtb.SeatBid{
					{
						Bid: []openrtb.Bid{
							getOpenRTBBid(impression),
						},
					},
				},
			}

			js, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// cancel the request before 2nd impression is returned
			// delay to let 1st impression return successfully
			if impression.ID == "unitCode2" {
				<-time.After(10 * time.Millisecond)
				cancel()
				<-r.Context().Done()
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	pbReq := pbs.PBSRequest{}

	adUnit1 := getAdUnit()
	adUnit2 := getAdUnit()
	adUnit2.Code = "unitCode2"
	adUnit2.Sizes = []openrtb.Format{
		{
			W: 8,
			H: 10,
		},
	}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			adUnit1,
			adUnit2,
		},
	}
	bids, err := an.Call(ctx, &pbReq, &pbBidder)

	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

	if len(bids) != 1 {
		t.Fatalf("Should have received one bid")
	}

	bid := findBidByAdUnitCode(bids, adUnit1.Code)
	if adUnit1.Sizes[0].H != bid.Height || adUnit1.Sizes[0].W != bid.Width {
		t.Fatalf("Received the wrong size")
	}
}

func TestIxInvalidJsonResponse(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Blah")
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			getAdUnit(),
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestIxInvalidStatusCode(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send 404
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			getAdUnit(),
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestIxBadRequest(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send 400
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			getAdUnit(),
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for bad request")
	}
}

func TestIxNoContent(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send 204
			http.Error(w, http.StatusText(http.StatusNoContent), http.StatusNoContent)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			getAdUnit(),
		},
	}

	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil || bids != nil {
		t.Fatalf("Must return nil for no content")
	}
}

func TestIxInvalidCallMissingSize(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(dummyIXServer),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	adUnit := getAdUnit()
	adUnit.Sizes = []openrtb.Format{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			adUnit,
		},
	}
	if _, err := an.Call(ctx, &pbReq, &pbBidder); err == nil {
		t.Fatalf("Should not have gotten an error for missing/invalid size: %v", err)
	}
}

func TestIxInvalidCallEmptyBidIDResponse(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(dummyIXServer),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	adUnit := getAdUnit()
	adUnit.BidID = ""
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			adUnit,
		},
	}
	if _, err := an.Call(ctx, &pbReq, &pbBidder); err == nil {
		t.Fatalf("Should have gotten an error for unknown adunit code")
	}
}

func TestIxMismatchUnitCode(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			body, err := ioutil.ReadAll(r.Body)

			var breq openrtb.BidRequest
			err = json.Unmarshal(body, &breq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			resp := openrtb.BidResponse{
				SeatBid: []openrtb.SeatBid{
					{
						Bid: []openrtb.Bid{
							{
								ID:     fmt.Sprintf("%d", rand.Intn(1000)),
								ImpID:  "unitCode_bogus",
								Price:  1.0,
								AdM:    "Content",
								CrID:   "567",
								W:      10,
								H:      12,
								DealID: "5",
							},
						},
					},
				},
			}

			js, err := json.Marshal(resp)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write(js)
		}),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			getAdUnit(),
		},
	}
	if _, err := an.Call(ctx, &pbReq, &pbBidder); err == nil {
		t.Fatalf("Should have gotten an error for unknown adunit code")
	}
}

func TestIxInvalidParam(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(dummyIXServer),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	adUnit := getAdUnit()
	adUnit.Params = json.RawMessage("Bogus invalid input")
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			adUnit,
		},
	}
	if _, err := an.Call(ctx, &pbReq, &pbBidder); err == nil {
		t.Fatalf("Should have gotten an error for unrecognized params")
	}
}

func TestIxSingleSlotSingleValidSize(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(dummyIXServer),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			getAdUnit(),
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

func TestIxTwoSlotValidSize(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(dummyIXServer),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	adUnit1 := getAdUnit()
	adUnit2 := getAdUnit()
	adUnit2.Code = "unitCode2"
	adUnit2.Sizes = []openrtb.Format{
		{
			W: 8,
			H: 10,
		},
	}
	adUnit2.Params = json.RawMessage("{\"siteId\":\"1111\"}")

	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			adUnit1,
			adUnit2,
		},
	}
	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

	if len(bids) != 2 {
		t.Fatalf("Should have received two bid")
	}

	bid := findBidByAdUnitCode(bids, adUnit1.Code)
	if adUnit1.Sizes[0].H != bid.Height || adUnit1.Sizes[0].W != bid.Width {
		t.Fatalf("Received the wrong size")
	}

	bid = findBidByAdUnitCode(bids, adUnit2.Code)
	if adUnit2.Sizes[0].H != bid.Height || adUnit2.Sizes[0].W != bid.Width {
		t.Fatalf("Received the wrong size")
	}
}

func TestIxTwoSlotMultiSizeOnlyValidIXSizeResponse(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(dummyIXServer),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	adUnit := getAdUnit()
	adUnit.Sizes = append(adUnit.Sizes, openrtb.Format{W: 20, H: 22})

	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			adUnit,
		},
	}
	bids, err := an.Call(ctx, &pbReq, &pbBidder)

	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

	if len(bids) != 2 {
		t.Fatalf("Should have received 2 bids")
	}

	for _, size := range adUnit.Sizes {
		if !bidResponseForSizeExist(bids, size.H, size.W) {
			t.Fatalf("Missing bid for specified size %d and %d", size.W, size.H)
		}
	}
}

func bidResponseForSizeExist(bids pbs.PBSBidSlice, h uint64, w uint64) bool {
	for _, v := range bids {
		if v.Height == h && v.Width == w {
			return true
		}
	}
	return false
}

func findBidByAdUnitCode(bids pbs.PBSBidSlice, c string) *pbs.PBSBid {
	for _, v := range bids {
		if v.AdUnitCode == c {
			return v
		}
	}
	return &pbs.PBSBid{}
}

func TestIxRequestLimit(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(dummyIXServer),
	)
	defer server.Close()

	conf := *adapters.DefaultHTTPAdapterConfig
	an := NewIxAdapter(&conf, server.URL)
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	adUnits := []pbs.PBSAdUnit{}

	for i := 0; i < requestLimit+1; i++ {
		adUnits = append(adUnits, getAdUnit())
	}

	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits:    adUnits,
	}

	bids, err := an.Call(ctx, &pbReq, &pbBidder)

	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}

	if len(bids) != requestLimit {
		t.Fatalf("Should have received %d bid", requestLimit)
	}
}
