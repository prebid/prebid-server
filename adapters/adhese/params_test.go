package adhese

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdhese, json.RawMessage(validParam)); err != nil {
			fmt.Println(err)
			t.Errorf("Schema rejected Adhese params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdhese, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"account": "demo", "location": "loc1", "format": "for1"}`,
	`{"account": "demo", "location": "loc1", "format": "for1", "targets": { "ab": ["test", "test2"]}}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`true`,
	`{"location": "loc1", "format": "for1"}`,
	`{"account": "demo", "format": "for1"}`,
	`{"account": "demo", "location": "loc1"}`,
	`{"account": "demo", "location": "loc1", "format": "for1", "targets": null`,
	`{"account": 5, "location": "loc1", "format": "for1"}`,
	`{"account": "demo", "location": 5, "format": "for1"}`,
	`{"account": "demo", "location": "loc1", "format": 5}`,
	`{"account": "demo", "location": "loc1", "format": "for1", "targets": "test"}`,
	`{"account": "demo", "location": "loc1", "format": "for1", "targets": 5}`,
}
