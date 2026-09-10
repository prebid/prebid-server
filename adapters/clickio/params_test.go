package clickio

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderClickio, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s\nError: %v", p, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderClickio, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"said": "auction-1"}`,
	`{"said": "auction-1", "psid": "preset-1"}`,
	`{"said": "auction-1", "template": "template-1"}`,
	`{"said": "auction-1", "psid": "preset-1", "template": "template-1"}`,
}

var invalidParams = []string{
	`{}`,
	`{"psid": "preset-1"}`,
	`{"template": "template-1"}`,
	`{"said": 123}`,
	`{"said": "auction-1", "psid": 123}`,
	`{"said": "auction-1", "template": 123}`,
	`{"said": true}`,
	`{"said": null}`,
}
