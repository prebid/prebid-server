package ucfunnel

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestUcfunnelAdapterNames(t *testing.T) {
	adapter := NewUcfunnelBidder("http://localhost/bid")
	adapterstest.VerifyStringValue(adapter.Name(), "ucfunnel", t)
}

func TestSkipNoCookies(t *testing.T) {
	adapter := NewUcfunnelBidder("http://localhost/bid")
	status := adapter.SkipNoCookies()
	if status != false {
		t.Errorf("actual = %t expected != %t", status, false)
	}
}

func TestMakeRequests(t *testing.T) {

	imp := openrtb.Imp{
		ID:     "1234",
		Banner: &openrtb.Banner{},
	}
	imp2 := openrtb.Imp{
		ID:    "1235",
		Video: &openrtb.Video{},
	}

	imp3 := openrtb.Imp{
		ID:    "1236",
		Audio: &openrtb.Audio{},
	}

	imp4 := openrtb.Imp{
		ID:     "1237",
		Native: &openrtb.Native{},
	}
	imp5 := openrtb.Imp{
		ID:     "1237",
		Native: &openrtb.Native{},
	}

	internalRequest01 := openrtb.BidRequest{Imp: []openrtb.Imp{}}
	internalRequest02 := openrtb.BidRequest{Imp: []openrtb.Imp{imp, imp2, imp3, imp4, imp5}}
	internalRequest03 := openrtb.BidRequest{Imp: []openrtb.Imp{imp, imp2, imp3, imp4, imp5}}

	internalRequest03.Imp[0].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[1].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[2].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[3].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[4].Ext = []byte(`{"bidder": {"adunitid": "aa","partnerid": ""}}`)

	adapter := NewUcfunnelBidder("http://localhost/bid")

	var testCases = []struct {
		in   []openrtb.BidRequest
		out1 [](int)
		out2 [](bool)
	}{
		{
			in:   []openrtb.BidRequest{internalRequest01, internalRequest02, internalRequest03},
			out1: [](int){1, 1, 0},
			out2: [](bool){false, false, true},
		},
	}

	for idx := range testCases {
		for i := range testCases[idx].in {
			RequestData, err := adapter.MakeRequests(&testCases[idx].in[i], nil)
			if ((RequestData == nil) == testCases[idx].out2[i]) && (len(err) == testCases[idx].out1[i]) {
				t.Errorf("actual = %v expected = %v", len(err), testCases[idx].out1[i])
			}
		}
	}
}

func TestMakeBids(t *testing.T) {
	imp := openrtb.Imp{
		ID:     "1234",
		Banner: &openrtb.Banner{},
	}
	imp2 := openrtb.Imp{
		ID:    "1235",
		Video: &openrtb.Video{},
	}

	imp3 := openrtb.Imp{
		ID:    "1236",
		Audio: &openrtb.Audio{},
	}

	imp4 := openrtb.Imp{
		ID:     "1237",
		Native: &openrtb.Native{},
	}
	imp5 := openrtb.Imp{
		ID:     "1237",
		Native: &openrtb.Native{},
	}

	internalRequest03 := openrtb.BidRequest{Imp: []openrtb.Imp{imp, imp2, imp3, imp4, imp5}}
	internalRequest04 := openrtb.BidRequest{Imp: []openrtb.Imp{imp}}

	internalRequest03.Imp[0].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[1].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[2].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[3].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[4].Ext = []byte(`{"bidder": {"adunitid": "aa","partnerid": ""}}`)
	internalRequest04.Imp[0].Ext = []byte(`{"bidder": {"adunitid": "0"}}`)

	mockResponse200 := adapters.ResponseData{StatusCode: 200, Body: json.RawMessage(`{"seatbid": [{"bid": [{"impid": "1234"}]},{"bid": [{"impid": "1235"}]},{"bid": [{"impid": "1236"}]},{"bid": [{"impid": "1237"}]}]}`)}
	mockResponse203 := adapters.ResponseData{StatusCode: 203, Body: json.RawMessage(`{"seatbid":[{"bid":[{"impid":"1234"}]},{"bid":[{"impid":"1235"}]}]}`)}
	mockResponse204 := adapters.ResponseData{StatusCode: 204, Body: json.RawMessage(`{"seatbid":[{"bid":[{"impid":"1234"}]},{"bid":[{"impid":"1235"}]}]}`)}
	mockResponse400 := adapters.ResponseData{StatusCode: 400, Body: json.RawMessage(`{"seatbid":[{"bid":[{"impid":"1234"}]},{"bid":[{"impid":"1235"}]}]}`)}
	mockResponseError := adapters.ResponseData{StatusCode: 200, Body: json.RawMessage(`{"seatbid":[{"bid":[{"im236"}],{"bid":[{"impid":"1237}]}`)}

	RequestData01 := adapters.RequestData{Method: "POST", Body: []byte(`{"imp":[{"id":"1234","banner":{}},{"id":"1235","video":{}},{"id":"1236","audio":{}},{"id":"1237","native":{}}]}`)}
	RequestData02 := adapters.RequestData{Method: "POST", Body: []byte(`{"imp":[{"id":"1234","banne"1235","video":{}},{"id":"1236","audio":{}},{"id":"1237","native":{}}]}`)}

	adapter := NewUcfunnelBidder("http://localhost/bid")

	var testCases = []struct {
		in1  []openrtb.BidRequest
		in2  []adapters.RequestData
		in3  []adapters.ResponseData
		out1 [](bool)
		out2 [](bool)
	}{
		{
			in1:  []openrtb.BidRequest{internalRequest03, internalRequest03, internalRequest03, internalRequest03, internalRequest03, internalRequest04},
			in2:  []adapters.RequestData{RequestData01, RequestData01, RequestData01, RequestData01, RequestData01, RequestData02},
			in3:  []adapters.ResponseData{mockResponse200, mockResponse203, mockResponse204, mockResponse400, mockResponseError, mockResponse200},
			out1: [](bool){true, false, false, false, false, false},
			out2: [](bool){false, true, false, true, true, true},
		},
	}

	for idx := range testCases {
		for i := range testCases[idx].in1 {
			BidderResponse, err := adapter.MakeBids(&testCases[idx].in1[i], &testCases[idx].in2[i], &testCases[idx].in3[i])

			if (BidderResponse == nil) == testCases[idx].out1[i] {
				fmt.Println(i)
				fmt.Println("BidderResponse")
				t.Errorf("actual = %t expected == %v", (BidderResponse == nil), testCases[idx].out1[i])
			}

			if (err == nil) == testCases[idx].out2[i] {
				fmt.Println(i)
				fmt.Println("error")
				t.Errorf("actual = %t expected == %v", err, testCases[idx].out2[i])
			}
		}
	}
}
