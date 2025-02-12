package sharethrough

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the JSON schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSharethrough, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the JSON schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSharethrough, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
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
