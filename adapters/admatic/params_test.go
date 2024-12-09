package admatic

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
		if err := validator.Validate(openrtb_ext.BidderAdmatic, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAdmatic, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{	"host": "layer.serve.admatic.com.tr",
		"networkId": 1111,
		"ext": {
			"key1": "value1",
			"key2": "value2"
		}
	}`,
	`{"host": "layer.serve.admatic.com.tr", "networkId": 1111}`,
}

var invalidParams = []string{
	`{"ext": {
		"key1": "value1",
		"key2": "value2"
	}`,
	`{}`,
	`{"host": 123, "networkId":"1111"}`,
	`{"host": "layer.serve.admatic.com.tr", "networkId":"1111"}`,
	`{"host": 1111, "networkId":1111}`,
}
