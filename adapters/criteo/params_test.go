package criteo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/criteo.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.criteo

// TestValidParams makes sure that the criteo schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderCriteo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected criteo params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the criteo schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderCriteo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"zoneid": 123456}`,
	`{"zoneId": 123456}`,
	`{"networkid": 78910}`,
	`{"networkId": 78910}`,
	`{"zoneid": 123456, "networkid": 78910}`,
	`{"zoneId": 123456, "networkId": 78910}`,
	`{"zoneid": 0, "networkid": 0}`,
	`{"zoneId": 0, "networkId": 0}`,
	`{"zoneid": 123456, "pubid": "testpubid"}`,
	`{"zoneid": 123456, "uid": 100}`,
	`{"zoneid": 123456, "networkid": 78910, "pubid": "testpubid"}`,
	`{"zoneid": 123456, "networkid": 78910, "uid": 100}`,
	`{"zoneid": 123456, "networkid": 78910, "uid": 100, "pubid": "testpubid"}`,
	`{"networkId": 78910, "pubid": "testpubid"}`,
	`{"networkid": 78910, "uid": 100}`,
	`{"networkid": 78910, "uid": 100, "pubid": "testpubid"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"zoneid": -123}`,
	`{"zoneId": -123}`,
	`{"networkid": -321}`,
	`{"networkId": -321}`,
	`{"zoneid": -123, "networkid": -321}`,
	`{"zoneId": -123, "networkId": -321}`,
	`{"zoneid": -1}`,
	`{"networkid": -1}`,
	`{"zoneid": -1, "networkid": -1}`,
	`{"zoneid": 0, "networkid": 0, "pubid": ""}`,
	`{"zoneid": 0, "networkid": 0, "pubid": null}`,
	`{"zoneid": 0, "networkid": 0, "uid": null}`,
}
