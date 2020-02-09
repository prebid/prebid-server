package adoppler

import (
	"encoding/json"
	"net/http"
	"reflect"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/errortypes"
	"github.com/prebid/prebid-server/openrtb_ext"
)

var bidRequest *openrtb.BidRequest = &openrtb.BidRequest{
	ID: "req1",
	Imp: []openrtb.Imp{
		{
			ID:     "imp1",
			Banner: &openrtb.Banner{},
			Ext: json.RawMessage([]byte(
				`{"bidder": {"adunit": "10"}}`)),
		},
		{
			ID:    "imp2",
			Video: &openrtb.Video{},
			Ext: json.RawMessage([]byte(
				`{"bidder": {"adunit": "20"}}`)),
		},
	},
}

func TestMakeRequests(t *testing.T) {
	ads := NewAdopplerBidder("http://adoppler.com")
	datas, errs := ads.MakeRequests(bidRequest, nil)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}

	exps := []*adapters.RequestData{
		{
			Method: "POST",
			Uri:    "http://adoppler.com/processHeaderBid/10",
			Body:   []byte(`{"id":"req1-10","imp":[{"id":"imp1","banner":{},"ext":{"bidder":{"adunit":"10"}}}]}`),
			Headers: http.Header{
				"Accept":            []string{"application/json"},
				"Content-Type":      []string{"application/json;charset=utf-8"},
				"X-Openrtb-Version": []string{"2.5"},
			},
		},
		{
			Method: "POST",
			Uri:    "http://adoppler.com/processHeaderBid/20",
			Body:   []byte(`{"id":"req1-20","imp":[{"id":"imp2","video":{"mimes":null},"ext":{"bidder":{"adunit":"20"}}}]}`),
			Headers: http.Header{
				"Accept":            []string{"application/json"},
				"Content-Type":      []string{"application/json;charset=utf-8"},
				"X-Openrtb-Version": []string{"2.5"},
			},
		},
	}
	if len(exps) != len(datas) {
		t.Fatalf("%d != %d", len(exps), len(datas))
	}
	assertRequestData(t, exps[0], datas[0])
	assertRequestData(t, exps[1], datas[1])
}

func TestMakeBidsBadRequest(t *testing.T) {
	ads := NewAdopplerBidder("http://adoppler.com")
	datas, errs := ads.MakeRequests(bidRequest, nil)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
	resp := &adapters.ResponseData{
		StatusCode: http.StatusBadRequest,
	}

	_, errs = ads.MakeBids(bidRequest, datas[0], resp)
	if len(errs) != 1 {
		t.Fatalf("%d != %d", 1, len(errs))
	}
	if _, ok := errs[0].(*errortypes.BadInput); !ok {
		t.Fatalf("%v is not *errortypes.BadInput type", errs[0])
	}
}

func TestMakeBidsServerError(t *testing.T) {
	ads := NewAdopplerBidder("http://adoppler.com")
	datas, errs := ads.MakeRequests(bidRequest, nil)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
	resp := &adapters.ResponseData{
		StatusCode: http.StatusInternalServerError,
	}

	_, errs = ads.MakeBids(bidRequest, datas[0], resp)
	if len(errs) != 1 {
		t.Fatalf("%d != %d", 1, len(errs))
	}
	if _, ok := errs[0].(*errortypes.BadServerResponse); !ok {
		t.Fatalf("%v is not *errortypes.BadServerResponse type", errs[0])
	}
}

func TestMakeBidsNoContent(t *testing.T) {
	ads := NewAdopplerBidder("http://adoppler.com")
	datas, errs := ads.MakeRequests(bidRequest, nil)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
	resp := &adapters.ResponseData{
		StatusCode: http.StatusNoContent,
	}

	bidResp, errs := ads.MakeBids(bidRequest, datas[0], resp)
	if len(errs) != 0 {
		t.Fatalf("%d != %d", 0, len(errs))
	}
	if bidResp != nil {
		t.Fatalf("%v != nil", bidResp)
	}
}

func TestMakeBidsOKBanner(t *testing.T) {
	ads := NewAdopplerBidder("http://adoppler.com")
	datas, errs := ads.MakeRequests(bidRequest, nil)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
	resp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`
                    {"id": "resp1",
                     "seatbid": [{"bid": [{"id": "bid1",
                                           "impid": "imp1",
                                           "price": 1.50,
                                           "adm": "<b>a banner</b>"}]}]}
                `),
	}

	bidResp, errs := ads.MakeBids(bidRequest, datas[0], resp)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
	if bidResp == nil {
		t.Fatalf("nil == %v", bidResp)
	}
	assertIntEqual(t, 1, len(bidResp.Bids))
	bid := bidResp.Bids[0]
	assertStringEqual(t, string(openrtb_ext.BidTypeBanner),
		string(bid.BidType))
	if bid.BidVideo != nil {
		t.Fatalf("nil != %v", bid.BidVideo)
	}
	assertStringEqual(t, "bid1", bid.Bid.ID)
	assertStringEqual(t, "<b>a banner</b>", bid.Bid.AdM)
}

func TestMakeBidsOKVideo(t *testing.T) {
	ads := NewAdopplerBidder("http://adoppler.com")
	datas, errs := ads.MakeRequests(bidRequest, nil)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
	resp := &adapters.ResponseData{
		StatusCode: http.StatusOK,
		Body: []byte(`
                    {"id": "resp1",
                     "seatbid": [{"bid": [{"id": "bid1",
                                           "impid": "imp2",
                                           "price": 1.50,
                                           "adm": "<VAST />",
                                           "ext": {
                                               "ads": {
                                                   "video": {
                                                       "duration": 90
                                                   }
                                               }
                                           }}]}]}
                `),
	}

	bidResp, errs := ads.MakeBids(bidRequest, datas[1], resp)
	if len(errs) != 0 {
		t.Fatal(errs[0])
	}
	if bidResp == nil {
		t.Fatalf("nil == %v", bidResp)
	}
	assertIntEqual(t, 1, len(bidResp.Bids))
	bid := bidResp.Bids[0]
	assertStringEqual(t, string(openrtb_ext.BidTypeVideo),
		string(bid.BidType))
	if bid.BidVideo == nil {
		t.Fatalf("nil == %v", bid.BidVideo)
	}
	assertIntEqual(t, 90, bid.BidVideo.Duration)
	assertStringEqual(t, "bid1", bid.Bid.ID)
	assertStringEqual(t, "<VAST />", bid.Bid.AdM)
}

func assertRequestData(t *testing.T, exp *adapters.RequestData,
	act *adapters.RequestData) {

	if exp.Method != act.Method {
		t.Fatalf("%v != %v", exp.Method, act.Method)
	}
	if exp.Uri != act.Uri {
		t.Fatalf("%v != %v", exp.Uri, act.Uri)
	}
	if string(exp.Body) != string(act.Body) {
		t.Fatalf("%v != %v", string(exp.Body), string(act.Body))
	}
	if !reflect.DeepEqual(exp.Headers, act.Headers) {
		t.Fatalf("%v != %v", exp.Headers, act.Headers)
	}
}

func assertIntEqual(t *testing.T, exp int, act int) {
	t.Helper()
	if exp != act {
		t.Fatalf("%v != %v", exp, act)
	}
}

func assertStringEqual(t *testing.T, exp string, act string) {
	t.Helper()
	if exp != act {
		t.Fatalf("%v != %v", exp, act)
	}
}
