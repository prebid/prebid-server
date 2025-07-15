package risemediatech

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the JSON schema: %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderRiseMediaTech, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s — error: %v", p, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the JSON schema: %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRiseMediaTech, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"bidfloor": 0.01}`,
	`{"bidfloor": 2.5, "testMode": 1}`,
}

var invalidParams = []string{
	`{"bidfloor": "1.2"}`,
	`{"testMode": "yes"}`,
	`{"bidfloor": -5}`,
	`{"testMode": 9999}`,
}
