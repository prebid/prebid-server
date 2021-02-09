package unicorn

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderUnicorn, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderUnicorn, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{
      "accountId": 199578,
      "publisherId": 123456,
      "mediaId": "test_media",
      "placementId": "test_placement"
   }`,
	`{
      "accountId": 199578,
      "mediaId": "test_media"
   }`,
}

var invalidParams = []string{
	`{}`,
	`{
      "accountId": "199578",
      "publisherId": "123456",
      "mediaId": 12345,
      "placementId": 12345
   }`,
	`{
      "publisherId": 123456,
      "placementId": "test_placement"
   }`,
}
