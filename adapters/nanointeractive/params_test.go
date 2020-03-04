package nanointeractive

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/nanointeractive.json
//
// These also validate the format of the external API: request.imp[i].ext.nanointeracive

// TestValidParams makes sure that the NanoInteractive schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderNanoInteractive, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected NanoInteractive params: %s", validParam)
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
		if err := validator.Validate(openrtb_ext.BidderNanoInteractive, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pid": "dafad098"}`,
	`{"pid":"dfasfda","nq":["search query"]}`,
	`{"pid":"dfasfda","nq":["search query"],"subId":"any string value","category":"any string value"}`,
}

var invalidParams = []string{
	`{"pid":123}`,
	`{"pid":"12323","nq":"search query not an array"}`,
	`{"pid":"12323","category":1}`,
	`{"pid":"12323","subId":23}`,
	``,
	`null`,
	`true`,
	`9`,
	`1.2`,
	`[]`,
	`{}`,
	`placementId`,
	`zone`,
	`zoneId`,
}
