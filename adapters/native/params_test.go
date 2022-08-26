package native

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/native.json
//
// These also validate the format of the external API: request.imp[i].ext.rubicon

// TestValidParams makes sure that the Rubicon schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderNative, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected native params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Rubicon schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderNative, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"accountId":6,"siteId":4,"zoneId":1234}`,
	`{"accountId":"6","siteId":"4","zoneId":"1234"}`,
	// `{"inv":321,"mname":"pcl1"}`,
	// `{"inv":321,"mname":"12345"}`,
	// `{"mid":123,"inv":321,"mname":"pcl1"}`,
	// `{"mid":"123","inv":321,"mname":"pcl1"}`,
	// `{"mid":"123","priceType":"gross"}`,
	// `{"mid":"123","priceType":"net"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"notAccountId":"123"}`,
	`{"accountId":"6","zoneId":"1234"}`,
	`{"accountId":"6","siteId":"4"}`,
	`{"siteId":"4","zoneId":"1234"}`,
	`{"accountId":"abba"}`,
	`{"accountId":"tomato","siteId":"siteId","zoneId":"zoneId"}`,
	`{"accountId":"2","siteId":"siteId","zoneId":"1"}`,
	`{"accountId":"tomato","1":"siteId","zoneId":"2"}`,
	`{"accountId":"1","siteId":"2","zoneId":"zoneId"}`,
	// `{"inv":321,"mname":12345}`,
	// `{"inv":321}`,
	// `{"inv":"321"}`,
	// `{"mname":"12345"}`,
	// `{"mid":"123","priceType":"GROSS"}`,
}
