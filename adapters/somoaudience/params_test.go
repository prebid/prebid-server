package somoaudience

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/somoaudience.json
//
// These also validate the format of the external API: request.imp[i].ext.somoaudience

// TestValidParams makes sure that the somoaudience schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSomoaudience, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected somoaudience params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the somoaudience schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSomoaudience, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placement_hash":"22a58cfb0c9b656bff713d1236e930e8"}`,
	`{"placement_hash":"22a58cfb0c9b656bff713d1236e930e8", "bid_floor": 1.05}`,
}

var invalidParams = []string{
	`{"placement_hash": 323423}`,
	`{"tag_id":"234234"}`,
	`{"placement_hash":"22a58cfb0c9b656bff713d1236e930e8", "bid_floor": "423s"}`,
	`{"placement_hash":"22a58cfb0c9b656bff713d1236e930e8", "bid_floor": -1}`,
}
