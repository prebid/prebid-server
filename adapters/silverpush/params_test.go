package silverpush

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file intends to test static/bidder-params/silverpush.json

// These also validate the format of the external API: request.imp[i].bidRequestExt.silverpush

// TestValidParams makes sure that the Smaato schema accepts all imp.bidRequestExt fields which Smaato supports.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSilverPush, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected silverpush params: %s \n Error: %s", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the Smaato schema rejects all the imp.bidRequestExt fields which are not support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSilverPush, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId":"test-id-1234-silverpush","bidfloor": 0.05}`,
	`{"publisherId":"test123","bidfloor": 0.05}`,
	`{"publisherId":"testSIlverpush","bidfloor": 0.05}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"bidfloor": "1123581321"}`,
	`{"publisherId":false}`,
	`{"bidfloor":false}`,
	`{"publisherId":0,"bidfloor": 1123581321}`,
	`{"publisherId":false,"bidfloor": true}`,
	`{"instl": 0}`,
	`{"secure": 0}`,
	`{"bidfloor": "1123581321"}`,
	`{"publisherId":{}}`,
}
