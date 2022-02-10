package unruly

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderUnruly, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Unruly params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderUnruly, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"siteid": 123}`,
}

var invalidParams = []string{
	`{}`,                // Missing required siteId
	`{"siteid": "123"}`, // Invalid siteId type
	`{"uuid": "123"}`,   // Missing required siteId
	`{"SiteId": "abc"}`, // Invalid capitalization (json is case sensitive)
	`{"siteId": []}`,    // Invalid siteid data type
}
