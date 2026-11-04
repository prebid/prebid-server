package redmob

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
		if err := validator.Validate(openrtb_ext.BidderRedmob, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderRedmob, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"supplyPartnerId":2101}`,
	`{"supplyPartnerId":1}`,
	`{"supplyPartnerId":2147483647}`,
}

var invalidParams = []string{
	``,
	`null`,
	`[]`,
	`{}`,
	`{"supplyPartnerId":null}`,
	`{"supplyPartnerId":"2101"}`,
	`{"supplyPartnerId":2101.5}`,
	`{"supplyPartnerId":0}`,
	`{"supplyPartnerId":-2101}`,
	`{"supplyPartnerID":2101}`,
	`{"placementId":2101}`,
	`{"supplyPartnerId":2101,"bidFloor":0.5}`,
}
