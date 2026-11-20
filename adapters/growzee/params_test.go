package growzee

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
	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderGrowzee, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected growzee params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}
	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderGrowzee, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisherId": "1d5c3b2a-0f4e-4a6b-9c8d-7e6f5a4b3c2d"}`,
	`{"publisherId": "9F8E7D6C-5B4A-4321-8FED-CBA987654321"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`[]`,
	`{}`,
	`{"publisherId": ""}`,
	`{"publisherId": 123}`,
	`{"publisherId": "not-a-uuid"}`,
	`{"publisherId": "1d5c3b2a0f4e4a6b9c8d7e6f5a4b3c2d"}`,
}
