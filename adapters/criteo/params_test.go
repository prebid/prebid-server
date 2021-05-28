package criteo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/criteo.json
//
// These also validate the format of the external API: request.imp[i].ext.criteo

// TestValidParams makes sure that the criteo schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderCriteo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected criteo params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the criteo schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderCriteo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"zoneid": 123456}`,
	`{"networkid": 78910}`,
	`{"zoneid": 123456, "networkid": 78910}`,
	`{"zoneid": 0, "networkid": 0}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"zoneid": -123}`,
	`{"networkid": -321}`,
	`{"zoneid": -123, "networkid": -321}`,
	`{"zoneid": -1}`,
	`{"networkid": -1}`,
	`{"zoneid": -1, "networkid": -1}`,
}
