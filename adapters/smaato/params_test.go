package smaato

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file intends to test static/bidder-params/smaato.json

// These also validate the format of the external API: request.imp[i].ext.smaato

// TestValidParams makes sure that the Smaato schema accepts all imp.ext fields which Smaato supports.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSmaato, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected smaato params: %s \n Error: %s", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the Smaato schema rejects all the imp.ext fields which are not support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSmaato, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"id":"test-id-1234-smaato","tagId": "1123581321","instl": 0,"secure": 0}`,
	`{"id":"test-id-1234-smaato","tagId": "1123581321","instl": 1,"secure": 0}`,
	`{"id":"test-id-1234-smaato","tagId": "1123581321","instl": 1,"secure": 1}`,
	`{"id":"test-id-1234-smaato","tagId": "1123581321","instl": 0,"secure": 1}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"id":"test-id-1234-smaato"}`,
	`"tagId": "1123581321"}`,
	`{"instl": 0}`,
	`{"secure": 0}`,
	`{"tagId": "1123581321","instl": 0,"secure": 0}`,
	`{"instl": 0,"secure": 0}`,
	`{"id":"test-id-1234-smaato","tagId": "1123581321"}`,
	`{"id":"test-id-1234-smaato","tagId": "1123581321","instl": 0}`,
	`{"id":"test-id-1234-smaato","instl": 0,"secure": 0}`,
}
