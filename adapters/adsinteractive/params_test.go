package adsinteractive

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/adsinteractive.json

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdsinteractive, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adsinteractive params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Adsinteractive schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdsinteractive, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"adUnit": "Example_ad_unit_1"}`,
}

var invalidParams = []string{
	`{"adunit": "Example_ad_unit_1"}`,
	`{"AdUnit": "Example_ad_unit_1"}`,
	`{"ad_unit": Example_ad_unit_1}`,
	``,
	`null`,
	`true`,
	`[]`,
	`{}`,
	`nil`,
	`53`,
	`9.1`,
}
