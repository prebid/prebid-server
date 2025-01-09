package kargo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderKargo, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderKargo, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"placementId": ""}`,
	`{"placementId": "11523"}`,
	`{"adSlotID": ""}`,
	`{"adSlotID": "11523"}`,
}

var invalidParams = []string{
	`{"placementId": 42}`,
	`{"placementId": }`,
	`{"placementID": "32321"}`,
	`{"adSlotID": 42}`,
	`{"adSlotID": }`,
	`{"adSlotId": "32321"}`,
	`{"id": }`,
	`{}`,
	`{"placementId": "11523", "adSlotID": "12345"}`, // Can't include both
}
