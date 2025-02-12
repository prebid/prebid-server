package adview

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdView, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adview params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdView, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{ "placementId": "posid00001", "accountId": "accountid01"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`{"adCode": "string", "seatCode": 5, "originalPublisherid": "string"}`,
	`{ "accountId": "accountid01" }`,
	`{ "placementId": "posid00001" }`,
	`{ "placementId": "", "accountId": "" }`,
}
