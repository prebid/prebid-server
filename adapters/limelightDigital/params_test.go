package limelightDigital

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderLimelightDigital, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderLimelightDigital, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"host": "example.com", "publisherId": 1}`,
	`{"host": "example.com", "publisherId": "1"}`,
	`{"host": "example.com", "publisherId": 42}`,
	`{"host": "example.com", "publisherId": "42"}`,
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
	`{"publisherId": 42}`,

	`{"host": "", "publisherId": 42}`,
	`{"host": 42, "publisherId": 42}`,
	`{"host": "example.com:42", "publisherId": 42}`,
	`{"host": "example.com/test", "publisherId": 42}`,
	`{"host": "example.com:42/test", "publisherId": 42}`,
	`{"host": "example", "publisherId": 1}`,
	`{"host": ".example", "publisherId": 1}`,
	`{"host": ".example.com", "publisherId": 1}`,
	`{"host": ".test.example.com", "publisherId": 1}`,
	`{"host": "example.", "publisherId": 1}`,
	`{"host": "example.com.", "publisherId": 1}`,
	`{"host": "test.example.com.", "publisherId": 1}`,
	`{"host": ".example.", "publisherId": 1}`,
	`{"host": ".example.com.", "publisherId": 1}`,
	`{"host": ".test.example.com.", "publisherId": 1}`,

	`{"host": "example.com", "publisherId": ""}`,
	`{"host": "example.com", "publisherId": 0}`,
	`{"host": "example.com", "publisherId": "0"}`,
	`{"host": "example.com", "publisherId": -1}`,
	`{"host": "example.com", "publisherId": "-1"}`,
	`{"host": "example.com", "publisherId": 01}`,
	`{"host": "example.com", "publisherId": "01"}`,
	`{"host": "example.com", "publisherId": -01}`,
	`{"host": "example.com", "publisherId": "-01"}`,
	`{"host": "example.com", "publisherId": -42}`,
	`{"host": "example.com", "publisherId": "-42"}`,
}
