package driftpixel

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

var validParams = []string{
	`{"env":"driftpixel-stage", "pid":"123456"}`,
	`{"pid":"123456"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderDriftPixel, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected driftpixel params: %s", validParam)
		}
	}
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"some": "param"}`,
	`{"env":"driftpixel-stage"}`,
	`{"othervalue":"Lorem ipsum"}`,
	`{"env":"driftpixel-stage", pid:""}`,
	`{pid:101010}`,
	`{pid:"valid-pid", env: 0}`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderDriftPixel, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
