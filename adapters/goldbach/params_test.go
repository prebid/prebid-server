package goldbach

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/goldbach.json

// TestValidParams makes sure that the goldbach schema accepts all imp.ext fields which we intend to support.

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderGoldbach, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected goldbach params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the goldbach schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderGoldbach, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId":"123","slotId":"333"}`,
	`{"publisherId":"123","slotId":"333","customTargeting":{"key1":"value1","key2":["value2","value3"]}}`,
}

var invalidParams = []string{
	`4.2`,
	`5`,
	`[]`,
	``,
	`null`,
	`true`,
	`{}`,
	`{"publisherId":123,"slotId":"333"}`,
	`{"publisherId":"123","slotId":333}`,
	`{"publisherId":"1234"}`,
	`{"slotId":"abc"}`,
	`{"publisherId":"123","slotId":"333","customTargeting":{"key1":123,"key2":["value2","value3"]}}`,
	`{"publisherId":"123","slotId":"333","customTargeting":{"key1":false,"key2":["value2","value3"]}}`,
	`{"publisherId":"123","slotId":"333","customTargeting":{"key1":"value1","key2":[123,456]}}`,
	`{"publisherId":"123","slotId":"333","customTargeting":{"key1":"value1","key2":[true,false]}}`,
}
