package realry

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// Validates static/bidder-params/realry.json — the JSON schema applied
// to imp.ext.prebid.bidder.realry at request-parse time.

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}
	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderRealry, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected realry params: %s — %v", p, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}
	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRealry, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed unexpected realry params: %s", p)
		}
	}
}

var validParams = []string{
	`{"placementId": "slot-1"}`,
	`{"placementId": "slot-1", "sellerId": "seller-acme"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"sellerId": "seller-acme"}`,
	`{"placementId": ""}`,
	`{"placementId": "slot-1", "sellerId": ""}`,
}
