package screencore

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
		if err := validator.Validate(openrtb_ext.BidderScreencore, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderScreencore, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"sspPlacementId": "test", "region": "us"}`,
	`{"sspPlacementId": "1", "region": "eu"}`,
	`{"sspPlacementId": "1", "region": "asia"}`,
}

var invalidParams = []string{
	`{}`,
	`{"sspPlacementId": "1"}`,
	`{"region": "us"}`,
	`{"sspPlacementId": "", "region": "us"}`,
	`{"sspPlacementId": "1", "region": ""}`,
	`{"sspPlacementId": 42, "region": "us"}`,
	`{"sspPlacementId": "1", "region": 42}`,
	`{"sspPlacementId": "1", "region": "au"}`,
	`{"placementId": "1"}`,
	`{"endpointId": "1"}`,
}
