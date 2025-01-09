package smartadserver

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/smartadserver.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.smartadserver

// TestValidParams makes sure that the smartadserver schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSmartAdserver, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected smartadserver params: %s \n Error: %s", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the smartadserver schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSmartAdserver, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"networkId":73}`,
	`{"networkId":73,"siteId":1,"pageId":2,"formatId":3}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"networkId":"73"}`,
	`{"networkId":"73","siteId":"1","pageId":"2","formatId":"3"}`,
	`{"siteId":1,"pageId":2,"formatId":3}`,
	`{"networkId":73,"pageId":2,"formatId":3}`,
	`{"networkId":73,"siteId":1,"formatId":3}`,
	`{"networkId":73,"siteId":1,"pageId":2}`,
}
