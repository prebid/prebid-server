package adquery

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
		if err := validator.Validate(openrtb_ext.BidderAdquery, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAdquery, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"placementId": "6d93f2a0e5f0fe2cc3a6e9e3ade964b43b07f897", "type": "banner300x250"}`,
}

var invalidParams = []string{
	`{}`,
	`{"placementId": 42}`,
	`{"type": 3}`,
	`{"placementId": "6d93f2a0e5f0fe2cc3a6e9e3ade964b43b07f897"}`,
	`{"type": "banner"}`,
	`{"placementId": 42, "type": "banner"}`,
	`{"placementId": "too_short", "type": "banner"}`,
	`{"placementId": "6d93f2a0e5f0fe2cc3a6e9e3ade964b43b07f897", "type": ""}`,
}
