package dmx

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderDmx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected dmx params: %s", validParam)
		}
	}
	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderDmx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema was not supposed to be valid: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"memberid": "anyMemberId", "placement_id":"anyPlacementId", "seller_id":"anySellerId", "dmxid":"anyDmxid", "tagid": "25251", "bidfloor": 0.01}`,
	`{"memberid": "anyMemberId", "seller_id": "anySellerId", "dmxid": "anyDmxid", "tagid": "25251", "bidfloor": 0.01}`,
	`{"memberid": "anyMemberId", "placement_id": "anyPlacementId", "dmxid": "anyDmxid", "tagid": "25251", "bidfloor": 0.01}`,
	`{"memberid": "anyMemberId", "placement_id": "anyPlacementId", "seller_id": "anySellerId", "tagid": "25251", "bidfloor": 0.01}`,
	`{"memberid": "anyMemberId", "placement_id": "anyPlacementId", "seller_id": "anySellerId", "dmxid": "anyDmxid", "bidfloor": 0.01}`,
	`{"memberid": "anyMemberId", "placement_id": "anyPlacementId", "seller_id": "anySellerId", "dmxid": "anyDmxid", "tagid": "25251"}`,
	`{"memberid": "anyMemberId"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`true`,
	`{}`,
	// Invalid because missing "memberid"
	`{"placement_id":"anyPlacementId", "seller_id":"anySellerId", "dmxid":"anyDmxid", "tagid": "25251", "bidfloor": 0.01}`,
	// Invalid because "memberid" not a string
	`{"memberid": 5}`,
	// Invalid since dmxid need to be a string
	`{"memberid": "1222", "dmxid": 432423}`,
	//
	// ...more invalid param scenarios
	//
}
