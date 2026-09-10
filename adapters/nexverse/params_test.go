package nexverse

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{"uid": "12345", "pubId": "54321", "pubEpid": "abcde"}`,
	`{"uid": "12345", "pubId": "54321", "pubEpid": "abcde", "isDebug": true}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"uid": "12345"}`,
	`{"uid": "12345", "pubId": "54321"}`,
	`{"uid": "", "pubId": "54321", "pubEpid": "abcde"}`,
	`{"uid": "12345", "pubId": "", "pubEpid": "abcde"}`,
	`{"uid": "12345", "pubId": "54321", "pubEpid": ""}`,
	`{"uid": 12345, "pubId": "54321", "pubEpid": "abcde"}`,
	`{"uid": "12345", "pubId": "54321", "pubEpid": "abcde", "isDebug": "yes"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderNexverse, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Nexverse params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderNexverse, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
