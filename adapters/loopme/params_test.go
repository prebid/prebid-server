package loopme

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/loopme.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.loopme

// TestValidParams makes sure that the loopme schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderLoopme, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected loopme params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the loopme schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderLoopme, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId": "10000000"}`,
	`{"publisherId": "10000001", "bundleId": "4321"}`,
	`{"publisherId": "10000002", "placementId": "8888"}`,
	`{"publisherId": "10000003", "bundleId": "5432", "placementId": "7777"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`undefined`,
	`0`,
	`{}`,
	`[]`,
	`{"publisherId": ""}`,
	`{"placementId": ""}`,
	`{"bundleId": ""}`,
	`{"publisherId": "", "placementId": ""}`,
	`{"publisherId": "", "bundleId": ""}`,
	`{"placementId": "", "bundleId": ""}`,
	`{"publisherId": "", "placementId": "", "bundleId": ""}`,
	`{"publisherId": 0}`,
	`{"placementId": 0}`,
	`{"bundleId": 0}`,
	`{"publisherId": 0, "placementId": 0}`,
	`{"publisherId": 0, "bundleId": 0}`,
	`{"placementId": 0, "bundleId": 0}`,
	`{"publisherId": 0, "placementId": 0, "bundleId": 0}`,
	`{"publisherId": "10000000", "placementId": 0}`,
	`{"publisherId": "10000000", "placementId": 100000}`,
	`{"publisherId": "10000000", "bundleId": 0}`,
	`{"publisherId": "10000000", "bundleId": 100000}`,
	`{"placementId": "10000000", "bundleId": 0}`,
	`{"placementId": "10000000", "bundleId": 100000}`,
	`{"publisherId": "10000000", "placementId": "", "bundleId": ""}`,
	`{"publisherId": "", "placementId": "100000", "bundleId": ""}`,
	`{"publisherId": "", "placementId": "", "bundleId": "bundle_id_test"}`,
	`{"unknownField": "value"}`,
	`{"bundleId": []}`,
	`{"placementId": {}}`,
	`{"publisherId": null}`,
	`{"bundleId": null}`,
}
