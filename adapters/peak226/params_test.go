package peak226

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{ "publisherId": "pub-123", "placementId": "plc-456" }`,
	`{ "publisherId": "pub-123", "placementId": "plc-456", "region": "us" }`,
	`{ "publisherId": "pub-123", "placementId": "plc-456", "region": "eu" }`,
	`{ "publisherId": "pub-123", "placementId": "plc-456", "region": "jp" }`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderPeak226, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Peak226 params: %s\n Error: %s", validParam, err)
		}
	}
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{ "placementId": "plc-456" }`,
	`{ "publisherId": "pub-123" }`,
	`{ "publisherId": "", "placementId": "plc-456" }`,
	`{ "publisherId": "pub-123", "placementId": "" }`,
	`{ "publisherId": 123, "placementId": "plc-456" }`,
	`{ "publisherId": "pub-123", "placementId": 456 }`,
	`{ "publisherId": "pub-123", "placementId": "plc-456", "region": "asia" }`,
	`{ "publisherId": "pub-123", "placementId": "plc-456", "region": 1 }`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderPeak226, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
