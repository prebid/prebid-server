package between

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/between.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.between

// TestValidParams makes sure that the between schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBetween, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Between params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Between schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBetween, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"host":"lbs-eu1.ads", "publisher_id": "1"}`,
	`{"host":"lbs-ru1.ads", "publisher_id": "2"}`,
	`{"host":"lbs-us-east1.ads", "publisher_id": "3"}`,
	`{"host":"lbs-asia1.ads", "publisher_id": "4"}`,
}

var invalidParams = []string{
	`{"host":"badhost.ads", "publisher_id": "1"}`,
	`{"host":"lbs-eu1.ads", "publisher_id": 1}`,
	`{"host":"lbs-eu1.ads", "publisher_id": "1""}`,
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
}
