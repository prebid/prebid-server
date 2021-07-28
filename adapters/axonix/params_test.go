package axonix

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/axonix.json
//
// These also validate the format of the external API: request.imp[i].ext.axonix

// TestValidParams makes sure that the Axonix schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAxonix, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Axonix params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Axonix schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAxonix, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"host": "axonix.com", "supplyId": "test-supply"}`,
	`{"host": "test.com", "supplyId": "test"}`,
}

var invalidParams = []string{
	`{"host": "10", "supplyId": 100}`,
	`{"host": true, "supplyId": false}`,
	`{"host": "test", "supplyId": true}`,
	`{"host": "test", "supplyId": 123}`,
	`{"host": true, "supplyId": "test"}`,
	`{"host": 123, "supplyId": "test}`,
	`{"host": ""}`,
	``,
	`null`,
	`true`,
	`9`,
	`1.2`,
	`[]`,
	`{}`,
}
