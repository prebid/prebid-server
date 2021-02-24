package jixie

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderJixie, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected jixie params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderJixie, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"unit": "1000008-AA77BB88CC" }`,
	`{"unit": "1000008-AA77BB88CC", "accountid": "9988776655", "jxprop1": "somethingimportant" }`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`true`,
	`{}`,
	`{"unit":12345678}`,
	`{"Unit":"12345678"}`,
	`{"Unit": 12345678}`,
	`{"AdUnit": "1"}`,
	`{"adUnit": 1}`,
	`{"unit": ""}`,
	`{"unit": "12345678901234567"}`,
	`{"unit":"1000008-AA77BB88CC", "accountid",  "jxprop1": "somethingimportant" }`,
	`{"unit":"1000008-AA77BB88CC", malformed, }`,
}
