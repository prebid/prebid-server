package adspiro

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
		if err := validator.Validate(openrtb_ext.BidderAdspiro, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAdspiro, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisherId":"66f2c1d9a4b5e6f7a8b9c0d1"}`,
	`{"publisherId":"test"}`,
	`{"publisherId":"p"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`"publisherId"`,
	`[]`,
	`{}`,
	`{"publisherId":""}`,
	`{"publisherId":null}`,
	`{"publisherId":123}`,
	`{"publisherId":["test"]}`,
	`{"publisherId":{"id":"test"}}`,
	`{"publisherid":"test"}`,
	`{"publisherId":"test","placementId":"placement-1"}`,
}
