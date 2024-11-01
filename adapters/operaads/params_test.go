package operaads

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/operaads.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.operaads

// TestValidParams makes sure that the operaads schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOperaads, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected operaads params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the operaads schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOpenx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId": "s123", "endpointId": "ep12345", "publisherId": "pub12345"}`,
}

var invalidParams = []string{
	`{"placementId": "s123"}`,
	`{"endpointId": "ep12345"}`,
	`{"publisherId": "pub12345"}`,
	`{"placementId": "s123", "endpointId": "ep12345"}`,
	`{"placementId": "s123", "publisherId": "pub12345"}`,
	`{"endpointId": "ep12345", "publisherId": "pub12345"}`,
	`{"placementId": "", "endpointId": "", "publisherId": ""}`,
}
