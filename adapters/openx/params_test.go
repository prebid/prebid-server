package openx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/openx.json
//
// These also validate the format of the external API: request.imp[i].ext.openx

// TestValidParams makes sure that the openx schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOpenx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected openx params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the openx schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOpenx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"unit": "123", "delDomain": "foo.ba"}`,
	`{"unit": "123", "delDomain": "foo.bar"}`,
	`{"unit": "123", "delDomain": "foo.bar", "customFloor": 0.1}`,
	`{"unit": "123", "delDomain": "foo.bar", "customParams": {"foo": "bar"}}`,
	`{"unit": "123", "delDomain": "foo.bar", "customParams": {"foo": ["bar", "baz"]}}`,
}

var invalidParams = []string{
	`{"unit": "123"}`,
	`{"delDomain": "foo.bar"}`,
	`{"unit": "", "delDomain": "foo.bar"}`,
	`{"unit": "123", "delDomain": ""}`,
	`{"unit": "123a", "delDomain": "foo.bar"}`,
	`{"unit": "123", "delDomain": "foo.b"}`,
	`{"unit": "123", "delDomain": "foo.barr"}`,
	`{"unit": "123", "delDomain": ".bar"}`,
	`{"unit": "123", "delDomain": "foo.bar", "customFloor": "0.1"}`,
	`{"unit": "123", "delDomain": "foo.bar", "customFloor": -0.1}`,
	`{"unit": "123", "delDomain": "foo.bar", "customParams": "foo: bar"}`,
}
