package clydo

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
		if err := validator.Validate(openrtb_ext.BidderClydo, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderClydo, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"partnerId": "rxeMfUb992Ocxm9C"}`,
	`{"partnerId": "jDrTOH0ADArSEqrR"}`,
	`{"partnerId": "MZP6aFUyHqjoqMCs"}`,
	`{"partnerId": "rxeMfUb992Ocxm9C", "region": "us"}`,
	`{"partnerId": "rxeMfUb992Ocxm9C", "region": "usw"}`,
	`{"partnerId": "jDrTOH0ADArSEqrR", "region": "eu"}`,
	`{"partnerId": "MZP6aFUyHqjoqMCs", "region": "apac"}`,
}

var invalidParams = []string{
	`{"partnerId": ""}`,
	`{"partnerId": 111}`,
	`{"partnerId": "rxeMfUb992Ocxm9C", "region": "uswest"}`,
	`{"partnerId": "rxeMfUb992Ocxm9C", "region": "europa"}`,
	`{"partnerId": "rxeMfUb992Ocxm9C", "region": "singapore"}`,
}
