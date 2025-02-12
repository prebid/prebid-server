package algorix

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
		if err := validator.Validate(openrtb_ext.BidderAlgorix, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected algoirx params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the algorix schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAlgorix, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"sid": "11233", "token": "sin", "placementId": "123", "appId": "abc"}`,
	`{"sid": "11244", "token": "iad"}`,
	`{"sid": "11244", "token": "iad", "region": "APAC"}`,
	`{"sid": "11244", "token": "iad", "region": "USE"}`,
	`{"sid": "11244", "token": "iad", "region": "EUC"}`,
}

var invalidParams = []string{
	`{"sid": "11233"}`,
	`{"token": "aaa"}`,
	`{"sid": 123, "token": "sin"}`,
	`{"sid": "", "token": "iad"}`,
	`{"sid": "11233", "token": ""}`,
	`{"sid": "11233", "token": "sin", "placementId": 123, "appId": "abc"}`,
	`{"sid": "11233", "token": "sin", "placementId": "123", "appId": 456}`,
	`{"sid": "11233", "token": "sin", "placementId": 123, "appId": 456}`,
	`{"sid": "11233", "token": "sin", "region": 123}`,
}
