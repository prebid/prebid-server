package thetradedesk

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
		if err := validator.Validate(openrtb_ext.BidderTheTradeDesk, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected TheTradeDesk params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the TheTradeDesk schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderTheTradeDesk, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId": "123456", "supplySourceId": "xyz_123"}`,
	`{"publisherId": "pub-123456", "supplySourceId": "direct_123"}`,
	`{"publisherId": "publisherIDAllString", "supplySourceId": "direct123"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"publisherId": 123456, "supplySourceId": 123}`,
	`{"publisherId": 0, "supplySourceId": 0}`,
}
