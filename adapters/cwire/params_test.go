package cwire

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/cwire.json
// These also validate the format of the external API: request.imp[i].ext.bidder
// TestValidParams makes sure that the cwire schema accepts all imp.ext fields which we intend to support.

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderCWire, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected cwire params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the cwire schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderCWire, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"cwcreative":"3746","cwdebug":true,"cwfeatures":["feat1","feat2"]}`,
	`{"cwcreative":"3746","cwdebug":true}`,
	`{"cwcreative":"3746"}`,
	`{"cwdebug":true,"cwfeatures":["feat1","feat2"]}`,
	`{"cwdebug":true}`,
	`{"cwfeatures":["feat1","feat2"]}`,
	`{"cwfeatures":["feat1"]}`,
	`{"cwfeatures":[]}`,
	`{"pageId":321,"cwcreative":"3746"}`,
	`{"pageId":321}`,
	`{"placementId":123,"cwcreative":"3746"}`,
	`{"placementId":123,"cwdebug":true,"cwfeatures":["feat1","feat2"]}`,
	`{"placementId":123,"pageId":321,"cwcreative":"3746","cwdebug":true,"cwfeatures":["feat1","feat2"]}`,
	`{"placementId":123,"pageId":321,"cwcreative":"3746"}`,
	`{"placementId":123,"pageId":321}`,
	`{"placementId":123}`,
	`{"placementId":123,"domainId":333,"pageId":321,"cwcreative":"3746"}`,
	`{"placementId":123,"domainId":333,"pageId":321}`,
	`{"placementId":123},"domainId":333}`,
	`{}`,
}

var invalidParams = []string{
	`4.2`,
	`5`,
	`[]`,
	``,
	`null`,
	`true`,
	`{"cwcreative":1234}`,
	`{"placementId":"abc"}`,
	`{"cwdebug":"TRUE"}`,
	`{"cwdebug":"FALSE"}`,
	`{"cwfeatures":[1,2,3]}`,
}
