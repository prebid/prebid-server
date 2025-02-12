package colossus

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// TestValidParams makes sure that the colossus schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderColossus, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected colossus params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the colossus schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderColossus, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"TagID": "61317"}`,
	`{"groupId": "123"}`,
}

var invalidParams = []string{
	`{"id": "123"}`,
	`{"tagid": "123"}`,
	`{"TagID": 16}`,
	`{"TagID": "123", "groupId": "456"}`,
}
