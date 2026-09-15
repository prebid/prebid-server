package zeroclickfraud

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{ "host": "host", "sourceId": 1 }`,
	`{ "host": "host.example.com", "sourceId": 1 }`,
	`{ "host": "host.example.com:8080", "sourceId": 1 }`,
	`{ "host": "host-example.test", "sourceId": 1 }`,
	`{ "host": "Host.Example.com", "sourceId": 1 }`,
	`{ "host": "localhost:3000", "sourceId": 1 }`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderZeroClickFraud, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected ZeroClickFraud params: %s", validParam)
		}
	}
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{ "sourceId": 1 }`,
	`{ "host": "host" }`,
	`{ "host": "", "sourceId": 1 }`,
	`{ "host": "/path", "sourceId": 1 }`,
	`{ "host": "//evil.com", "sourceId": 1 }`,
	`{ "host": "host/path", "sourceId": 1 }`,
	`{ "host": "host?query=1", "sourceId": 1 }`,
	`{ "host": "host#fragment", "sourceId": 1 }`,
	`{ "host": "user@host", "sourceId": 1 }`,
	`{ "host": "https://host.com", "sourceId": 1 }`,
	`{ "host": "host:notaport", "sourceId": 1 }`,
	`{ "host": "host:8080:extra", "sourceId": 1 }`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderZeroClickFraud, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
