package visx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderVisx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected visx params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderVisx, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"uid":13245}`,
	`{"uid":"13245"}`,
	`{"uid":13245, "size": [10,5]}`,
	`{"uid":13245, "other_optional": true}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`[]`,
	`true`,
	`2`,
	`{"size":12345678}`,
	`{"size":""}`,
	`{"uid": "-1"}`,
	`{"uid": "232af"}`,
	`{"uid": "af213"}`,
	`{"uid": "af"}`,
	`{"size": true}`,
	`{"uid": true, "size":"1234567"}`,
}
