package sharethrough

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSharethrough, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Sharethrough params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSharethrough, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pkey": "123"}`,
	`{"pkey": "123", "bcat": []}`,
	`{"pkey": "123", "bcat": ["IAB-1"]}`,
	`{"pkey": "abc", "badv": []}`,
	`{"pkey": "abc", "badv": ["advertiser.com"]}`,
	`{"pkey": "abc123", "bcat": [], "badv": []}`,
	`{"pkey": "abc123", "bcat": ["IAB-1", "IAB-2"], "badv": ["other.advertiser.com"]}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"pkey": 123}`,
	`{"bcat": 123}`,
	`{"badv": 123}`,
	`{"bcat": ["IAB-1", "IAB-2"]}`,
	`{"badv": ["other.advertiser.com"]}`,
	`{"bcat": ["IAB-1", "IAB-2"], "badv": ["other.advertiser.com"]}`,
	`{"pkey": 123, "bcat": ["IAB-1", "IAB-2"], "badv": ["other.advertiser.com"]}`,
}
