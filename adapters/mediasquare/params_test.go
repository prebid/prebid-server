package mediasquare

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
		if err := validator.Validate(openrtb_ext.BidderMediasquare, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderMediasquare, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"owner":"owner-test", "code": "code-test"}`,
}

var invalidParams = []string{
	`{"owner":"owner-test", "code": 42}`,
	`{"owner":"owner-test", "code": nil}`,
	`{"owner":"owner-test", "code": ""}`,
	`{"owner": 42, "code": "code-test"}`,
	`{"owner": nil, "code": "code-test"}`,
	`{"owner": "", "code": "code-test"}`,
	`nil`,
	``,
	`[]`,
	`true`,
}
