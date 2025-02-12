package readpeak

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
		if err := validator.Validate(openrtb_ext.BidderReadpeak, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderReadpeak, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisherId": ""}`,
	`{"publisherId": "Some Pub ID", "siteId": ""}`,
	`{"publisherId": "Some Pub ID", "siteId": "Some Site ID"}`,
	`{"publisherId": "Some Pub ID", "siteId": "Some Site ID", "bidfloor": 1.5}`,
	`{"publisherId": "Some Pub ID", "siteId": "Some Site ID", "bidfloor": 1.5, "tagId": "Some tag ID"}`,
}

var invalidParams = []string{
	`{"publisherId": 42}`,
	`{"publisherId": "42", "siteId": 42}`,
	`{"siteId": 42}`,
	`{"publisherId": "Some Pub ID", "siteId": 42}`,
	`{"publisherId": "Some Pub ID", "siteId": "Some Site ID", bidfloor: "1.5"}`,
	`{"publisherId": "Some Pub ID", "siteId": "Some Site ID", bidfloor: 1.5, tagId: 1}`,
}
