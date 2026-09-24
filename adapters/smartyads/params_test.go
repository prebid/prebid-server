package smartyads

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{ "host": "host", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host.example.com", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host.example.com:8080", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host-example.test", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "Host.Example.com", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "localhost:3000", "sourceid": "partner", "accountid": "hash" }`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSmartyAds, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected SmartyAds params: %s", validParam)
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
	`{"adCode": "string", "seatCode": 5, "originalPublisherid": "string"}`,
	`{ "host": "ep1", "sourceid": "partner" }`,
	`{ "host": "ep1", "accountid": "hash" }`,
	`{ "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "", "sourceid": "", "accountid": "" }`,
	`{ "host": "/path", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "//evil.com", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host/path", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host?query=1", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host#fragment", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "user@host", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "https://host.com", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host:notaport", "sourceid": "partner", "accountid": "hash" }`,
	`{ "host": "host:8080:extra", "sourceid": "partner", "accountid": "hash" }`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSmartyAds, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
