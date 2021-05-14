package yieldlab

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/yieldlab.json
//
// These also validate the format of the external API: request.imp[i].ext.yieldlab

// TestValidParams makes sure that the yieldlab schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderYieldlab, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected yieldlab params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the yieldlab schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderYieldlab, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"adslotId": "123","supplyId":"23456","adSize":"100x100"}`,
	`{"adslotId": "123","supplyId":"23456","adSize":"100x100","extId":"asdf"}`,
	`{"adslotId": "123","supplyId":"23456","adSize":"100x100","extId":"asdf","targeting":{"a":"b"}}`,
	`{"adslotId": "123","supplyId":"23456","adSize":"100x100","targeting":{"a":"b"}}`,
	`{"adslotId": "123","supplyId":"23456","adSize":"100x100","targeting":{"a":"b"}}`,
}

var invalidParams = []string{
	`{"supplyId":"23456","adSize":"100x100"}`,
	`{"adslotId": "123","adSize":"100x100","extId":"asdf"}`,
	`{"adslotId": "123","supplyId":"23456","extId":"asdf","targeting":{"a":"b"}}`,
	`{"adslotId": "123","supplyId":"23456"}`,
	`{"adSize":"100x100","supplyId":"23456"}`,
	`{"adslotId": "123","adSize":"100x100"}`,
	`{"supplyId":"23456"}`,
	`{"adslotId": "123"}`,
	`{}`,
	`[]`,
	`{"a":"b"}`,
	`null`,
}
