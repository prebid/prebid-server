package bmtm

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

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBmtm, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected valid params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBmtm, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placement_id": 329}`,
	`{"placement_id": 12450}`,
}

var invalidParams = []string{
	`{"placement_id": "548d4e75w7a5d8e1w7w5r7ee7"}`,
	`{"placement_id": "42"}`,
}
