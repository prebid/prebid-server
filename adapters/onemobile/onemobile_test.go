package onemobile

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
	"github.com/prebid/prebid-server/adapters/adapterstest"
	"github.com/prebid/prebid-server/cache/dummycache"
	"github.com/prebid/prebid-server/config"
	"github.com/prebid/prebid-server/pbs"
	"github.com/prebid/prebid-server/usersync"
)

/**
 * Verify adapter names are setup correctly.
 */
func TestOneMobileAdapterNames(t *testing.T) {
	adapter := NewOneMobileAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	adapterstest.VerifyStringValue(adapter.Name(), "onemobile", t)
}

/**
 * Test required parameters not sent
 */
func TestPulsePointRequiredBidParameters(t *testing.T) {
	adapter := NewOneMobileAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	ctx := context.TODO()
	req := SampleRequest(1, t)
	bidder := req.Bidders[0]

	bidder.AdUnits[0].Params = json.RawMessage("{\"dcn\": \"\", \"pos\": \"header\"}")
	_, errTag := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errTag.Error(), "Missing param dcn", t)

	// remove "cp" param and verify error message.
	bidder.AdUnits[0].Params = json.RawMessage("{\"dcn\": \"d1g31s\", \"pos\": \"\"}")
	_, errPub := adapter.Call(ctx, req, bidder)
	adapterstest.VerifyStringValue(errPub.Error(), "Missing PublisherId param cp", t)
}

/**
 * Produces a sample PBSRequest, for the impressions given.
 */
func SampleRequest(numberOfImpressions int, t *testing.T) *pbs.PBSRequest {
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
					BidderCode: "onemobile",
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
	httpReq := httptest.NewRequest("GET", CreateService(adapterstest.BidOnTags("")).Server.URL, body)
	httpReq.Header.Add("Referer", "http://news.pub/topnews")
	pc := usersync.ParsePBSCookieFromRequest(httpReq, &config.HostCookie{})
	pc.TrySync("pulsepoint", "pulsepointUser123")
	fakewriter := httptest.NewRecorder()
	pc.SetCookieOnResponse(fakewriter, "", 90*24*time.Hour)
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
