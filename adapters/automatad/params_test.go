package automatad

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAutomatad, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected valid params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAutomatad, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"position": "123abc"}`,
	`{"placementId": "a34gh6d"}`,
	`{"position": "123abc", "placementId" : "a34gh6d"}`,
}

var invalidParams = []string{
	`{"position": 123abc}`,
	`"placementId" : 46}`,
	`{"position": "123abc", "placementId" : 46}`,
	`{"position": 100, "placementId" : "a34gh6d"}`,
	`{"position": 100, "placementId" : 200}`,
}
