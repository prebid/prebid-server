package logan

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// TestValidParams makes sure that the logan schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderLogan, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected logan params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the logan schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderLogan, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId": "6"}`,
}

var invalidParams = []string{
	`{"id": "123"}`,
	`{"placementID": "123"}`,
	`{"PlacementID": "123"}`,
	`{"placementId": 16}`,
	`{"placementId": ""}`,
}
