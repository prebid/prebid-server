package lockerdome

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file tests static/bidder-params/lockerdome.json
// and validates the format of the external API: request.imp[i].ext.lockerdome

// TestValidParams makes sure that the LockerDome schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderLockerDome, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected LockerDome params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the LockerDome schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderLockerDome, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"adUnitId": "LD1234567890"}`, // adUnitId can start with "LD"
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`true`,
	`1`,
	`1.5`,
	`[]`,
	`{}`,
	`{"adUnitId": true}`,
	`{"adUnitId": 123456789}`, // adUnitId can't be a number
}
