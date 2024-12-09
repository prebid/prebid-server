package concert

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
		if err := validator.Validate(openrtb_ext.BidderConcert, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderConcert, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"partnerId": "partner_name"}`,
	`{"partnerId": "partner_name", "placementId": 1234567}`,
	`{"partnerId": "partner_name", "placementId": 1234567, "site": "site_name"}`,
	`{"partnerId": "partner_name", "placementId": 1234567, "site": "site_name", "slot": "slot_name"}`,
	`{"partnerId": "partner_name", "placementId": 1234567, "site": "site_name", "slot": "slot_name", "sizes": [[1030, 590]]}`,
}

var invalidParams = []string{
	`{"partnerId": ""}`,
	`{"placementId": 1234567}`,
	`{"site": "site_name"}`,
	`{"slot": "slot_name"}`,
	`{"sizes": [[1030, 590]]}`,
	`{"placementId": 1234567, "site": "site_name", "slot": "slot_name", "sizes": [[1030, 590]]}`,
}
