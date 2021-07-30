package tappx

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderTappx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected tappx params: %s \n Error: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderTappx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"tappxkey":"pub-12345-android-9876", "endpoint":"ZZ1INTERNALTEST149147915", "host":"test.tappx.com"}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint":"ZZ1INTERNALTEST149147915", "host":"test.tappx.com", "bidfloor":0.5}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint":"ZZ1INTERNALTEST149147915", "host":"test.tappx.com", "bidfloor":0.5, "mktag":"txmk-xxxxx-xxx-xxxx"}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint":"ZZ1INTERNALTEST149147915", "host":"test.tappx.com", "bidfloor":0.5, "bcid":["123"]}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint":"ZZ1INTERNALTEST149147915", "host":"test.tappx.com", "bidfloor":0.5, "bcrid":["245"]}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint":"ZZ1INTERNALTEST149147915", "host":"test.tappx.com", "bidfloor":0.5, "bcrid":["245", "321"]}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint":"ZZ1INTERNALTEST149147915", "host":"test.tappx.com", "bidfloor":0.5, "bcid":["123", "654"], "bcrid":["245", "321"]}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"tappxkey":""}`,
	`{"tappxkey":"pub-12345-android-9876"}`,
	`{"endpoint":""}`,
	`{"endpoint":"ZZ1INTERNALTEST149147915"}`,
	`{"host":""}`,
	`{"host": 1}`,
	`{"host": 1.2}`,
	`{"host": null}`,
	`{"host": true}`,
	`{"tappxkey": 1, "endpoint":"ZZ1INTERNALTEST149147915"}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint": 1}`,
	`{"tappxkey": 1, "endpoint": 1}`,
	`{"tappxkey": 1, "endpoint":"ZZ1INTERNALTEST149147915", "host":""}`,
	`{"tappxkey":"pub-12345-android-9876", "endpoint": 1, "host":""}`,
	`{"tappxkey": 1, "endpoint": 1, "host": 123}`,
	`{"tappxkey": "1", "endpoint": 1}`,
	`{"tappxkey": "1", "endpoint": "ZZ1INTERNALTEST149147915", "host":[]]}`,
	`{"tappxkey": "1", "endpoint": 1, "host":"host"}`,
	`{"tappxkey": "1", "endpoint": "1", "host":"host", "mktag":1}`,
	`{"tappxkey": "1", "endpoint": "1", "host":"host", "mktag":[1,2]}`,
	`{"tappxkey": "1", "endpoint": "1", "host":"host", "bcid":""}`,
	`{"tappxkey": "1", "endpoint": "1", "host":"host", "bcid":"123", bcrid: ["123"]}`,
	`{"tappxkey": "1", "endpoint": "1", "host":"host", "bcid":["123"], bcrid: 123}`,
	`{"tappxkey": "1", "endpoint": "1", "host":"host", "bcid":["123"], bcrid: [123]}`,
	`{"tappxkey": "1", "endpoint": "1", "host":"host", "bcid":[123], bcrid: ["123"]}`,
}
