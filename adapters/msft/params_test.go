package msft

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file tests static/bidder-params/msft.json
//
// These validate the format of the external API: request.imp[i].ext.prebid.bidder.msft

// TestValidParams makes sure that the msft schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderMicrosoft, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected msft params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the msft schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderMicrosoft, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placement_id": 123}`,
	`{"member": 123, "inv_code": "test-inv"}`,
	`{"placement_id": 123, "allow_smaller_sizes": true}`,
	`{"placement_id": 123, "use_pmt_rule": false}`,
	`{"placement_id": 123, "keywords": "key1=val1,key2=val2"}`,
	`{"placement_id": 123, "traffic_source_code": "test-source"}`,
	`{"placement_id": 123, "pubclick": "http://example.com/click"}`,
	`{"placement_id": 123, "ext_inv_code": "ext-inv-123"}`,
	`{"placement_id": 123, "ext_imp_id": "ext-imp-456"}`,
	`{"placement_id": 123, "banner_frameworks": [1, 2, 3]}`,
	`{"member": 958, "inv_code": "test-inv-code", "keywords": "genre=rock,age=25"}`,
	`{"placement_id": 123, "allow_smaller_sizes": true, "use_pmt_rule": true, "keywords": "key=value", "traffic_source_code": "multi-param-test"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`42`,
	`[]`,
	`{}`,
	`{"member": 123}`,
	`{"inv_code": "test"}`,
	`{"placement_id": "123"}`,
	`{"member": "123", "inv_code": "test"}`,
	`{"placement_id": 123, "allow_smaller_sizes": "not-a-boolean"}`,
	`{"placement_id": 123, "use_pmt_rule": "not-a-boolean"}`,
	`{"placement_id": 123, "keywords": ["not", "a", "string"]}`,
	`{"placement_id": 123, "traffic_source_code": 123}`,
	`{"placement_id": 123, "pubclick": true}`,
	`{"placement_id": 123, "ext_inv_code": 456}`,
	`{"placement_id": 123, "ext_imp_id": false}`,
	`{"placement_id": 123, "banner_frameworks": "not-array"}`,
	`{"placement_id": 123, "banner_frameworks": ["1", "2", 3]}`,
	`{"member": 123, "inv_code": 456}`,
	`{"placement_id": "string-not-int"}`,
}
