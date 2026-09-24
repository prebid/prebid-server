package rediads

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
		if err := validator.Validate(openrtb_ext.BidderRediads, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderRediads, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"account_id": "123"}`,
	`{"account_id": "123", "slot": "12345"}`,
	`{"account_id": "123", "slot": "12345", "endpoint": "bidding2"}`,
	`{"account_id": "123", "endpoint": "host"}`,
	`{"account_id": "123", "endpoint": "host.example.com"}`,
	`{"account_id": "123", "endpoint": "host-example.test"}`,
	`{"account_id": "123", "endpoint": "Host.Example.com"}`,
	`{"account_id": "123", "endpoint": "localhost"}`,
}

var invalidParams = []string{
	`{"account": 42}`,
	`{}`,
	`{"account_id": "123", "endpoint": "/path"}`,
	`{"account_id": "123", "endpoint": "//evil.com"}`,
	`{"account_id": "123", "endpoint": "host/path"}`,
	`{"account_id": "123", "endpoint": "host?query=1"}`,
	`{"account_id": "123", "endpoint": "host#fragment"}`,
	`{"account_id": "123", "endpoint": "user@host"}`,
	`{"account_id": "123", "endpoint": "https://host.com"}`,
	`{"account_id": "123", "endpoint": "host:notaport"}`,
	`{"account_id": "123", "endpoint": "host:8080:extra"}`,
	`{"account_id": "123", "endpoint": "host:8080"}`,
}
