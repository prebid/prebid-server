package synacormedia

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/synacormedia.json
//
// These also validate the format of the external API: request.imp[i].ext.synacormedia

// TestValidParams makes sure that the synacormedia schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSynacormedia, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected synacormedia params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the synacormedia schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSynacormedia, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"seatId": "123", "tagId":"234"}`,
}

var invalidParams = []string{
	`{"seatId": 123, "tagId":234}`,
}
