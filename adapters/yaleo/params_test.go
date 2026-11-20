package yaleo

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
		if err := validator.Validate(openrtb_ext.BidderYaleo, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderYaleo, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973"}`,
	`{"placementId": "12345"}`,
	`{"placementId": "12345", "memberId": 3927}`,
	`{"placementId": "12345", "maxCpm": 2.5}`,
	`{"placementId": "12345", "memberId": 12345, "maxCpm": 2.5}`,
	`{"placementId": "12345", "forceDarkTheme": true}`,
}

var invalidParams = []string{
	`{}`,
	`{"placementId": ""}`,
	`{"placementId": 12345}`,
	`{"placementId": null}`,
	`{"placementId": true}`,
	`{"memberId": 3927}`,
	`{"placementId": "12345", "memberId": "3927"}`,
	`{"placementId": "12345", "maxCpm": "2.5"}`,
}
