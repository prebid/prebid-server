package aidem

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// This file actually intends to test static/bidder-params/aidem.json TODO: MUST BE CREATED
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAidem, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected aidem params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAidem, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"siteId":"123", "publisherId":"1234"}`,
	`{"siteId":"123", "publisherId":"1234", "placementId":"12345"}`,
	`{"siteId":"123", "publisherId":"1234", "rateLimit":1}`,
	`{"siteId":"123", "publisherId":"1234", "placementId":"12345", "rateLimit":1}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"siteId":"", "publisherId":""}`,
	`{"siteId":"only siteId is present"}`,
	`{"publisherId":"only publisherId is present"}`,
	`{"ssiteId":"123","ppublisherId":"123"}`,
	`{"aid":123, "placementId":"123", "siteId":"321"}`,
}
