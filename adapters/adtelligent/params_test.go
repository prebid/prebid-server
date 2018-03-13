package adtelligent

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file actually intends to test static/bidder-params/adtelligent.json
// These also validate the format of the external API: request.imp[i].ext.adtelligent
// TestValidParams makes sure that the adtelligent schema accepts all imp.ext fields which we intend to support.

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdtelligent, openrtb.RawJSON(validParam)); err != nil {
			t.Errorf("Schema rejected adtelligent params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the adtelligent schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdtelligent, openrtb.RawJSON(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"aid":123}`,
	`{"aid":123,"placementId":1234}`,
	`{"aid":123,"siteId":4321}`,
	`{"aid":123,"siteId":0,"bidFloor":0}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"aid":"123"}`,
	`{"aid":"0"}`,
	`{"aid":"123","placementId":"123"}`,
	`{"aid":123, "placementId":"123", "siteId":"321"}`,
}
