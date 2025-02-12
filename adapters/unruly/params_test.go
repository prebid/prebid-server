package unruly

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
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
	`{"siteId": 123}`,
}

var invalidParams = []string{
	`{}`,                // Missing required siteId
	`{"siteid": "123"}`, // Invalid siteid type
	`{"siteId": "123"}`, // Invalid siteId type (string)
	`{"uuid": "123"}`,   // Missing required siteId
	`{"SiteId": "abc"}`, // Invalid capitalization (json is case sensitive)
	`{"Siteid": 123}`,   // Invalid capitalization (json is case sensitive)
	`{"siteid": []}`,    // Invalid siteid data type
	`{"siteId": []}`,    // Invalid siteid data type
}
