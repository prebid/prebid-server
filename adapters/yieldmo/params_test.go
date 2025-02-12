package yieldmo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/yieldmo.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.yieldmo

// TestValidParams makes sure that the Yieldmo schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderYieldmo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Yieldmo params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Yieldmo schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderYieldmo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId": "123"}`,
}

var invalidParams = []string{
	`{"invalid_param": "123"}`,
}
