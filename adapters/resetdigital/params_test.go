package resetdigital

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
		if err := validator.Validate(openrtb_ext.BidderResetDigital, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected ResetDigital params: %s \n Error: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderResetDigital, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected ResetDigital params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placement_id":"1000"}`,
	`{"placement_id":"0"}`,
	`{"placement_id":"abc"}`,
	`{"placement_id":"123abc"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{"placement_id":}`,
	`{"placement_id":""}`,
	`{"placement_id": 123}`,
	`{"cp":"1000"}`,
}
