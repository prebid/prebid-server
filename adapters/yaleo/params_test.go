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
	`{"placementId": "460A301D-A9F7-4104-B55F-2A3D4674A973"}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "memberId": 3927}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "maxCpm": 2.5}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "maxCpm": 0}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "memberId": 12345, "maxCpm": 2.5}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "forceDarkTheme": true}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "forceDarkTheme": false}`,
}

var invalidParams = []string{
	`{}`,
	`{"placementId": ""}`,
	`{"placementId": "12345"}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a97"}`,
	`{"placementId": " 460a301d-a9f7-4104-b55f-2a3d4674a973"}`,
	`{"placementId": 12345}`,
	`{"placementId": null}`,
	`{"placementId": true}`,
	`{"memberId": 3927}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "memberId": "3927"}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "memberId": 0}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "memberId": 1.5}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "maxCpm": "2.5"}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "maxCpm": -1}`,
	`{"placementId": "460a301d-a9f7-4104-b55f-2a3d4674a973", "forceDarkTheme": "true"}`,
}
