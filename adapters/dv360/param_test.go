package dv360

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

var validParams = []string{
	`{}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderDV360, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}
