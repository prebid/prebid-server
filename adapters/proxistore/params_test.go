package proxistore

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

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderProxistore, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderProxistore, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"website": "example.com", "language": "fr"}`,
	`{"website": "test-site", "language": "en"}`,
	`{"website": "publisher.be", "language": "nl"}`,
}

var invalidParams = []string{
	`{}`,
	`{"website": "example.com"}`,
	`{"language": "fr"}`,
	`{"website": "", "language": "fr"}`,
	`{"website": "example.com", "language": ""}`,
	`{"website": 123, "language": "fr"}`,
	`{"website": "example.com", "language": 456}`,
	`null`,
}
