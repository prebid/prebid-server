package ix

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
		if err := validator.Validate(openrtb_ext.BidderIx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected ix params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderIx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"siteid":"1234"}`,
	`{"siteID":"12345"}`,
	`{"siteId":"123456"}`,
	`{"siteid":"1234567", "size": [640,480]}`,
}

var invalidParams = []string{
	`{"siteid":""}`,
	`{"siteID":""}`,
	`{"siteId":""}`,
	`{"siteid":"1234", "siteID":"12345"}`,
	`{"siteid":"1234", "siteId":"123456"}`,
	`{"siteid":123}`,
	`{"siteids":"123"}`,
	`{"notaparam":"123"}`,
	`{"siteid":"123", "size": [1,2,3]}`,
	`null`,
	`true`,
	`0`,
	`abc`,
	`[]`,
	`{}`,
}
