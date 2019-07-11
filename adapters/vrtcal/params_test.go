package vrtcal

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

//Vrtcal doesn't currently require any custom fields. This file is included for conformity only
//We do include an unused, non-required custom param in static/bidder-params/vrtcal.json, but only to hinder the prebid server from crashing by looking for at least 1 custom param

// This file actually intends to test static/bidder-params/vrtcal.json
//
// These also validate the format of the external API: request.imp[i].ext.vrtcal
// TestValidParams makes sure that the Vrtcal schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderVrtcal, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Vrtcal params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Vrtcal schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderVrtcal, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{}

var invalidParams = []string{}
