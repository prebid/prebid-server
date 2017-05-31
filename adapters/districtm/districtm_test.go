package districtm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/openrtb"
	"github.com/prebid/prebid-server/pbs"
)

func TestDistrictMInvalidCall(t *testing.T) {

	an := NewAdapter()
	an.URI = "dummy"
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

func TestDistrictMTimeout(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-time.After(2 * time.Millisecond)
		}),
	)
	defer server.Close()

	an := NewAdapter()
	an.URI = server.URL
	ctx, cancel := context.WithTimeout(context.Background(), 0)
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
				Params: json.RawMessage("{\"placementId\": 10}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("Timeout error not received for invalid request: %v", err)
	}
}

func TestDistrictMInvalidJson(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Blah")
		}),
	)
	defer server.Close()

	an := NewAdapter()
	an.URI = server.URL
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
				Params: json.RawMessage("{\"placementId\": 10}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestDistrictMInvalidStatusCode(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send 404
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}),
	)
	defer server.Close()

	an := NewAdapter()
	an.URI = server.URL
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
				Params: json.RawMessage("{\"placementId\": 10}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestMissingPlacementId(t *testing.T) {
	an := NewAdapter()
	an.URI = "dummy"
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
				Params: json.RawMessage("{\"XXX\": 10}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestDistrictMBasicResponse(t *testing.T) {

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

	an := NewAdapter()
	an.URI = server.URL
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
				Params: json.RawMessage("{\"placementId\": 10}"),
			},
		},
	}
	bids, err := an.Call(ctx, &pbReq, &pbBidder)
	if err != nil {
		t.Fatalf("Should not have gotten an error: %v", err)
	}
	if len(bids) != 1 {
		t.Fatalf("Did not receive 1 bid")
	}
}

func TestDistrictMUserSyncInfo(t *testing.T) {

	an := NewAdapter()
	an.Configure("localhost", nil)

	if an.GetUsersyncInfo().URL != "//ib.adnxs.com/getuid?localhost%2Fsetuid%3Fbidder%3Dadnxs%26uid%3D%24UID" {
		t.Fatalf("should have matched")
	}
	if an.GetUsersyncInfo().Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.GetUsersyncInfo().SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
