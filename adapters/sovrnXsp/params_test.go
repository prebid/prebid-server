package sovrnXsp

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSovrnXsp, json.RawMessage(param)); err != nil {
			t.Errorf("Schema rejected sovrnXsp params: %s", param)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSovrnXsp, json.RawMessage(param)); err == nil {
			t.Errorf("Schema allowed sovrnXsp params: %s", param)
		}
	}
}

var validParams = []string{
	`{"pub_id":"1234"}`,
	`{"pub_id":"1234","med_id":"1234"}`,
	`{"pub_id":"1234","med_id":"1234","zone_id":"abcdefghijklmnopqrstuvwxyz"}`,
	`{"pub_id":"1234","med_id":"1234","zone_id":"abcdefghijklmnopqrstuvwxyz","force_bid":true}`,
	`{"pub_id":"1234","med_id":"1234","zone_id":"abcdefghijklmnopqrstuvwxyz","force_bid":false}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`0`,
	`[]`,
	`{}`,
	`{"pub_id":""}`,
	`{"pub_id":"123"}`,
	`{"pub_id":"1234","zone_id":"123"}`,
}
