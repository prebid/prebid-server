package seedingAlliance

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
		if err := validator.Validate(openrtb_ext.BidderSeedingAlliance, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderSeedingAlliance, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"adUnitId": "1234"}`,
	`{"adUnitId": "AB12"}`,
	`{"adUnitId": "1234", "seatId": "1234"}`,
	`{"adUnitId": "AB12", "seatId": "AB12"}`,
}

var invalidParams = []string{
	`{"adUnitId": 42}`,
	`{"adUnitId": "1234", "seatId": 42}`,
	`{"adUnitId": 1234, "seatId": "42"}`,
}
