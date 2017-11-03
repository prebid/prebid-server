package sovrn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/pbs"

	"context"
	"net/http"

	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/cache/dummycache"
)

func TestSovrnUserSyncInfo(t *testing.T) {
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, "http://sovrn/rtb/bid", "http://sovrn/userSync?", "http://localhost:8000")
	adapters.VerifyStringValue(adapter.GetUsersyncInfo().Type, "redirect", t)
	adapters.VerifyStringValue(adapter.GetUsersyncInfo().URL, "http://sovrn/userSync?location=http%3A%2F%2Flocalhost%3A8000%2Fsetuid%3Fbidder%3Dsovrn%26uid%3D%5BSOVRNID%5D", t)
}

func TestSovrnOpenRtbRequest(t *testing.T) {
	service := CreateSovrnService(adapters.BidOnTags(""))
	server := service.Server
	ctx := context.TODO()
	req := SampleSovrnRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL, "http://sovrn/userSync?", "http://localhost")
	adapter.Call(ctx, req, bidder)

	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 1, t)
	adapters.VerifyStringValue(service.LastBidRequest.Imp[0].TagID, "123456", t)
	adapters.VerifyBannerSize(service.LastBidRequest.Imp[0].Banner, 728, 90, t)
}

func TestSovrnBiddingBehavior(t *testing.T) {
	server := CreateSovrnService(adapters.BidOnTags("123456")).Server
	ctx := context.TODO()
	req := SampleSovrnRequest(1, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL, "http://sovrn/userSync?", "http://localhost")
	bids, _ := adapter.Call(ctx, req, bidder)

	adapters.VerifyIntValue(len(bids), 1, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapters.VerifyStringValue(bids[0].BidderCode, "sovrn", t)
	adapters.VerifyStringValue(bids[0].Adm, "<div>This is an Ad</div>", t)
	adapters.VerifyStringValue(bids[0].Creative_id, "Cr-234", t)
	adapters.VerifyIntValue(int(bids[0].Width), 728, t)
	adapters.VerifyIntValue(int(bids[0].Height), 90, t)
	adapters.VerifyIntValue(int(bids[0].Price*100), 210, t)
}

/**
 * Verify bidding behavior on multiple impressions, some impressions make a bid
 */
func TestSovrntMultiImpPartialBidding(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateSovrnService(adapters.BidOnTags("123456"))
	server := service.Server
	ctx := context.TODO()
	req := SampleSovrnRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL, "http://sovrn/userSync?", "http://localhost")
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
func TestSovrnMultiImpAllBid(t *testing.T) {
	// setup server endpoint to return bid.
	service := CreateSovrnService(adapters.BidOnTags("123456,123457"))
	server := service.Server
	ctx := context.TODO()
	req := SampleSovrnRequest(2, t)
	bidder := req.Bidders[0]
	adapter := NewSovrnAdapter(adapters.DefaultHTTPAdapterConfig, server.URL, "http://sovrn/userSync?", "http://localhost")
	bids, _ := adapter.Call(ctx, req, bidder)
	// two impressions sent.
	// number of bids should be 1
	adapters.VerifyIntValue(len(service.LastBidRequest.Imp), 2, t)
	adapters.VerifyIntValue(len(bids), 2, t)
	adapters.VerifyStringValue(bids[0].AdUnitCode, "div-adunit-1", t)
	adapters.VerifyStringValue(bids[1].AdUnitCode, "div-adunit-2", t)
}

func SampleSovrnRequest(numberOfImpressions int, t *testing.T) *pbs.PBSRequest {
	req := pbs.PBSRequest{
		AccountID: "1",
		AdUnits:   make([]pbs.AdUnit, 2),
	}
	tagID := 123456

	for i := 0; i < numberOfImpressions; i++ {
		req.AdUnits[i] = pbs.AdUnit{
			Code: fmt.Sprintf("div-adunit-%d", i+1),
			Sizes: []openrtb.Format{
				{
					W: 728,
					H: 90,
				},
			},
			Bids: []pbs.Bids{
				{
					BidderCode: "sovrn",
					BidID:      fmt.Sprintf("Bid-%d", i+1),
					Params:     json.RawMessage(fmt.Sprintf("{\"tagid\": %d }", tagID+i)),
				},
			},
		}

	}

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	if err != nil {
		t.Fatalf("Error when serializing request")
	}

	httpReq := httptest.NewRequest("POST", CreateSovrnService(adapters.BidOnTags("")).Server.URL, body)
	httpReq.Header.Add("Referer", "http://news.pub/topnews")
	pc := pbs.ParsePBSCookieFromRequest(httpReq)
	pc.TrySync("sovrn", "sovrnUser123")
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "")
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

func CreateSovrnService(tagsToBid map[string]bool) adapters.OrtbMockService {
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
	}))

	service.Server = server
	service.LastBidRequest = &lastBidRequest

	return service
}
