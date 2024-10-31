package missena

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderMissena, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderMissena, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"apiKey": "PA-123456"}`,
	`{"apiKey": "PA-123456", "placement": "sticky"}`,
	`{"apiKey": "PA-123456", "test": "native"}`,
}

var invalidParams = []string{
	`{"apiKey": ""}`,
	`{"apiKey": 42}`,
	`{"placement": 111}`,
	`{"placement": "sticky"}`,
	`{"apiKey": "PA-123456", "placement": 111}`,
	`{"test": "native"}`,
	`{"apiKey": "PA-123456", "test": 111}`,
}
