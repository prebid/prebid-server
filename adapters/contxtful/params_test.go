package contxtful

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema: %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderContxtful, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected valid params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema: %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderContxtful, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId": "12345", "customerId": "customer-123"}`,
	`{"placementId": "placement-123", "customerId": "test-customer"}`,
}

var invalidParams = []string{
	`{}`,
	`{"placementId": "12345"}`,
	`{"customerId": "customer-123"}`,
	`{"placementId": 12345, "customerId": "customer-123"}`,
	`{"placementId": "12345", "customerId": 123}`,
}
