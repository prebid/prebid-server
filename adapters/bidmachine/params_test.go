package bidmachine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBidmachine, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Bidmachine params: %s \n Error: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBidmachine, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"seller_id":"1", "host":"api-eu", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"api-us", "path":"auction/rtb/v2"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"some_random_field":""}`,
	`{"seller_id":""}`,
	`{"seller_id": 1}`,
	`{"seller_id": 1.2}`,
	`{"seller_id": null}`,
	`{"seller_id": true}`,
	`{"seller_id": []}`,
	`{"seller_id": {}}`,
	`{"host":""}`,
	`{"host": 1}`,
	`{"host": 1.2}`,
	`{"host": null}`,
	`{"host": true}`,
	`{"host": []}`,
	`{"host": {}}`,
	`{"path":""}`,
	`{"path": 1}`,
	`{"path": 1.2}`,
	`{"path": null}`,
	`{"path": true}`,
	`{"path": []}`,
	`{"path": {}}`,
	`{"seller_id":"", "path": "", host: ""}`,
	`{"seller_id": 1, "path": 2, host: 3}`,
	`{"seller_id": 1.2}, "path": 5.5, host: 3.3`,
	`{"seller_id": null, "path": null, host: null}`,
	`{"seller_id": true, "path": false, host: true}`,
	`{"seller_id": [], "path": [], host: []}`,
	`{"seller_id": {}, "path": {}, host: {}}`,
}
