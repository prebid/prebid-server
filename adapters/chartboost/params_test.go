package chartboost

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// This file actually intends to test static/bidder-params/chartboost.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.chartboost

// TestValidParams makes sure that the chartboost schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderChartboost, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected chartboost params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the chartboost schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderChartboost, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"app_id": "5a4b1c2d3e4f5a6b7c8d9e0f", "location_name": "default-banner"}`,
	`{"app_id": "abc", "location_name": "rewarded-video"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`undefined`,
	`0`,
	`{}`,
	`[]`,
	`{"app_id": "5a4b1c2d3e4f5a6b7c8d9e0f"}`,
	`{"location_name": "default-banner"}`,
	`{"app_id": "", "location_name": "default-banner"}`,
	`{"app_id": "5a4b1c2d3e4f5a6b7c8d9e0f", "location_name": ""}`,
	`{"app_id": "", "location_name": ""}`,
	`{"app_id": 0, "location_name": "default-banner"}`,
	`{"app_id": "5a4b1c2d3e4f5a6b7c8d9e0f", "location_name": 0}`,
	`{"app_id": null, "location_name": "default-banner"}`,
	`{"app_id": "5a4b1c2d3e4f5a6b7c8d9e0f", "location_name": null}`,
	`{"app_id": [], "location_name": "default-banner"}`,
	`{"app_id": {}, "location_name": "default-banner"}`,
	`{"appId": "5a4b1c2d3e4f5a6b7c8d9e0f", "locationName": "default-banner"}`,
}
