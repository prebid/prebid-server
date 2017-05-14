package adapters

import (
	"context"
	"encoding/json"
	"github.com/prebid/prebid-server/pbs"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fmt"

	"github.com/prebid/openrtb"
)

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
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code: "unitCode",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"12\"}"),
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
				Code: "unitCode",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"12\"}"),
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
				Code: "unitCode",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"12\"}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestPubmaticMissingPublisherId(t *testing.T) {

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
				Code: "unitCode",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"adSlot\": \"12\"}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for missing publisherId")
	}
}

func TestPubmaticMissingAdSlot(t *testing.T) {

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
				Code: "unitCode",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"PublisherId\": \"10\"}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for missing adSlot")
	}
}

func TestPubmaticBasicResponse(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := openrtb.BidResponse{
				SeatBid: []openrtb.SeatBid{
					{
						Bid: []openrtb.Bid{
							{
								ID:     "1234",
								ImpID:  "unitCode",
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

	conf := *DefaultHTTPAdapterConfig
	an := NewPubmaticAdapter(&conf, server.URL, "localhost")
	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{
		BidderCode: "bannerCode",
		AdUnits: []pbs.PBSAdUnit{
			{
				Code:  "unitCode",
				BidID: "bidid",
				Sizes: []openrtb.Format{
					{
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"publisherId\": \"10\", \"adSlot\": \"12\"}"),
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
