package adapters

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/pbs"

	"fmt"

	"github.com/prebid/openrtb"
)

func TestIndexInvalidCall(t *testing.T) {

	an := NewIndexAdapter(DefaultHTTPAdapterConfig, "http://appnexus-eu.lb.indexww.com/bidder?p=184932", "localhost")
	an.URI = "blah"
	s := an.Name()
	if s == "" {
		t.Fatal("Missing name")
	}

	ctx := context.TODO()
	pbReq := pbs.PBSRequest{}
	pbBidder := pbs.PBSBidder{}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error recived for invalid request")
	}
}

func TestIndexTimeout(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-time.After(2 * time.Millisecond)
		}),
	)
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewIndexAdapter(&conf, server.URL, "localhost")
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
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"siteID\": 12}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil || err != context.DeadlineExceeded {
		t.Fatalf("Invalid timeout error received")
	}
}

func TestIndexInvalidJson(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "Blah")
		}),
	)
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewIndexAdapter(&conf, server.URL, "localhost")
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
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"siteID\": 12}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestIndexInvalidStatusCode(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send 404
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}),
	)
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewIndexAdapter(&conf, server.URL, "localhost")
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
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"siteID\": 12}"),
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for invalid request")
	}
}

func TestIndexMissingSiteId(t *testing.T) {

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Send 404
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		}),
	)
	defer server.Close()

	conf := *DefaultHTTPAdapterConfig
	an := NewIndexAdapter(&conf, server.URL, "localhost")
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
						W: 10,
						H: 12,
					},
				},
			},
		},
	}
	_, err := an.Call(ctx, &pbReq, &pbBidder)
	if err == nil {
		t.Fatalf("No error received for missing siteID")
	}
}

func TestIndexBasicResponse(t *testing.T) {

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
	an := NewIndexAdapter(&conf, server.URL, "localhost")
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
						W: 10,
						H: 12,
					},
				},
				Params: json.RawMessage("{\"siteID\": 12}"),
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

func TestIndexUserSyncInfo(t *testing.T) {

	an := NewIndexAdapter(DefaultHTTPAdapterConfig, "http://appnexus-eu.lb.indexww.com/bidder?p=184932", "localhost")
	if an.usersyncInfo.URL != "//ssum-sec.casalemedia.com/usermatchredir?s=184932&cb=localhost%2Fsetuid%3Fbidder%3DindexExchange%26uid%3D__UID__" {
		t.Fatalf("should have matched")
	}
	if an.usersyncInfo.Type != "redirect" {
		t.Fatalf("should be redirect")
	}
	if an.usersyncInfo.SupportCORS != false {
		t.Fatalf("should have been false")
	}
}
