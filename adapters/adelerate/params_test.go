package adelerate

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{"placementId":"123", "publisherId":"456"}`,
	`{"placementId":"abc", "publisherId":"def", "floor": 1.5, "floorCurrency": "EUR"}`,
	`{"placementId":"abc", "publisherId":"def", "floor": 0.5}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"placementId":"123"}`,
	`{"publisherId":"456"}`,
	`{"placementId":"", "publisherId":"456"}`,
	`{"placementId":"123", "publisherId":""}`,
	`{"some": "param"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdelerate, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adelerate params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdelerate, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
