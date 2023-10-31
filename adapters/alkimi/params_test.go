package alkimi

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAppush, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAppush, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"token": "ABC"}`,
	`{"bidFloor": 1.0}`,
	`{"instl": 1}`,
	`{"exp": 30}`,
}

var invalidParams = []string{
	`{"token": 42}`,
	`{"bidFloor": "invalid"}`,
	`{"instl": "invalid"}`,
	`{"exp": "invalid"}`,
}
