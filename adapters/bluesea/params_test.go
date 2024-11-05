package bluesea

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// TestValidParams makes sure that the bluesea schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBluesea, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected bluesea params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the bluesea schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBluesea, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pubid": "1001", "token": "t35w089r1n92k946"}`,
}

var invalidParams = []string{
	`{1001`,
	`{"pubid": "1001"}`,
	`{"pubid": "1001", "Token": "invalid-token"}`,
	`{"Pubid": "1001", "token": "invalid-token"}`,
	`{"pubid": "abc", "token": "t35w089r1n92k946"}`,
	`{"pubid": "1001", "token": "test-token"}`,
}
