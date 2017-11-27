package appnexus

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file actually intends to test static/bidder-params/appnexus.json
//
// These also validate the format of the external API: request.imp[i].ext.appnexus

// TestValidParams makes sure that the appnexus schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAppnexus, openrtb.RawJSON(validParam)); err != nil {
			t.Errorf("Schema rejected appnexus params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the appnexus schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAppnexus, openrtb.RawJSON(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId":123}`,
	`{"placementId":123,"position":"above"}`,
	`{"placementId":123,"position":"below"}`,
	`{"member":"123","invCode":"456"}`,
	`{"placementId":123, "keywords":[{"key":"foo","value":["bar"]}]}`,
	`{"placementId":123, "keywords":[{"key":"foo","value":["bar", "baz"]}]}`,
	`{"placementId":123, "keywords":[{"key":"foo"}]}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"placementId":"123"}`,
	`{"member":"123"}`,
	`{"member":"123","invCode":45}`,
	`{"placementId":"123","member":"123","invCode":45}`,
	`{"placementId":123, "position":"left"}`,
	`{"placementId":123, "position":"left"}`,
	`{"placementId":123, "reserve":"45"}`,
	`{"placementId":123, "keywords":[]}`,
	`{"placementId":123, "keywords":["foo"]}`,
	`{"placementId":123, "keywords":[{"key":"foo","value":[]}]}`,
	`{"placementId":123, "keywords":[{"value":["bar"]}]}`,
}
