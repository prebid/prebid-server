package invibes

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderInvibes, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected invibes params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderInvibes, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId":"123", "domainId": 1}`,
	`{"placementId":"123", "domainId": 2, "debug":{}}`,
	`{"placementId":"123", "domainId": 0, "debug":{"testLog":true}}`,
	`{"placementId":"123", "domainId": 0, "debug":{"testBvid":"1234"}}`,
}

var invalidParams = []string{
	``,
	`{"placementId":123}`,
	`{"placementId":"123", "debug":"malformed"}`,
	`{"placementId":"123", "domainId": "abc"}`,
	`{"placementId":"123", "domainId": 2, "debug":{"testLog":1}}`,
	`null`,
	`true`,
	`0`,
	`abc`,
	`[]`,
	`{}`,
}
