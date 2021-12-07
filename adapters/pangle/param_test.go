package pangle

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

var validParams = []string{
	`{"token": "SomeAccessToken"}`,
	`{"token": "SomeAccessToken", "appid": "12345", "placementid": "12345"}`,
}

var invalidParams = []string{
	`{"token": ""}`,
	`{"token": 42}`,
	`{"token": null}`,
	`{}`,
	// appid & placementid
	`{"appid": "12345", "placementid": "12345"}`,
	`{"token": "SomeAccessToken", "appid": "12345"}`,
	`{"token": "SomeAccessToken", "placementid": "12345"}`,
	`{"token": "SomeAccessToken", "appid": 12345, "placementid": 12345}`,
	`{"token": "SomeAccessToken", "appid": null, "placementid": null}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderPangle, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderPangle, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}
