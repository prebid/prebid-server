package smaato

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file intends to test static/bidder-params/smaato.json

// These also validate the format of the external API: request.imp[i].bidRequestExt.smaato

// TestValidParams makes sure that the Smaato schema accepts all imp.bidRequestExt fields which Smaato supports.
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

// TestInvalidParams makes sure that the Smaato schema rejects all the imp.bidRequestExt fields which are not support.
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
	`{"publisherId":"test-id-1234-smaato","adspaceId": "1123581321"}`,
	`{"publisherId":"test-id-1234-smaato","adbreakId": "4123581321"}`,
	`{"publisherId":"test-id-1234-smaato","adspaceId": "1123581321","adbreakId": "4123581321"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"publisherId":"test-id-1234-smaato"}`,
	`{"adspaceId": "1123581321"}`,
	`{"publisherId":false}`,
	`{"adspaceId":false}`,
	`{"publisherId":0,"adspaceId": 1123581321}`,
	`{"publisherId":false,"adspaceId": true}`,
	`{"instl": 0}`,
	`{"secure": 0}`,
	`{"adspaceId": "1123581321","instl": 0,"secure": 0}`,
	`{"instl": 0,"secure": 0}`,
	`{"publisherId":"test-id-1234-smaato","instl": 0,"secure": 0}`,
}
