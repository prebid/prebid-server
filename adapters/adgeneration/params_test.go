package adgeneration

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}
	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdgeneration, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adgeneration params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdgeneration, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"id":"12345"}`,
	`{"id":"123456"}`,
	`{"id":""}`,
	`{"id":"12345","other_params":"hoge"}`,
}

var invalidParams = []string{
	`{}`,
	`null`,
	`12345`,
	`{"id":123456}`,
}
