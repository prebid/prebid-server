package revantage

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas: %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderRevantage, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected revantage params that should be valid: %s\nError: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas: %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRevantage, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema accepted revantage params that should be invalid: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"feedId":"feed-abc"}`,
	`{"feedId":"feed-abc","placementId":"plc-1"}`,
	`{"feedId":"feed-abc","placementId":"plc-1","publisherId":"pub-1"}`,
	`{"feedId":"x"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"placementId":"plc-1"}`,
	`{"feedId":""}`,
	`{"feedId":123}`,
	`{"feedId":null}`,
	`{"feedId":"feed-abc","placementId":42}`,
}
