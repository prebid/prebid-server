package madvertise

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/Madvertise.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.madvertise

// TestValidParams makes sure that the Madvertise schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderMadvertise, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Madvertise params: %s \n Error: %s", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the Madvertise schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderMadvertise, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"zoneId":"/1111111/banner"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`[]`,
	`{}`,
	`{"zoneId":""}`,
	`{"zoneId":/1111111}`,
	`{"zoneId":/1111"}`,
}
