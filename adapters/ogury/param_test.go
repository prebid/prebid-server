package ogury

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOgury, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"adUnitId": "12", "assetKey": "OGY"}`,
	`{"publisherId": "0b33fb0a-7b2d-44f0-ab0a-a6df93740dba"}`,
	`{"publisherId": "0b33fb0a-7b2d-44f0-ab0a-a6df93740dba", "assetKey": "ogy"}`,
	`{"publisherId": "0b33fb0a-7b2d-44f0-ab0a-a6df93740dba", "adUnitId": "12"}`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOgury, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var invalidParams = []string{
	``,
	`null`,
	`[]`,
	`{"adUnitId": "test 12"}`,
	`{"assetKey": "ogy asset"}`,
	`{"adUnitId": 12, "assetKey": "OGY"}`,
	`{"adUnitId": "45test", "assetKey": false}`,
	`{"publisherId": true}`,

	`{"publisherId": "some publisher", "assetKey": "ogy asset", "adUnitId": "1"}`,
}
