package marsmedia

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/marsmedia.json
//
// These also validate the format of the external API: request.imp[i].ext.marsmedia

// TestValidParams makes sure that the Marsmedia schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderMarsmedia, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Marsmedia params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Marsmedia schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderMarsmedia, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"zone": "9999"}`,
}

var invalidParams = []string{
	`{"zone": 100}`,
	`{"headerbidding": false}`,
	`{"zone": true}`,
	`{"zoneId": 123, "headerbidding": true}`,
	`{"zoneID": "1"}`,
	``,
	`null`,
	`true`,
	`9`,
	`1.2`,
	`[]`,
	`{}`,
}
