package bidstack

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

var validParams = []string{
	`{"publisherId": "355cf800-8348-433a-9d95-70345fa70afc"}`,
	`{"publisherId": "355cf800-8348-433a-9d95-70345fa70afc","placementId":"Some placement ID"}`,
	`{"publisherId": "355cf800-8348-433a-9d95-70345fa70afc","consent":false}`,
	`{"publisherId": "355cf800-8348-433a-9d95-70345fa70afc","placementId":"Some placement ID","consent":true}`,
}

var invalidParams = []string{
	`{"publisherId": ""}`,
	`{"publisherId": "non-uuid"}`,
	`{"consent": "true"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBidstack, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderBidstack, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}
