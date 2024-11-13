package gamma

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/gamma.json

// TestValidParams makes sure that the Gamma schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderGamma, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Gamma params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Gamma schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderGamma, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"id": "1397808490", "wid": "1513150517", "zid": "1513151405"}`,
	`{"id": "1397808490", "wid": "1513150517", "zid": "1513151405", "app_id": "123456789"}`,
}

var invalidParams = []string{
	`{"id": 100}`,
	`{"wid": false}`,
	`{"zid": true}`,
	`{"pub_id": 123, "headerbidding": true}`,
	`{"pub_id": "1"}`,
	``,
	`null`,
	`true`,
	`9`,
	`1.2`,
	`[]`,
	`{}`,
}
