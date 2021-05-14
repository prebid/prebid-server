package yeahmobi

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderYeahmobi, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected yeahmobi params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the yeahmobi schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderYeahmobi, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pubId": "11233", "zoneId": "sin"}`,
	`{"pubId": "11244", "zoneId": "iad"}`,
}

var invalidParams = []string{
	`{"pubId": "11233"}`,
	`{"zoneId": "aaa"}`,
	`{"pubId": 123, "zoneId": "sin"}`,
	`{"pubId": "", "zoneId": "iad"}`,
	`{"pubId": "11233", "zoneId": ""}`,
}
