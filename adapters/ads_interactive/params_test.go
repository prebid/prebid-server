package ads_interactive

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
		if err := validator.Validate(openrtb_ext.BidderAdsInteractive, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAdsInteractive, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"placementId": "test"}`,
	`{"placementId": "1"}`,
	`{"endpointId": "test"}`,
	`{"endpointId": "1"}`,
	`{"placementId": "test", "unknownField": "value"}`,
}

var invalidParams = []string{
	`{}`,
	`{"placementId": 42}`,
	`{"endpointId": 42}`,
	`{"placementId": "1", "endpointId": "1"}`,
	`{"placementId": ""}`,
	`{"endpointId": ""}`,
	`{"randomField": "value"}`,
}
