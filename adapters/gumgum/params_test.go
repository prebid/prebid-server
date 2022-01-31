package gumgum

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
		if err := validator.Validate(openrtb_ext.BidderGumGum, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected gumgum params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderGumGum, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"zone":"dc9d6be1"}`,
	`{"pubId":12345678}`,
	`{"zone":"dc9d6be1", "pubId":12345678}`,
	`{"zone":"dc9d6be1", "slot":1234567}`,
	`{"pubId":12345678, "slot":1234567}`,
	`{"pubId":12345678, "irisid": "iris_6f9285823a48bne5"}`,
	`{"zone":"dc9d6be1", "irisid": "iris_6f9285823a48bne5"}`,
	`{"zone":"dc9d6be1", "pubId":12345678, "irisid": "iris_6f9285823a48bne5"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`{}`,
	`[]`,
	`true`,
	`2`,
	`{"zone":12345678}`,
	`{"zone":""}`,
	`{"placementId": 1}`,
	`{"zone": true}`,
	`{"placementId": 1, "zone":"1234567"}`,
	`{"pubId":"123456"}`,
	`{"slot":123456}`,
	`{"zone":"1234567", "irisid": ""}`,
	`{"zone":"1234567", "irisid": 1234}`,
	`{"irisid": "iris_6f9285823a48bne5"}`,
}
