package adagio

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/pubmatic.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.pubmatic

// TestValidParams makes sure that the adagio schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdagio, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adagio params: %s \n Error: %s", validParam, err)
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
		if err := validator.Validate(openrtb_ext.BidderAdagio, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"organizationId":"1000","placement":"some-placement"}`,
	`{"organizationId":"1000","placement":"some-placement"}`,
	`{"organizationId":"1000","placement":"some-placement","pagetype":"some-pagetype"}`,
	`{"organizationId":"1000","placement":"some-placement","category":"some-category"}`,
	`{"organizationId":"1000","placement":"some-placement","pagetype":"some-pagetype","category":"some-category"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"organizationId":"1000"}`,
	`{"placement":"some-placement"}`,
	`{"category":"some-category"}`,
	`{"pagetype":"some-pagetype"}`,
	`{"organizationId":"1000"}`,
	`{"organizationId":1000,"placement":"some-placement"}`,
	`{"organizationId":"1000","placement":"this-is-a-very-very-long-placement"}`,
	`{"organizationId":"1000","placement":123456}`,
	`{"organizationId":"1000","placement":"some-placement","pagetype":123456}`,
	`{"organizationId":"1000","placement":"some-placement","pagetype":"this-is-a-very-very-long-pagetype"}`,
	`{"organizationId":"1000","placement":"some-placement","category":123456}`,
	`{"organizationId":"1000","placement":"some-placement","category":"this-is-a-very-very-long-category"}`,
}
