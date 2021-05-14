package ttx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/33across.json
//
// These also validate the format of the external API: request.imp[i].ext.33across

// TestValidParams makes sure that the 33across schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.Bidder33Across, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected 33Across params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the 33Across schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.Bidder33Across, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"productId": "inview", "siteId": "fakesiteid1"}`,
	`{"productId": "siab", "siteId": "fakesiteid2"}`,
	`{"productId": "inview", "siteId": "foo.ba", "zoneId": "zone1"}`,
}

var invalidParams = []string{
	`{"productId": "inview"}`,
	`{"siteId": "fakesiteid2"}`,
	`{"productId": 123, "siteId": "fakesiteid2"}`,
	`{"productId": "siab", "siteId": 123}`,
	`{"productId": "siab", "siteId": "fakesiteid2", "zoneId": 123}`,
}
