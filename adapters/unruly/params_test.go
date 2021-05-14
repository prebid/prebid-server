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
	`{"uuid": "123", "siteid": "abc"}`,
}

var invalidParams = []string{
	`{}`,                               // Missing required uuid + siteId
	`{"siteId": "123"}`,                // Missing required uuid
	`{"uuid": "123"}`,                  // Missing required siteId
	`{"uuid": 123, "siteid": "abc"}`,   // Wrong uuid data type
	`{"uuid": "123", "siteid": 123}`,   // Wrong siteid data type
	`{"UUID": "123", "SiteId": "abc"}`, // Invalid capitalization (json is case sensitive)
}
