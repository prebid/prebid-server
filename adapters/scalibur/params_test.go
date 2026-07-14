package scalibur

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderScalibur, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected scalibur params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderScalibur, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{}`,
	`{"placementId":"p123"}`,
	`{"placementId":"p123", "bidfloor": 1.5}`,
	`{"placementId":"p123", "bidfloor": 1.5, "bidfloorcur": "USD"}`,
	`{"host":"eu.scalibur.io"}`,
	`{"host":"host:8080"}`,
	`{"bidfloor": 1.5}`,
	`{"placementId":"p123", "customKey": "customValue"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{"placementId": 123}`,
	`{"host":"evil.com/path?x=1"}`,
	`{"host":"https://eu.scalibur.io"}`,
}
