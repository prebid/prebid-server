package tpmn

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// TestValidParams makes sure that the tpmn schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderTpmn, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected TPMN params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the tpmn schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderTpmn, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"inventoryId": "00000001", "publisherId": "publisherId"}`,
}

var invalidParams = []string{
	`{"inventoryId": "00000001"}`,
	`{"inventoryId": 123}`,
	`{"inventoryid": 123}`,
	`{"inventoryId": 1, "publisherId": "publisherId"}`,
	`{"inventoryid": "00000001", "publisherId": "publisherId"}`,
	`{"inventoryId": "00000001", "publisherid": "publisherId"}`,
	`{"inventoryid": "00000001", "publisherid": "publisherId"}`,
}
