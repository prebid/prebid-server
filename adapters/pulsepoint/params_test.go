package pulsepoint

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
		if err := validator.Validate(openrtb_ext.BidderPulsepoint, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected pulsepoint params: %s \n Error: %s", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the pubmatic schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderPulsepoint, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected pulsepoint params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"cp":1000, "ct": 2000}`,
	`{"cp":1001, "ct": 2001}`,
	`{"cp":"1000", "ct": "2000"}`,
	`{"cp":"1000", "ct": 2000}`,
	`{"cp":1000, "ct": "2000"}`,
	`{"cp":1001, "ct": 2001, "cf": "1x1"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"cp":"1000"}`,
	`{"ct":"1000"}`,
	`{"cp":1000}`,
	`{"ct":1000}`,
}
