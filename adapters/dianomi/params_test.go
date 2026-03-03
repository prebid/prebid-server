package dianomi

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/dianomi.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.dianomi

// TestValidParams makes sure that the dianomi schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderDianomi, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected dianomi params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the dianomi schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderDianomi, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"smartadId":123}`,
	`{"smartadId":"123"}`,
	`{"smartadId":"123","priceType":"gross"}`,
	`{"smartadId":"123","priceType":"net"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"notsmartadId":"123"}`,
	`{"smartadID":"smartadId"}`,
	`{"SmartadId":"123","priceType":"GROSS"}`,
}
