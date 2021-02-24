package adyoulike

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/adyoulike.json
//
// These also validate the format of the external API: request.imp[i].ext.adyoulike

// TestValidParams makes sure that the adyoulike schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdyoulike, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adyoulike params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the adyoulike schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdyoulike, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placement":"123"}`,
	`{"placement":"123","campaign":"456"}`,
	`{"placement":"123","campaign":"456","track":"789"}`,
	`{"placement":"123","campaign":"456","track":"789","creative":"ABC"}`,
	`{"placement":"123","campaign":"456","track":"789","creative":"ABC","source":"SSP"}`,
	`{"placement":"123","campaign":"456","track":"789","creative":"ABC","source":"SSP","debug":"info"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"invalid":"123"}`,
	`{"placement":123}`,
	`{"placement":"123","campaign":123}`,
}
