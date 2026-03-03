package bidmatic

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/bidmatic.json
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.bidmatic
// TestValidParams makes sure that the bidmatic schema accepts all imp.ext fields which we intend to support.

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBidmatic, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected bidmatic params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the bidmatic schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBidmatic, json.RawMessage(invalidParam)); err == nil {
			ext := openrtb_ext.ExtImpBidmatic{}
			err = json.Unmarshal([]byte(invalidParam), &ext)
			if err == nil {
				t.Errorf("Schema allowed unexpected params: %s", invalidParam)
			}
		}
	}
}

var validParams = []string{
	`{"source":123}`,
	`{"source":"123"}`,
	`{"source":123,"placementId":1234}`,
	`{"source":123,"siteId":4321}`,
	`{"source":"123","siteId":0,"bidFloor":0}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"source":"qwerty"}`,
	`{"source":"123","placementId":"123"}`,
	`{"source":123, "placementId":"123", "siteId":"321"}`,
}
