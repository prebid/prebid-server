package trafficgate

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

// TestValidParams makes sure that the trafficgate schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderTrafficGate, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected trafficgate params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the trafficgate schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderTrafficGate, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId": "11", "host": "example"}`,
}

var invalidParams = []string{
	`{"id": "456"}`,
	`{"placementid": "3456", "host": ""}`,
	`{"placement_id": 346, "host": ""}`,
	`{"placementID": "", "HOST": "example"}`,
	`{"placementId": 234}`,
}
