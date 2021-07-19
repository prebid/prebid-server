package smarthub

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

var validParams = []string{
	`{"host":"smarthub.test", "seat":"9Q20EdGxzgWdfPYShScl", "token":"eKmw6alpP3zWQhRCe3flOpz0wpuwRFjW"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSmartHub, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected smarthub params: %s", validParam)
		}
	}
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"anyparam": "anyvalue"}`,
	`{"host":"smarthub.test"}`,
	`{"seat":"9Q20EdGxzgWdfPYShScl"}`,
	`{"token":"Y9Evrh40ejsrCR4EtidUt1cSxhJsz8X1"}`,
	`{"seat":"9Q20EdGxzgWdfPYShScl", "token":"alNYtemWggraDVbhJrsOs9pXc3Eld32E"}`,
	`{"host":"smarthub.test", "token":"LNywdP2ebX5iETF8gvBeEoB6Cam64eeq"}`,
	`{"host":"smarthub.test", "seat":"9Q20EdGxzgWdfPYShScl"}`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSmartHub, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
