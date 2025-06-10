package ucfunnel

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/adapters"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestMakeRequests(t *testing.T) {

	imp := openrtb2.Imp{
		ID:     "1234",
		Banner: &openrtb2.Banner{},
	}
	imp2 := openrtb2.Imp{
		ID:    "1235",
		Video: &openrtb2.Video{},
	}

	imp3 := openrtb2.Imp{
		ID:    "1236",
		Audio: &openrtb2.Audio{},
	}

	imp4 := openrtb2.Imp{
		ID:     "1237",
		Native: &openrtb2.Native{},
	}
	imp5 := openrtb2.Imp{
		ID:     "1237",
		Native: &openrtb2.Native{},
	}

	internalRequest01 := openrtb2.BidRequest{Imp: []openrtb2.Imp{}}
	internalRequest02 := openrtb2.BidRequest{Imp: []openrtb2.Imp{imp, imp2, imp3, imp4, imp5}}
	internalRequest03 := openrtb2.BidRequest{Imp: []openrtb2.Imp{imp, imp2, imp3, imp4, imp5}}

	internalRequest03.Imp[0].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[1].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[2].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[3].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest03.Imp[4].Ext = []byte(`{"bidder": {"adunitid": "aa","partnerid": ""}}`)

	bidder, buildErr := Builder(openrtb_ext.BidderUcfunnel, config.Adapter{
		Endpoint: "http://localhost/bid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	var testCases = []struct {
		giveRequest openrtb2.BidRequest
		wantErr     bool
		wantRequest bool
		wantImpIDs  []string
	}{
		{
			giveRequest: internalRequest01,
			wantErr:     true,
			wantRequest: false,
			wantImpIDs:  []string{},
		},
		{
			giveRequest: internalRequest02,
			wantErr:     true,
			wantRequest: false,
			wantImpIDs:  []string{imp.ID, imp2.ID, imp3.ID, imp4.ID, imp5.ID},
		},
		{
			giveRequest: internalRequest03,
			wantErr:     false,
			wantRequest: true,
			wantImpIDs:  []string{imp.ID, imp2.ID, imp3.ID, imp4.ID, imp5.ID},
		},
	}

	for _, tc := range testCases {
		RequestData, err := bidder.MakeRequests(&tc.giveRequest, nil)
		if tc.wantErr {
			assert.Len(t, err, 1)
		} else {
			assert.Len(t, err, 0)
		}
		if tc.wantRequest {
			assert.Len(t, RequestData, 1)
			assert.ElementsMatch(t, tc.wantImpIDs, RequestData[0].ImpIDs)
		} else {
			assert.Len(t, RequestData, 0)
		}
	}
}

func TestMakeBids(t *testing.T) {
	imp := openrtb2.Imp{
		ID:     "1234",
		Banner: &openrtb2.Banner{},
	}
	imp2 := openrtb2.Imp{
		ID:    "1235",
		Video: &openrtb2.Video{},
	}

	imp3 := openrtb2.Imp{
		ID:    "1236",
		Audio: &openrtb2.Audio{},
	}

	imp4 := openrtb2.Imp{
		ID:     "1237",
		Native: &openrtb2.Native{},
	}
	imp5 := openrtb2.Imp{
		ID:     "1237",
		Native: &openrtb2.Native{},
	}

	internalRequest03 := openrtb2.BidRequest{Imp: []openrtb2.Imp{imp, imp2, imp3, imp4, imp5}}
	internalRequest04 := openrtb2.BidRequest{Imp: []openrtb2.Imp{imp}}

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

	bidder, buildErr := Builder(openrtb_ext.BidderUcfunnel, config.Adapter{
		Endpoint: "http://localhost/bid"}, config.Server{ExternalUrl: "http://hosturl.com", GvlID: 1, DataCenter: "2"})

	if buildErr != nil {
		t.Fatalf("Builder returned unexpected error %v", buildErr)
	}

	var testCases = []struct {
		in1  []openrtb2.BidRequest
		in2  []adapters.RequestData
		in3  []adapters.ResponseData
		out1 [](bool)
		out2 [](bool)
	}{
		{
			in1:  []openrtb2.BidRequest{internalRequest03, internalRequest03, internalRequest03, internalRequest03, internalRequest03, internalRequest04},
			in2:  []adapters.RequestData{RequestData01, RequestData01, RequestData01, RequestData01, RequestData01, RequestData02},
			in3:  []adapters.ResponseData{mockResponse200, mockResponse203, mockResponse204, mockResponse400, mockResponseError, mockResponse200},
			out1: [](bool){true, false, false, false, false, false},
			out2: [](bool){false, true, false, true, true, true},
		},
	}

	for idx := range testCases {
		for i := range testCases[idx].in1 {
			BidderResponse, err := bidder.MakeBids(&testCases[idx].in1[i], &testCases[idx].in2[i], &testCases[idx].in3[i])

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
