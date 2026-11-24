package riptide

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// TestValidParams makes sure the riptide schema accepts every imp.ext field the adapter supports.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderRiptide, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected riptide params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure the riptide schema rejects the imp.ext fields the adapter doesn't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRiptide, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId": "p", "placementId": "t"}`,
	`{"publisherId": "p", "placementId": "t", "bidfloor": 0.25}`,
	`{"publisherId": "p", "placementId": "t", "bidfloor": 0}`,
}

var invalidParams = []string{
	``,
	`null`,
	`[]`,
	`{}`,
	`{"publisherId": "p"}`,
	`{"placementId": "t"}`,
	`{"publisherId": "", "placementId": "t"}`,
	`{"publisherId": "p", "placementId": ""}`,
	`{"publisherId": 1, "placementId": "t"}`,
	`{"publisherId": "p", "placementId": 1}`,
	`{"publisherId": "p", "placementId": "t", "bidfloor": -1}`,
	`{"publisherId": "p", "placementId": "t", "bidfloor": "1"}`,
}
