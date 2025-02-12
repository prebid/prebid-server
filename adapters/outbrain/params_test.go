package outbrain

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
		if err := validator.Validate(openrtb_ext.BidderOutbrain, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderOutbrain, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisher": {"id": "publisher-id"}}`,
	`{"publisher": {"id": "publisher-id", "name": "publisher-name", "domain": "publisher-domain.com"}, "tagid": "tag-id", "bcat": ["bad-category"], "badv": ["bad-advertiser"]}`,
}

var invalidParams = []string{
	`{"publisher": {"id": 1234}}`,
	`{"publisher": {"id": "pub-id", "name": 1234}}`,
	`{"publisher": {"id": "pub-id", "domain": 1234}}`,
	`{"publisher": {"id": "pub-id"}, "tagid": 1234}`,
	`{"publisher": {"id": "pub-id"}, "bcat": "not-array"}`,
	`{"publisher": {"id": "pub-id"}, "bcat": [1234]}`,
	`{"publisher": {"id": "pub-id"}, "badv": "not-array"}`,
	`{"publisher": {"id": "pub-id"}, "badv": [1234]}`,
}
