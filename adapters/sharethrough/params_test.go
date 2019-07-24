package sharethrough

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
		if err := validator.Validate(openrtb_ext.BidderSharethrough, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Sharethrough params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSharethrough, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pkey": "123"}`,
	`{"pkey": "123", "iframe": true}`,
	`{"pkey": "abc", "iframe": false}`,
	`{"pkey": "abc123", "iframe": true, "iframeSize": [20, 20]}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"pkey": 123}`,
	`{"iframe": 123}`,
	`{"iframeSize": [20, 20]}`,
	`{"pkey": 123, "iframe": 123}`,
	`{"pkey": 123, "iframe": true, "iframeSize": [20]}`,
	`{"pkey": 123, "iframe": true, "iframeSize": []}`,
	`{"pkey": 123, "iframe": true, "iframeSize": 123}`,
}
