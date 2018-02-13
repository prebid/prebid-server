package adtelligent

import (
	"encoding/json"
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/openrtb_ext"
	"net/http"
	"strings"
	"testing"
)

func TestAdtelligentAdapterInfo(t *testing.T) {
	an := NewAdtelligentAdapter(adapters.DefaultHTTPAdapterConfig)

	name := an.Name()
	if name != "Adtelligent" {
		t.Errorf("Name '%s' != 'Adtelligent'", name)
	}

	familyName := an.FamilyName()
	if familyName != "adtelligent" {
		t.Errorf("FamilyName '%s' != 'adtelligent'", familyName)
	}

	skipNoCookies := an.SkipNoCookies()
	if skipNoCookies != false {
		t.Errorf("SkipNoCookies should be false")
	}
}

func TestAdtelligentOpenRTBRequest(t *testing.T) {
	bidder := new(AdtelligentAdapter)

	bidReq := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-banner-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
				"sourceId": 1111
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
			Ext: openrtb.RawJSON(`{"bidder": {
				"sourceId": 2222,
				"bidFloor": 12
			}}`),
		}},
		Device: &openrtb.Device{
			DNT: 1,
			UA:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_5) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.140 Safari/537.36",
			IP:  "172.217.20.206",
		},
	}

	reqs, errs := bidder.MakeRequests(bidReq)

	if len(errs) > 0 {
		t.Errorf("Got unexpected errors while building HTTP requests: %v", errs)
	}
	if len(reqs) != 2 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 2)
	}

	for i := 0; i < len(reqs); i++ {
		httpReq := reqs[i]
		if httpReq.Method != "POST" {
			t.Errorf("Expected a POST message. Got %s", httpReq.Method)
		}

		var rpRequest openrtb.BidRequest
		if err := json.Unmarshal(httpReq.Body, &rpRequest); err != nil {
			t.Fatalf("Failed to unmarshal HTTP request: %v", rpRequest)
		}

		if rpRequest.ID != bidReq.ID {
			t.Errorf("Bad Request ID. Expected %s, Got %s", bidReq.ID, rpRequest.ID)
		}

		if len(rpRequest.Imp) != 1 {
			t.Fatalf("Wrong len(bidReq.Imp). Expected %d, Got %d", 1, len(rpRequest.Imp))
		}

		imp := rpRequest.Imp[0]

		var impExt adtelligentImpExt
		if err := json.Unmarshal(imp.Ext, &impExt); err != nil {
			t.Fatal("Error unmarshalling impExt from the outgoing bidReq")
		}

		if nil != imp.Banner {

			if imp.Banner.Format[0].W != 300 {
				t.Fatalf("Banner width does not match. Expected %d, Got %d", 300, imp.Banner.Format[0].W)
			}
			if imp.Banner.Format[0].H != 250 {
				t.Fatalf("Banner height does not match. Expected %d, Got %d", 250, imp.Banner.Format[0].H)
			}
			if imp.Banner.Format[1].W != 300 {
				t.Fatalf("Banner width does not match. Expected %d, Got %d", 300, imp.Banner.Format[1].W)
			}
			if imp.Banner.Format[1].H != 600 {
				t.Fatalf("Banner height does not match. Expected %d, Got %d", 600, imp.Banner.Format[1].H)
			}

			if imp.BidFloor != 0 {
				t.Fatalf("Bad Banner BidFloor. Expected %f, Got %f", float64(0), imp.BidFloor)
			}
		} else {

			if imp.Video.W != 640 {
				t.Fatalf("Video width does not match. Expected %d, Got %d", 640, imp.Video.W)
			}
			if imp.Video.H != 360 {
				t.Fatalf("Video height does not match. Expected %d, Got %d", 360, imp.Video.H)
			}
			if imp.Video.MIMEs[0] != "video/mp4" {
				t.Fatalf("Video MIMEs do not match. Expected %s, Got %s", "video/mp4", imp.Video.MIMEs[0])
			}
			if imp.Video.MinDuration != 15 {
				t.Fatalf("Video min duration does not match. Expected %d, Got %d", 15, imp.Video.MinDuration)
			}
			if imp.Video.MaxDuration != 30 {
				t.Fatalf("Video max duration does not match. Expected %d, Got %d", 30, imp.Video.MaxDuration)
			}

			if imp.BidFloor != 12 {
				t.Fatalf("Bad Video BidFloor. Expected %f, Got %f", float64(12), imp.BidFloor)
			}
		}

	}
}

func TestAdtelligentOpenRTBResponse(t *testing.T) {

	bidReq := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID: "test-imp-banner-id",
			Banner: &openrtb.Banner{
				Format: []openrtb.Format{{
					W: 300,
					H: 250,
				}, {
					W: 300,
					H: 600,
				}},
			},
			Ext: openrtb.RawJSON(`{"bidder": {
				"sourceId": 1111
			}}`),
		}},
	}

	reqJson, err := json.Marshal(bidReq)
	if nil != err {
		t.Fatalf("Error while encoding bidReq, err: %s", err)
	}

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    reqJson,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`
				{
					"bidid": "test-request-id",
					"cur": "USD",
					"id": "random-identifier",
					"seatbid": [
						{
							"bid": [
								{
									"id": "Fhb4whelQ4bFdO3g",
									"impid": "test-imp-banner-id",
									"price": 15.7,
									"adm": ".... adm ....",
									"w": 350,
									"h": 250
								}
							]
						}
					]
				}

		`),
	}

	bidder := new(AdtelligentAdapter)
	bids, errs := bidder.MakeBids(bidReq, reqData, httpResp)

	if len(bids) != 1 {
		t.Fatalf("Expected 1 bid. Got %d", len(bids))
	}

	if len(errs) != 0 {
		t.Errorf("Expected 0 errors. Got %d", len(errs))
	}

	if bids[0].BidType != openrtb_ext.BidTypeBanner {
		t.Errorf("Expected a banner bid. Got: %s", bids[0].BidType)
	}

}

