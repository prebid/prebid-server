package kueezrtb

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

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderKueezRTB, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected valid params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderKueezRTB, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"cId": "provided_cid_123"}`,
}

var invalidParams = []string{
	`{"cId": 123}`,
	`{"cId": true}`,
	`{"cId": ["array"]}`,
	`{"cId": {}`,
	`{"cId": ""}`,
	`{"cId": null}`,
	`{"cId": "provided_cid_123", "extra": "field"}`,
	`{"cid": "valid_cid"}`,
	`{"cId": "invalid_chars_!@#$%^&*()"}`,
}
