package escalax

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderEscalax, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderEscalax, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"accountId": "acc1", "sourceId": "src1"}`,
	`{"accountId": "acc1", "sourceId": "src1", "region": "us"}`,
	`{"accountId": "acc1", "sourceId": "src1", "region": "eu"}`,
	`{"accountId": "acc1", "sourceId": "src1", "region": "apac"}`,

	`{"supplyPlacementId": "sp1"}`,
	`{"supplyPlacementId": "sp1", "region": "us"}`,
	`{"supplyPlacementId": "sp1", "region": "eu"}`,
	`{"supplyPlacementId": "sp1", "region": "apac"}`,
}

var invalidParams = []string{
	`{}`,
	`{"accountId": "acc1"}`,
	`{"sourceId": "src1"}`,
	`{"region": "us"}`,

	`{"accountId": "acc1", "sourceId": "src1", "supplyPlacementId": "sp1"}`,

	`{"accountId": 42, "sourceId": "src1"}`,
	`{"accountId": "acc1", "sourceId": 42}`,
	`{"supplyPlacementId": 42}`,
	`{"accountId": "acc1", "sourceId": "src1", "region": 42}`,

	`{"accountId": "", "sourceId": "src1"}`,
	`{"accountId": "acc1", "sourceId": ""}`,
	`{"supplyPlacementId": ""}`,

	`{"accountId": "acc1", "sourceId": "src1", "region": "asia"}`,
	`{"supplyPlacementId": "sp1", "region": "au"}`,
}
