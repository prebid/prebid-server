package pubmatic

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

// This file actually intends to test static/bidder-params/pubmatic.json
//
// These also validate the format of the external API: request.imp[i].ext.pubmatic

// TestValidParams makes sure that the pubmatic schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderPubmatic, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected pubmatic params: %s \n Error: %s", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the pubmatic schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderPubmatic, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId":"7890"}`,
	`{"adSlot":"","publisherId":"7890"}`,
	`{"adSlot":"AdTag_Div1","publisherId":"7890"}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890"}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890","keywords":[{"key": "pmZoneID", "value":["zone1"]},{"key": "dctr", "value":[ "v1","v2"]}]}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890","keywords":[{"key": "pmZoneID", "value":["zone1", "zone2"]}], "wrapper":{"profile":5123}}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"adSlot":"AdTag_Div1@728x90:0"}`,
	`{"adSlot":"AdTag_Div1@728x90:0","publisherId":1}`,
	`{"adSlot":123,"publisherId":"7890"}`,
	`{"adSlot":123,"publisherId":7890}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "keywords":[]}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "keywords":["foo"]}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "keywords":[{"key": "foo", "value":[]}]}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "keywords":[{"key": "foo"}]}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "keywords":[{"value":[]}]}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "wrapper":{}}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "wrapper":{"version":"1","profile":5123}}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890", "wrapper":{"version":"1"}}`,
	`{"adSlot":"AdTag_Div1@728x90","publisherId":"7890","keywords":[{"key": "pmZoneID", "value":["1"]}], "wrapper":{"version":1,"profile":"5123"}}`,
}
