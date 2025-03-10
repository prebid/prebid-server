package mobkoi

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(test *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if err != nil {
		test.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		err := validator.Validate(openrtb_ext.BidderMobkoi, json.RawMessage(validParam))

		if err != nil {
			test.Errorf("Schema rejected Mobkoi params: %s\nError: %v", validParam, err)
		}
	}
}

func TestInvalidParams(test *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if err != nil {
		test.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderMobkoi, json.RawMessage(invalidParam))

		if err == nil {
			test.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId":"abc"}`,
	`{"placementId":"abc", "adServerBaseUrl":"mobkoi.com"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`1`,
	`1.0`,
	`[]`,
	`{}`,
	`{"placement_id":"abc"}`,
	`{"placementId":123, "adServerBaseUrl":"mobkoi.com"}`,
}
