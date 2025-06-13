package flatads

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
		if err := validator.Validate(openrtb_ext.BidderFlatads, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderFlatads, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{	"token": "66668888",
		"publisherId": "1111"
	}`,
	`{"token": "66668888", "publisherId": "1111"}`,
}

var invalidParams = []string{
	`{}`,
	`{"token": 66668888, "networkId":1111}`,
	`{"token": "66668888"", "networkId":1111}`,
	`{"token": 66668888, "networkId":"1111""}`,
	`{"token": "", "publisherId": "1111"}`,
	`{"token": "66668888", "publisherId": ""}`,
	`{"token": "", "publisherId": ""}`,
}
