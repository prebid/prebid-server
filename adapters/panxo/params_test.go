package panxo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the JSON schema: %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderPanxo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected valid params: %s\nError: %v", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the JSON schema: %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderPanxo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema should have rejected invalid params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"propertyKey": "abc123def456"}`,
	`{"propertyKey": "a"}`,
}

var invalidParams = []string{
	`{}`,
	`{"propertyKey": ""}`,
	`{"propertyKey": 123}`,
	`null`,
}
