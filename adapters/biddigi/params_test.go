package biddigi

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// This file intends to test static/bidder-params/biddigi.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.biddigi

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBiddigi, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected biddigi params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBiddigi, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"seatKey": "bds_exampleseatkey"}`,
	`{"seatKey": "bds_exampleseatkey", "placementId": "biddigi-placement-1"}`,
}

var invalidParams = []string{
	`null`,
	`true`,
	`[]`,
	`{}`,
	`"bds_exampleseatkey"`,
	`{"placementId": "biddigi-placement-1"}`,
	`{"seatKey": ""}`,
	`{"seatKey": 12345}`,
	`{"seatKey": true}`,
	`{"seatKey": "bds_exampleseatkey", "placementId": 12345}`,
	`{"seatKey": "bds_exampleseatkey", "placementId": ""}`,
}
