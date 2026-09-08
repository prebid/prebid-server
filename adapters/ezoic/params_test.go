package ezoic

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// This file intends to test static/bidder-params/ezoic.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.ezoic

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderEzoic, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected ezoic params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderEzoic, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{}`,
	`{"placementId": "ezoic-placement-1"}`,
}

var invalidParams = []string{
	`null`,
	`true`,
	`[]`,
	`"placementId"`,
	`{"placementId": 12345}`,
	`{"placementId": true}`,
}
