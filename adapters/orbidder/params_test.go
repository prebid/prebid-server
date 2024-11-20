package orbidder

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/orbidder.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.orbidder

// TestValidParams makes sure that the orbidder schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOrbidder, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected orbidder params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the orbidder schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOrbidder, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId":"123","accountId":"orbidder-test"}`,
	`{"placementId":"123","accountId":"orbidder-test","bidfloor":0.5}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"placement_id":"123"}`,
	`{"placementId":123}`,
	`{"placementId":"123"}`,
	`{"account_id":"orbidder-test"}`,
	`{"accountId":123}`,
	`{"accountId":"orbidder-test"}`,
	`{"placementId":123,"account_id":"orbidder-test"}`,
	`{"placementId":"123","account_id":123}`,
	`{"placementId":"123","accountId":"orbidder-test","bidfloor":"0.5"}`,
	`{"placementId":"123","bidfloor":"0.5"}`,
	`{"accountId":"orbidder-test","bidfloor":"0.5"}`,
}
