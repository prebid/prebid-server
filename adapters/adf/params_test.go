package adf

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/adf.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.adf

// TestValidParams makes sure that the adform schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdf, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adform params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the adform schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdf, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"mid":123}`,
	`{"mid":"123"}`,
	`{"inv":321,"mname":"pcl1"}`,
	`{"inv":321,"mname":"12345"}`,
	`{"mid":123,"inv":321,"mname":"pcl1"}`,
	`{"mid":"123","inv":321,"mname":"pcl1"}`,
	`{"mid":"123","priceType":"gross"}`,
	`{"mid":"123","priceType":"net"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"notmid":"123"}`,
	`{"mid":"placementID"}`,
	`{"inv":321,"mname":12345}`,
	`{"inv":321}`,
	`{"inv":"321"}`,
	`{"mname":"12345"}`,
	`{"mid":"123","priceType":"GROSS"}`,
}
