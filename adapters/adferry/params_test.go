package adferry

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// The bidder-params schema (static/bidder-params/adferry.json) must accept the
// shapes publishers actually write and reject a missing placementId.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}
	for _, p := range []string{
		`{"placementId": "a1b2c3d4"}`,
		`{"placementId": "111c7fb088", "bidFloor": 2.5, "currency": "USD"}`,
	} {
		if err := validator.Validate(openrtb_ext.BidderAdferry, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}
	for _, p := range []string{
		``,
		`null`,
		`{}`,
		`{"placementId": ""}`,
		`{"placementId": 123}`,
		`{"bidFloor": 1.0}`,
	} {
		if err := validator.Validate(openrtb_ext.BidderAdferry, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}
