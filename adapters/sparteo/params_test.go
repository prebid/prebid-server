package sparteo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/sparteo.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.sparteo

// TestValidParams verifies that the Sparteo JSON schema accepts all supported parameters.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas: %v", err)
	}

	// Iterate through valid JSON examples.
	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSparteo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected sparteo params: %s\nError: %v", validParam, err)
		}
	}
}

// TestInvalidParams verifies that the Sparteo JSON schema rejects unsupported or incorrectly formatted parameters.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas: %v", err)
	}

	// Iterate through invalid JSON examples.
	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSparteo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	// Minimal valid example: only required field.
	`{"networkId": "net123"}`,
	// Valid with one optional parameter.
	`{"networkId": "net123", "custom1": "reporting-1"}`,
	// Valid with all optional fields.
	`{"networkId": "net123", "custom1": "reporting-1", "custom2": "reporting_2", "custom3": "alpha", "custom4": "beta", "custom5": "gamma"}`,
	// Valid with additional properties (allowed by "additionalProperties": true).
	`{"networkId": "net123", "extraProperty": 42}`,
}

var invalidParams = []string{
	``,     // Empty string.
	`null`, // JSON null.
	`true`, // Boolean is not an object.
	`5`,    // Number is not an object.
	`4.2`,  // Number is not an object.
	`[]`,   // Array is not an object.
	`{}`,   // Missing required field "networkId".
	// Wrong type for "networkId"
	`{"networkId": 123}`,
	// Wrong type for optional parameters.
	`{"networkId": "net123", "custom1": 456}`,
	`{"networkId": "net123", "custom2": true}`,
}
