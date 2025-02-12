package impactify

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
		if err := validator.Validate(openrtb_ext.BidderImpactify, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Impactify params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderImpactify, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"appId": "impactify.io", "format": "screen", "style": "inline"}`,
	`{"appId": "impactify.io", "format": "screen", "style": "impact"}`,
}

var invalidParams = []string{
	`{"appId":"impactify.io"}`,
	`{"appId":"impactify.io", "format": "screen"}`,
	``,
	`null`,
	`true`,
	`[]`,
	`{}`,
}
