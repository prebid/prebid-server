package adnuntius

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/adnuntius.json
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.adnuntius
// TestValidParams makes sure that the adnuntius schema accepts all imp.ext fields which we intend to support.

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdnuntius, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adnuntius params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the adnuntius schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdnuntius, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"auId":"123"}`,
	`{"auId":"123", "network":"test"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"auId":123}`,
	`{"auID":"123"}`,
	`{"network":123}`,
	`{"network":123, "auID":123}`,
	`{"network":"test", "auID":123}`,
	`{"network":test, "auID":"123"}`,
}
