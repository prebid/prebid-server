package revx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to load JSON schemas: %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderRevX, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected valid RevX params: %s\nError: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to load JSON schemas: %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRevX, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema accepted invalid RevX params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pubname": "publisher123"}`,
}

var invalidParams = []string{
	`{}`,
	`{ "pubname": ""}`,
}
