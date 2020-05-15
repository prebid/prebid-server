package improvedigital

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderImprovedigital, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected improvedigital params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderImprovedigital, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId":13245}`,
	`{"placementId":13245, "size": {"w":16, "h":9}}`,
	`{"publisherId":13245, "placementKey": "slotA"}`,
	`{"placementId":13245, "keyValues":{"target1":["foo"],"target2":["bar", "baz"]}}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`true`,
	`2`,
	`{"size": {"w": 10, "h": 5}}`,
	`{"other_optional": true}`,
	`{"size":12345678}`,
	`{"size":""}`,
	`{"placementId":-9}`,
	`{"publisherId":9}`,
	`{"placementId": "1"}`,
	`{"size": true}`,
	`{"placementId": true, "size":"1234567"}`,
	`{"placementId":13245, "publisherId":13245, "placementKey": "slotA"}`,
}
