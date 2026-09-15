package seedtag

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSeedtag, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSeedtag, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"adUnitId": "27604970"}`,
	`{"publisherId": "abc123", "integrationType": "ronId"}`,
}

var invalidParams = []string{
	`{"adUnitId": 123}`,
	`{"adUnitId": ""}`,
	`{}`,
	`{"publisherId": "abc123"}`,
	`{"integrationType": "ronId"}`,
	`{"publisherId": "", "integrationType": "ronId"}`,
	`{"publisherId": "abc123", "integrationType": "unknown"}`,
	`{"adUnitId": "27604970", "publisherId": "abc123", "integrationType": "ronId"}`,
}
