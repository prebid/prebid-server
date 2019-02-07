package beachfront

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBeachfront, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected beachfront params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBeachfront, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"appId":"11bc5dd5-7421-4dd8-c926-40fa653bec76", "bidfloor":0.01}`,
}

var invalidParams = []string{
	`{"appId":1176, "bidfloor":0.01}`,
	`{"appId":"11bc5dd5-7421-4dd8-c926-40fa653bec76"}`,
}
