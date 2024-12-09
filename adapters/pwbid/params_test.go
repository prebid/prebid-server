package pwbid

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
		if err := validator.Validate(openrtb_ext.BidderPWBid, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderPWBid, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"siteId":"39f43a","bidFloor":0.10,"isTest":false}`,
	`{"siteId":"39f43a","bidFloor":0.10,"isTest":true}`,
	`{"siteId":"39f43a","bidFloor":0.10}`,
	`{"siteId":"39f43a"}`,
}

var invalidParams = []string{
	`{"siteId":42,"bidFloor":"asdf","isTest":123}`,
	`{"siteId":}`,
	`{"bidFloor":}`,
	`{"bidFloor":0.10}`,
	`null`,
}
