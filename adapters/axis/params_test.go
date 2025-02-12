package axis

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
		if err := validator.Validate(openrtb_ext.BidderAxis, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAxis, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", p)
		}
	}
}

var validParams = []string{
	`{"integration":"000000", "token":"000000"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"anyparam": "anyvalue"}`,
	`{"integration":"9Q20EdGxzgWdfPYShScl"}`,
	`{"token":"Y9Evrh40ejsrCR4EtidUt1cSxhJsz8X1"}`,
	`{"integration":"9Q20EdGxzgWdfPYShScl", "token":"alNYtemWggraDVbhJrsOs9pXc3Eld32E"}`,
	`{"integration":"", "token":""}`,
	`{"integration":"9Q20EdGxzgWdfPYShScl", "token":""}`,
	`{"integration":"", "token":"alNYtemWggraDVbhJrsOs9pXc3Eld32E"}`,
}
