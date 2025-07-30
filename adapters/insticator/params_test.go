package insticator

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/Insticator.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.Insticator

// TestValidParams makes sure that the Insticator schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderInsticator, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Insticator params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Insticator schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderInsticator, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId": "inview", "adUnitId": "fakesiteid1"}`,
	`{"publisherId": "siab", "adUnitId": "fakesiteid2"}`,
	`{"publisherId": "inview", "adUnitId": "foo.ba"}`,
}

var invalidParams = []string{
	`{"publisherId": "inview"}`,                        // Missing adUnitId
	`{"publisherId": 123, "adUnitId": "fakesiteid2"}`,  // publisherId should be a string
	`{"adUnitId": "fakesiteid3"}`,                      // Missing publisherId
	`{"publisherId": "inview", "adUnitId": 456}`,       // adUnitId should be a string
	`{"publisherId": null, "adUnitId": "fakesiteid5"}`, // Null publisherId
	`{"publisherId": "inview", "adUnitId": null}`,      // Null adUnitId
	`{"publisherId": true, "adUnitId": "fakesiteid6"}`, // publisherId should be a string, got boolean
	`{"publisherId": "inview", "adUnitId": [1, 2, 3]}`, // adUnitId should be a string, got array
}