func TestAdtelligentNoContentResponse(t *testing.T) {

	bidReq := &openrtb.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb.Imp{},
	}

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    nil,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(``),
	}

	bidder := new(AdtelligentAdapter)
	bids, errs := bidder.MakeBids(bidReq, reqData, httpResp)

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

	if len(errs) == 0 {
		t.Fatalf("Length of errs should be 0 instead of: %v", len(errs))
	}

}

func TestAdtelligentNot200OKResponse(t *testing.T) {

	bidReq := &openrtb.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb.Imp{},
	}

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    nil,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusBadRequest,
		Body:       []byte(``),
	}

	bidder := new(AdtelligentAdapter)
	bids, errs := bidder.MakeBids(bidReq, reqData, httpResp)

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

	if len(errs) != 1 {
		t.Fatalf("Length of errs should be 1 instead of: %v", len(errs))
	}

	if !strings.Contains(errs[0].Error(), "unexpected status code") {
		t.Fatalf("Unexpected error for NoContent status, should be: unexpected status code ... instead of: %s", errs[0].Error())
	}

}

func TestAdtelligentNotValidResponse(t *testing.T) {

	bidReq := &openrtb.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb.Imp{},
	}

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    nil,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body:       []byte(`some invalid response`),
	}

	bidder := new(AdtelligentAdapter)
	bids, errs := bidder.MakeBids(bidReq, reqData, httpResp)

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

	if len(errs) != 1 {
		t.Fatalf("Length of errs should be 1 instead of: %v", len(errs))
	}

	if !strings.Contains(errs[0].Error(), "error while decoding") {
		t.Fatalf("Unexpected error for InvalidRequest status, should be: error while decoding ... instead of: %s", errs[0].Error())
	}

}

func TestAdtelligentNotMappedBidAndImpressionIds(t *testing.T) {

	bidReq := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{
			{ID: "test-imp-banner-id"},
		},
	}

	reqData := &adapters.RequestData{
		Method:  "POST",
		Uri:     "test-uri",
		Body:    nil,
		Headers: nil,
	}

	httpResp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`

			{
				"bidid": "test-request-id",
				"cur": "USD",
				"id": "random-identifier",
				"seatbid": [
					{
						"bid": [
							{
								"id": "random-identifier",
								"impid": "not-valid-id"
							}
						]
					}
				]
			}

		`),
	}

	bidder := new(AdtelligentAdapter)
	bids, errs := bidder.MakeBids(bidReq, reqData, httpResp)

	if len(bids) != 0 {
		t.Fatalf("Length of bids should be 0 instead of: %v", len(bids))
	}

	if len(errs) != 1 {
		t.Fatalf("Length of errs should be 1 instead of: %v", len(errs))
	}

	if !strings.Contains(errs[0].Error(), "any impression with") {
		t.Fatalf("Unexpected error for InvalidRequest status, should be: error while decoding ... instead of: %s", errs[0].Error())
	}

}

func TestAdtelligentNotValidOpenRTBRequest(t *testing.T) {
	bidder := new(AdtelligentAdapter)

	bidReq := &openrtb.BidRequest{
		ID: "test-request-id",
		Imp: []openrtb.Imp{{
			ID:    "test-imp-banner-id",
			Audio: &openrtb.Audio{},
			Ext: openrtb.RawJSON(`{"bidder": {
				"sourceId": 1111
			}}`),
		}},
	}

	reqs, errs := bidder.MakeRequests(bidReq)

	if len(reqs) != 0 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 0)
	}

	if len(errs) != 1 {
		t.Fatalf("Length of errs should be 1 instead of: %v", len(errs))
	}

	if !strings.Contains(errs[0].Error(), "only Video and Banner") {
		t.Fatalf("Unexpected error for InvalidRequest status, should be: only Video and Banner ... instead of: %s", errs[0].Error())
	}

}

func TestAdtelligentEmptyRTBRequest(t *testing.T) {
	bidder := new(AdtelligentAdapter)

	bidReq := &openrtb.BidRequest{
		ID:  "test-request-id",
		Imp: []openrtb.Imp{},
	}

	reqs, errs := bidder.MakeRequests(bidReq)

	if len(reqs) != 0 {
		t.Fatalf("Unexpected number of HTTP requests. Got %d. Expected %d", len(reqs), 0)
	}

	if len(errs) != 1 {
		t.Fatalf("Length of errs should be 1 instead of: %v", len(errs))
	}

	if !strings.Contains(errs[0].Error(), "no impressions") {
		t.Fatalf("Unexpected error for InvalidRequest status, should be: no impressions ... instead of: %s", errs[0].Error())
	}

}
