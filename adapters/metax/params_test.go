package metax

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/metax.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.metax

// TestValidParams makes sure that the metax schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderMetaX, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected metax params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the metax schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderMetaX, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId": 10000000, "adunit": 100000}`,
}

var invalidParams = []string{
	``,
	`null`,
	`undefined`,
	`0`,
	`{}`,
	`[]`,
	`{"publisherId": ""}`,
	`{"adunit": ""}`,
	`{"publisherId": "", "adunit": ""}`,
	`{"publisherId": "10000000", "adunit": "100000"}`,
	`{"publisherId": 0, "adunit": 0}`,
	`{"publisherId": 10000000, "adunit": 0}`,
	`{"publisherId": 0, "adunit": 100000}`,
}
