package theadx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/theadx.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.theadx

// TestValidParams makes sure that the theadx schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderTheadx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected theadx params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the theadx schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderTheadx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"tagid":321}`,
	`{"tagid":321,"wid":"456"}`,
	`{"tagid":321,"pid":"12345"}`,
	`{"tagid":321,"pname":"plc_mobile_300x250"}`,
	`{"tagid":321,"inv":321,"mname":"pcl1"}`,
	`{"tagid":"123","wid":"456","pid":"12345","pname":"plc_mobile_300x250"}`,
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
