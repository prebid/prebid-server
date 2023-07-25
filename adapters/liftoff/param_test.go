package liftoff

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

var validParams = []string{
	`{"bid_token": "SomeAccessToken", "app_store_id": "12345", "placement_reference_id": "12345"}`,
}

var invalidParams = []string{
	`{"bid_token": ""}`,
	`{"bid_token": 42}`,
	`{"bid_token": null}`,
	`{}`,
	// app_store_id & placement_reference_id
	`{"app_store_id": "12345", "placement_reference_id": "12345"}`,
	`{"bid_token": "SomeAccessToken", "app_store_id": "12345"}`,
	`{"bid_token": "SomeAccessToken", "placement_reference_id": "12345"}`,
	`{"bid_token": "SomeAccessToken", "app_store_id": 12345, "placement_reference_id": 12345}`,
	`{"bid_token": "SomeAccessToken", "app_store_id": null, "placement_reference_id": null}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderLiftoff, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderLiftoff, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}
