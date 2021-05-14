package appnexus

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/appnexus.json
//
// These also validate the format of the external API: request.imp[i].ext.appnexus

// TestValidParams makes sure that the appnexus schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAppnexus, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected appnexus params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the appnexus schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAppnexus, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placement_id":123}`,
	`{"placementId":123,"position":"above"}`,
	`{"placement_id":123,"position":"below"}`,
	`{"member":"123","inv_code":"456"}`,
	`{"placementId":123, "keywords":[{"key":"foo","value":["bar"]}]}`,
	`{"placement_id":123, "keywords":[{"key":"foo","value":["bar", "baz"]}]}`,
	`{"placement_id":123, "keywords":[{"key":"foo"}]}`,
	`{"placement_id":123, "use_pmt_rule": true, "private_sizes": [{"w": 300, "h":250}]}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"placement_id":"123"}`,
	`{"placement_id":123, "placementId":123}`,
	`{"member":"123"}`,
	`{"member":"123","invCode":45}`,
	`{"placementId":"123","member":"123","invCode":45}`,
	`{"placement_id":123, "position":"left"}`,
	`{"placement_id":123, "position":"left"}`,
	`{"placement_id":123, "reserve":"45"}`,
	`{"placement_id":123, "keywords":[]}`,
	`{"placement_id":123, "keywords":["foo"]}`,
	`{"placementId":123, "keywords":[{"key":"foo","value":[]}]}`,
	`{"placementId":123, "keywords":[{"value":["bar"]}]}`,
	`{"placement_id":123, "use_pmt_rule": "true"}`,
	`{"placement_id":123, "private_sizes": [[300,250]]}`,
	`{"placement_id":123, "private_sizes": [{"w": "300", "h": "250"}]}`,
}
