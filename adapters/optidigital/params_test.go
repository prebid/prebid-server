package optidigital

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/optidigital.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.optidigital

// TestValidParams makes sure that the optidigital schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOptidigital, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected optidigital params: %s \n Error: %s", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the optidigital schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOptidigital, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId":"p123","placementId":"foo-placement"}`,
	`{"publisherId":"p123","placementId":"foo-placement","divId":""}`,
	`{"publisherId":"p123","placementId":"foo-placement","divId":"foo-div"}`,
	`{"publisherId":"p123","placementId":"foo-placement","divId":"foo-div","pageTemplate":""}`,
	`{"publisherId":"p123","placementId":"foo-placement","divId":"foo-div","pageTemplate":"foo-page-template"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"publisherId":""}`,
	`{"publisherId":"p123","placementId":""}`,
	`{"publisherId":"","placementId":"foo-placement"}`,
	`{"publisherId":"p","placementId":"foo-placement"}`,
}
