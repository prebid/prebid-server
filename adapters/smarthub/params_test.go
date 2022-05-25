package smarthub

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

var validParams = []string{
	`{"partnerName":"partnertest", "seat":"9Q20EdGxzgWdfPYShScl", "token":"eKmw6alpP3zWQhRCe3flOpz0wpuwRFjW"}`,
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
	`{"partnerName":"partnertest"}`,
	`{"seat":"9Q20EdGxzgWdfPYShScl"}`,
	`{"token":"Y9Evrh40ejsrCR4EtidUt1cSxhJsz8X1"}`,
	`{"seat":"9Q20EdGxzgWdfPYShScl", "token":"alNYtemWggraDVbhJrsOs9pXc3Eld32E"}`,
	`{"partnerName":"partnertest", "token":"LNywdP2ebX5iETF8gvBeEoB6Cam64eeq"}`,
	`{"partnerName":"partnertest", "seat":"9Q20EdGxzgWdfPYShScl"}`,
	`{"partnerName":"", "seat":"", "token":""}`,
	`{"partnerName":"", "seat":"9Q20EdGxzgWdfPYShScl", "token":"alNYtemWggraDVbhJrsOs9pXc3Eld32E"}`,
	`{"partnerName":"partnertest", "seat":"9Q20EdGxzgWdfPYShScl", "token":""}`,
	`{"partnerName":"partnertest", "seat":"", "token":"alNYtemWggraDVbhJrsOs9pXc3Eld32E"}`,
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
