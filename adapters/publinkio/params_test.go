package publinkio

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// The bidder-params schema (static/bidder-params/publinkio.json) must accept the
// shapes publishers actually write and allow standard imp.tagid without a placement override.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}
	for _, p := range []string{
		`{}`,
		`{"bidFloor": 1.0}`,
		`{"placementId": "a1b2c3d4"}`,
		`{"placementId": "111c7fb088", "bidFloor": 2.5, "currency": "USD"}`,
	} {
		if err := validator.Validate(openrtb_ext.BidderPublinkio, json.RawMessage(p)); err != nil {
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
		`{"placementId": ""}`,
		`{"placementId": 123}`,
		`{"placementId": "   "}`,
		`{"placementId": "tag", "currency": "EUR"}`,
		`{"placementId": "tag", "bidFloor": -1}`,
	} {
		if err := validator.Validate(openrtb_ext.BidderPublinkio, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}
