package frvradn

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
		if err := validator.Validate(openrtb_ext.BidderFRVRAdNetwork, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderFRVRAdNetwork, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisher_id": "247f36ed-bda5-4159-86f1-e383849e7810", "ad_unit_id": "63c36c82-3246-4931-97f9-4f16a9639ba9"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`{}`,
	`{"publisher_id": "247f36ed-bda5-4159-86f1-e383849e7810"}`,
	`{"ad_unit_id": "247f36ed-bda5-4159-86f1-e383849e7810"}`,
}
