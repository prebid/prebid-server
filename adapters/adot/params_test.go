package adot

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file actually intends to test static/bidder-params/adot.json
//
// These also validate the format of the external API: request.imp[i].ext.adot

// TestValidParams makes sure that the adot schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdot, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adot params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdot, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{}`,
	`{"placementId": "test-114"}`,
	`{"placementId": "test-113", "parallax": true}`,
	`{"placementId": "test-113", "parallax": false}`,
}

var invalidParams = []string{
	`{"parallax": 1}`,
	`{"placementId": 135123}`,
	`{"placementId": 113, "parallax": 1}`,
	`{"placementId": 142, "parallax": true}`,
	`{"placementId": "test-114", "parallax": 1}`,
}
