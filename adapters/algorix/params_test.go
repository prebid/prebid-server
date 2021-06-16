package algorix

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAlgorix, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected algoirx params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the algorix schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAlgorix, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"sid": "11233", "token": "sin"}`,
	`{"sid": "11244", "token": "iad"}`,
}

var invalidParams = []string{
	`{"sid": "11233"}`,
	`{"token": "aaa"}`,
	`{"sid": 123, "token": "sin"}`,
	`{"sid": "", "token": "iad"}`,
	`{"sid": "11233", "token": ""}`,
}
