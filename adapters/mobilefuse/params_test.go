package mobilefuse

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

func TestValidParams(test *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if err != nil {
		test.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		err := validator.Validate(openrtb_ext.BidderMobilefuse, json.RawMessage(validParam))

		if err != nil {
			test.Errorf("Schema rejected mobilefuse params: %s", validParam)
		}
	}
}

func TestInvalidParams(test *testing.T) {
	validator, errs := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if errs != nil {
		test.Fatalf("Failed to fetch the json-schemas. %v", errs)
	}

	for _, invalidParam := range invalidParams {
		errs := validator.Validate(openrtb_ext.BidderMobilefuse, json.RawMessage(invalidParam))

		if errs == nil {
			test.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placement_id":123,"pub_id":456}`,
	`{"placement_id":123,"pub_id":456,"tagid_src":"ext"}`,
	`{"placement_id":123, "pub_id":456, "tagid_src":""}`,
}

var invalidParams = []string{
	`{"placement_id":123}`,
	`{"pub_id":456}`,
	`{"placement_id":"123","pub_id":"456"}`,
	`{"placement_id":123, "placementId":123}`,
	`{"tagid_src":"ext"}`,
}
