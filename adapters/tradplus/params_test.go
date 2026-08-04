package tradplus

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderTradPlus, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected tradplus params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the tradplus schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderTradPlus, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"accountId": "11233"}`,
	`{"accountId": "11233", "zoneId": ""}`,
	`{"accountId": "11233", "zoneId": "host"}`,
	`{"accountId": "aaa", "zoneId": "host.example.com"}`,
	`{"accountId": "aa", "zoneId": "host-example.test"}`,
	`{"accountId": "aaa", "zoneId": "Host.Example.com"}`,
	`{"accountId": "aaa", "zoneId": "localhost"}`,
}

var invalidParams = []string{
	`{"accountId": ""}`,
	`{"accountId": "", "zoneId": ""}`,
	`{"accountId": "", "zoneId": "sin"}`,
	`{"accountId": 123}`,
	`{"accountId": {"test":1}}`,
	`{"accountId": true}`,
	`{"accountId": null}`,
	`{"zoneId": "aaa"}`,
	`{"zoneId": null}`,
	`{"accountId": "aaa", "zoneId": "/path"}`,
	`{"accountId": "aaa", "zoneId": "//evil.com"}`,
	`{"accountId": "aaa", "zoneId": "host/path"}`,
	`{"accountId": "aaa", "zoneId": "host?query=1"}`,
	`{"accountId": "aaa", "zoneId": "host#fragment"}`,
	`{"accountId": "aaa", "zoneId": "user@host"}`,
	`{"accountId": "aaa", "zoneId": "https://host.com"}`,
	`{"accountId": "aaa", "zoneId": "host:notaport"}`,
	`{"accountId": "aaa", "zoneId": "host:8080:extra"}`,
	`{"accountId": "aaa", "zoneId": "host:8080"}`,
}
