package connectad

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderConnectAd, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected ConnectAd params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderConnectAd, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"siteId": 123456, "networkId": 123456, "bidfloor": 0.10}`,
	`{"siteId": 123456, "networkId": 123456}`,
}

var invalidParams = []string{
	`{}`,
	`null`,
	`{"siteId": 123456, "networkId": "123456", "bidfloor": 0.10}`,
	`{"siteId": "123456", "networkId": 123456, "bidfloor": 0.10}`,
	`{"siteId": 123456, "networkId": 123456, "bidfloor": "0.10"}`,
	`{"siteId": "123456"}`,
	`{"networkId": 123456}`,
	`{"siteId": 123456}`,
	`{"invalid_param": "123"}`,
}
