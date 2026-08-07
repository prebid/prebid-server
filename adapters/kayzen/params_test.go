package kayzen

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{ "zone": "dc", "exchange": "ex" }`,
	`{ "zone": "host", "exchange": "ex" }`,
	`{ "zone": "host.example.com", "exchange": "ex" }`,
	`{ "zone": "host-example.test", "exchange": "ex" }`,
	`{ "zone": "Host.Example.com", "exchange": "ex" }`,
	`{ "zone": "localhost", "exchange": "ex" }`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderKayzen, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Kayzen params: %s", validParam)
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
	`{ "key": 2 }`,
	`{ "anyparam": "anyvalue" }`,
	`{ "zone": "dc" }`,
	`{ "exchange": "ex" }`,
	`{ "exchange": "", "zone" : "" }`,
	`{ "exchange": "ex", "zone" : "" }`,
	`{ "exchange": "", "zone" : "dc" }`,
	`{ "zone": "/path", "exchange": "ex" }`,
	`{ "zone": "//evil.com", "exchange": "ex" }`,
	`{ "zone": "host/path", "exchange": "ex" }`,
	`{ "zone": "host?query=1", "exchange": "ex" }`,
	`{ "zone": "host#fragment", "exchange": "ex" }`,
	`{ "zone": "user@host", "exchange": "ex" }`,
	`{ "zone": "https://host.com", "exchange": "ex" }`,
	`{ "zone": "host:notaport", "exchange": "ex" }`,
	`{ "zone": "host:8080:extra", "exchange": "ex" }`,
	`{ "zone": "host:8080", "exchange": "ex" }`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderKayzen, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
