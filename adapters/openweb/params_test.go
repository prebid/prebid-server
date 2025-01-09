package openweb

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/openweb.json
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.openweb
// TestValidParams makes sure that the openweb schema accepts all imp.ext fields which we intend to support.

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOpenWeb, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected openweb params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the openweb schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOpenWeb, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"aid":123,"placementId":"1234"}`,
	`{"org":"123","placementId":"1234"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"org":123}`,
	`{"org":0}`,
	`{"org":"123","placementId":123}`,
	`{"org":123, "placementId":"123"}`,
	`{"aid":123}`,
	`{"aid":"123","placementId":"123"}`,
}
