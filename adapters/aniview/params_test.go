package aniview

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

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAniview, json.RawMessage(validParam)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAniview, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"AV_PUBLISHERID": "1234567890abcdef12345678", "AV_CHANNELID": "abcdef1234567890abcdef12"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`{"AV_PUBLISHERID": "1234567890abcdef12345678"}`,
	`{"AV_CHANNELID": "abcdef1234567890abcdef12"}`,
	`{"AV_PUBLISHERID": 123, "AV_CHANNELID": "abcdef1234567890abcdef12"}`,
	`{"AV_PUBLISHERID": "", "AV_CHANNELID": "abcdef1234567890abcdef12"}`,
}
