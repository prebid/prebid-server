package alkimi

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
		if err := validator.Validate(openrtb_ext.BidderAlkimi, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAlkimi, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"token": "ABC"}`,
	`{"token": "ABC", "bidFloor": 1.0}`,
	`{"token": "ABC", "instl": 1}`,
	`{"token": "ABC", "exp": 30}`,
}

var invalidParams = []string{
	`{"token": 42}`,
	`{"token": "ABC", "bidFloor": "invalid"}`,
	`{"token": "ABC", "instl": "invalid"}`,
	`{"token": "ABC", "exp": "invalid"}`,
}
