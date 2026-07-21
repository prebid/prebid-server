package vdoai

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// This file tests static/bidder-params/vdoai.json
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.vdoai

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderVdoai, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected vdoai params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderVdoai, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"adUnitId":123456,"adUnitType":"banner","host":"exchange.ortb.net","publisherId":"pub-abc"}`,
	`{"adUnitId":123456,"adUnitType":"video","host":"ads.vdo.ai","publisherId":"pub-xyz"}`,
	`{"adUnitId":123456,"adUnitType":"banner","host":"exchange.ortb.net","publisherId":"pub-abc","bidfloor":1.5}`,
	`{"adUnitId":123456,"adUnitType":"banner","host":"exchange.ortb.net","publisherId":"pub-abc","custom1":"val1","custom2":"val2","custom3":"val3","custom4":"val4","custom5":"val5"}`,
	`{"adUnitId":123456,"adUnitType":"video","host":"ads.vdo.ai","publisherId":"pub-xyz","bidfloor":2.0,"custom1":"c1"}`,
	`{"adUnitId":"123456","adUnitType":"banner","host":"exchange.ortb.net","publisherId":"pub-abc"}`,
	`{"adUnitId":"vdo-123456","adUnitType":"banner","host":"exchange.ortb.net","publisherId":"pub-abc"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"adUnitId":"123456"}`,
	`{"adUnitType":"banner"}`,
	`{"adUnitId":123456,"adUnitType":"banner"}`,
	`{"adUnitId":123456,"adUnitType":"video"}`,
	`{"adUnitId":123456,"adUnitType":"banner","host":"exchange.ortb.net"}`,
	`{"adUnitId":123456,"adUnitType":"banner","publisherId":"pub-abc"}`,
	`{"adUnitId":"123456","adUnitType":"banner"}`,
	`{"adUnitId":"123456","adUnitType":"banner","host":"exchange.ortb.net","publisherId":"pub-abc","bidfloor":"notanumber"}`,
}
