package ucfunnel

import (
	"encoding/json"
	"testing"

	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/adapters"
	"github.com/prebid/prebid-server/adapters/adapterstest"
)

func TestJsonSamples(t *testing.T) {
	adapterstest.RunJSONBidderTest(t, "ucfunneltest", NewUcfunnelBidder(nil, "http://127.0.0.1:8081/tmp12.json"))
}

func TestAddHeadersToRequest(t *testing.T) {
	header := AddHeadersToRequest()

	if header == nil {
		t.Errorf("actual = %v expected != %v", nil, nil)
	}

	if header.Get("Content-Type") != "application/json;charset=utf-8" {
		t.Errorf("actual = %s expected != %v", header.Get("Content-Type"), nil)
	}

	if header.Get("Accept") != "application/json" {
		t.Errorf("actual = %s expected != %v", header.Get("Accept"), nil)
	}
}

func TestUcfunnelAdapterNames(t *testing.T) {
	adapter := NewUcfunnelAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	adapterstest.VerifyStringValue(adapter.Name(), "ucfunnel", t)
}

func TestSkipNoCookies(t *testing.T) {
	adapter := NewUcfunnelAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
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

	internalRequest := openrtb.BidRequest{Imp: []openrtb.Imp{imp, imp2}}
	adapter := NewUcfunnelAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	RequestData, err := adapter.MakeRequests(&internalRequest, nil)

	internalRequest.Imp[0].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)
	internalRequest.Imp[1].Ext = []byte(`{"bidder": {"adunitid": "ad-488663D474E44841E8A293379892348","partnerid": "par-7E6D2DB9A8922AB07B44A444D2BA67"}}`)

	adapter = NewUcfunnelAdapter(adapters.DefaultHTTPAdapterConfig, "http://localhost/bid")
	RequestData, err = adapter.MakeRequests(&internalRequest, nil)
	adapterstest.VerifyStringValue(RequestData[0].Method, "POST", t)
	adapterstest.VerifyStringValue(RequestData[0].Uri, adapter.URI+"par-7E6D2DB9A8922AB07B44A444D2BA67/request", t)

	mockResponse200 := adapters.ResponseData{StatusCode: 200, Body: json.RawMessage(`{"seatbid":[{"bid":[{"impid":"1234"}]},{"bid":[{"impid":"1235"}]}]}`)}
	mockResponse203 := adapters.ResponseData{StatusCode: 203, Body: json.RawMessage(`{"seatbid":[{"bid":[{"impid":"1234"}]},{"bid":[{"impid":"1235"}]}]}`)}
	mockResponse204 := adapters.ResponseData{StatusCode: 204, Body: json.RawMessage(`{"seatbid":[{"bid":[{"impid":"1234"}]},{"bid":[{"impid":"1235"}]}]}`)}
	mockResponse400 := adapters.ResponseData{StatusCode: 400, Body: json.RawMessage(`{"seatbid":[{"bid":[{"impid":"1234"}]},{"bid":[{"impid":"1235"}]}]}`)}

	BidderResponse200, err := adapter.MakeBids(&internalRequest, RequestData[0], &mockResponse200)
	if BidderResponse200 != nil && err != nil {
		t.Errorf("actual = %t expected == %v", err, nil)
	}
	BidderResponse203, err := adapter.MakeBids(&internalRequest, RequestData[0], &mockResponse203)
	if BidderResponse203 == nil && err == nil {
		t.Errorf("actual = %t expected != %v", err, nil)
	}
	BidderResponse204, err := adapter.MakeBids(&internalRequest, RequestData[0], &mockResponse204)
	if BidderResponse204 == nil && err != nil {
		t.Errorf("actual = %t expected != %v", err, nil)
	}
	BidderResponse400, err := adapter.MakeBids(&internalRequest, RequestData[0], &mockResponse400)
	if BidderResponse400 == nil && err == nil {
		t.Errorf("actual = %t expected != %v", err, nil)
	}

}
