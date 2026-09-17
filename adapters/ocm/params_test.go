package ocm

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

// This file intends to test static/bidder-params/ocm.json
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.ocm

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOcm, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected valid params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOcm, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId":"ocm-pub-1","placementId":"homepage-top"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"placementId":"homepage-top"}`,
	`{"publisherId":"ocm-pub-1"}`,
	`{"publisherId":123}`,
	`{"publisherId":""}`,
	`{"publisherId":"ocm-pub-1","placementId":42}`,
	`{"publisherId":"ocm-pub-1","placementId":""}`,
}
