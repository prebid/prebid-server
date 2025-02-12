package flipp

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
		if err := validator.Validate(openrtb_ext.BidderFlipp, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderFlipp, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"siteId": 1243066,
		"zoneIds": [285431]
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"siteId": 1243066,
		"options": {}
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"siteId": 1243066,
		"zoneIds": [285431],
		"options": {
			"startCompact": true
		}
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"siteId": 1243066,
		"zoneIds": [285431],
		"options": {
			"startCompact": false,
			"dwellExpand": true,
			"contentCode": "test-code"
		}
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"siteId": 1243066,
		"zoneIds": [285431],
		"ip": "123.123.123.123",
		"options": {
			"startCompact": false
		}
	}`,
}

var invalidParams = []string{
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"zoneIds": [285431]
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"siteId": 1243066,
		"zoneIds": [285431]
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "abc",
		"siteId": 1243066,
		"zoneIds": [285431]
	}`,
	`{
		"creativeType": "NativeX",
		"siteId": 1243066,
		"zoneIds": [285431]
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"siteId": "123abc",
		"zoneIds": [285431]
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"creativeType": "NativeX",
		"siteId": 1243066,
		"zoneIds": ["abc123"]
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"siteId": 1243066,
		"startCompact": "true"
	}`,
	`{
		"publisherNameIdentifier": "wishabi-test-publisher",
		"siteId": 1243066,
		"options": {
			"startCompact": "true"
		}
	}`,
}
