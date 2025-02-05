package displayio

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderDisplayio, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected dmx params: %s", validParam)
		}
	}
	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderDisplayio, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema was not supposed to be valid: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId": "anyPlacementId", "publisherId":"anyPublisherId", "inventoryId":"anyInventoryId"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`true`,
	`{}`,
	`{"placementId": 1, "publisherId":"anyPublisherId", "inventoryId":"anyInventoryId"}`,
	`{"placementId": "anyPlacementId", "publisherId":1, "inventoryId":"anyInventoryId"}`,
	`{"placementId": "anyPlacementId", "publisherId":"anyPublisherId", "inventoryId":1}`,
	`{"publisherId":"anyPublisherId", "inventoryId":"anyInventoryId"}`,
	`{"placementId": "anyPlacementId", "inventoryId":"anyInventoryId"}`,
	`{"placementId": "anyPlacementId", "publisherId":"anyPublisherId"}`,
	`{"placementId": "anyPlacementId"}`,
	`{"inventoryId":"anyInventoryId"}`,
	`{"publisherId":"anyPublisherId"}`,
}
