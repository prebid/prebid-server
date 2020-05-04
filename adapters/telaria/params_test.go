package telaria

import (
	"encoding/json"
	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
	"testing"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderTelaria, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Telaria params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Telaria schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderTelaria, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"adCode": "string", "seatCode": "string", "originalPublisherid": "string"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"adCode": "string", "originalPublisherid": "string"}`,
	`{"adCode": "string", "seatCode": 5, "originalPublisherid": "string"}`,
}