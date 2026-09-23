package aps

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAps, json.RawMessage(param)); err != nil {
			t.Errorf("Schema rejected valid params: %s", param)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAps, json.RawMessage(param)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", param)
		}
	}
}

var validParams = []string{
	`{"accountID":"test-account"}`,
	`{"accountID":"3703"}`,
}

var invalidParams = []string{
	`{}`,
	`{"accountID":123}`,
	`[]`,
	`"aps"`,
	`1`,
	`true`,
	`null`,
}
