package lemmadigital

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// Tests for static/bidder-params/lemmadigital.json

// Tests whether the schema supports the intended params.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schema. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderLemmadigital, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected params: %s \n Error: %s", validParam, err)
		}
	}
}

// Tests whether the schema rejects unsupported imp.ext fields.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schema. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderLemmadigital, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed invalid/unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pid":1, "aid": 1}`,
	`{"pid":2147483647, "aid": 2147483647}`,
}

var invalidParams = []string{
	``,
	`null`,
	`false`,
	`0`,
	`0.0`,
	`[]`,
	`{}`,
	`{"pid":1}`,
	`{"aid":1}`,
	`{"pid":"1","aid":1}`,
	`{"pid":1.0,"aid":"1"}`,
	`{"pid":"1","aid":"1"}`,
	`{"pid":false,"aid":true}`,
}
