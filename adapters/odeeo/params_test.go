package odeeo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{"sp": "123456789", "tk": "a1b2c3d4"}`,
	`{"sp": "1", "tk": "a"}`,
	`{"sp": "123456789", "tk": "a1b2c3d4", "extra": "ignored"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOdeeo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected odeeo params: %s", validParam)
		}
	}
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`"123456789"`,
	`[]`,
	`{}`,
	`{"sp": "123456789"}`,
	`{"tk": "a1b2c3d4"}`,
	`{"sp": "", "tk": ""}`,
	`{"sp": "", "tk": "a1b2c3d4"}`,
	`{"sp": "123456789", "tk": ""}`,
	`{"sp": 123456789, "tk": "a1b2c3d4"}`,
	`{"sp": "123456789", "tk": 1234}`,
	`{"sp": null, "tk": "a1b2c3d4"}`,
	`{"sp": ["123456789"], "tk": "a1b2c3d4"}`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOdeeo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
