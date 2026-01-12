package iqiyi

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderIqiyi, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected iqiyi params: %s \n Error: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderIqiyi, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"accountid":"123"}`,
	`{"accountid":"test-account-id"}`,
	`{"accountid":"a"}`,
	`{"accountid":"account-id-with-dashes"}`,
	`{"accountid":"account_id_with_underscores"}`,
	`{"accountid":"AccountID123"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`false`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"accountid":""}`,
	`{"accountid":123}`,
	`{"accountid":null}`,
	`{"accountid":true}`,
	`{"accountid":[]}`,
	`{"accountid":{}}`,
	`{"accountId":"123"}`,
	`{"AccountID":"123"}`,
	`{"account_id":"123"}`,
}

