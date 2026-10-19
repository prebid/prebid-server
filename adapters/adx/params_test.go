package adx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// This file intends to test static/bidder-params/adx.json
//
// These also validate the format of the external API:
// request.imp[i].ext.prebid.bidder.adx

// TestValidParams makes sure that the adx schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adx params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the adx schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"supply_id":"1"}`,
	`{"supply_id":"abc-123"}`,
	`{"supply_id":"sk_test"}`,
	`{"supply_id":"qr7m2xkd"}`, // the shape Console issues for a supply source
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"supply_id":""}`,
	`{"supply_id":1}`,
	`{"supply_id":null}`,
	`{"supplyId":"1"}`,
	// The key is one path segment: separators, query and fragment characters
	// would rewrite the request path, so they are rejected at the schema.
	`{"supply_id":"1/../../config/reload"}`,
	`{"supply_id":"a/b"}`,
	`{"supply_id":"1?debug=1"}`,
	`{"supply_id":"a#b"}`,
	`{"supply_id":"a b"}`,
	`{"supply_id":"%2F"}`,
}
