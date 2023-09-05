package appstock

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAppstock, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAppstock, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisherId": 1}`,
	`{"publisherId": "1"}`,
	`{"publisherId": 42}`,
	`{"publisherId": "42"}`,
	`{"host": "example.com", "publisherId": "42"}`,
	`{"publisherId": "42", "adUnitId": 123, "adUnitType": "banner"}`,
	`{"host": "example.com", "publisherId": "42", "adUnitId": 123, "adUnitType": "banner"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`true`,
	`{}`,
	`{"host": "example.com". "publisherId": 1}`,

	`{"host": "example.com"}`,

	`{"publisherId": ""}`,
	`{"publisherId": 0}`,
	`{"publisherId": "0"}`,
	`{"publisherId": -1}`,
	`{"publisherId": "-1"}`,
	`{"publisherId": 01}`,
	`{"publisherId": "01"}`,
	`{"publisherId": -01}`,
	`{"publisherId": "-01"}`,
	`{"publisherId": -42}`,
	`{"publisherId": "-42"}`,
}
