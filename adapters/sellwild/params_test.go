package sellwild

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
		if err := validator.Validate(openrtb_ext.BidderSellwild, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderSellwild, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"partnerCode":"weatherbug","token":"abc123"}`,
	`{"partnerCode":"Partner_1-a","token":"t"}`,
	`{"partnerCode":"` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `","token":"t"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`[]`,
	`{}`,
	`{"partnerCode":"weatherbug"}`,
	`{"token":"abc123"}`,
	`{"partnerCode":"","token":"abc123"}`,
	`{"partnerCode":"weatherbug","token":""}`,
	`{"partnerCode":"weather bug","token":"abc123"}`,
	`{"partnerCode":"a/b","token":"abc123"}`,
	`{"partnerCode":"` + "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" + `","token":"t"}`,
	`{"partnerCode":123,"token":"abc123"}`,
	`{"partnerCode":"weatherbug","token":123}`,
}
