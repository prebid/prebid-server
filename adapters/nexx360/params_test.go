package nexx360

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/nexx360.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.nexx360

// TestValidParams makes sure that the Nexx360 schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderNexx360, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Nexx360 params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Nexx360 schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderNexx360, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"tagId": "testnexx"}`,
	`{"placement": "testnexx"}`,
	`{"tagId": "testnexx", "placement": "testnexx"}`,
}

var invalidParams = []string{
	`{"productId": "inview"}`,
	`{"tagId": "" }`,
	`{"placement": "" }`,
	`{"tagId": "testnexx", "placement": "" }`,
	`{"tagId": "", "placement": "testnexx"}`,
	`{}`,
}
