package acuityads

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

var validParams = []string{
	`{ "host": "host", "accountid": "hash" }`,
	`{ "host": "host.example.com", "accountid": "hash" }`,
	`{ "host": "host.example.com:8080", "accountid": "hash" }`,
	`{ "host": "host-example.test", "accountid": "hash" }`,
	`{ "host": "Host.Example.com", "accountid": "hash" }`,
	`{ "host": "localhost:3000", "accountid": "hash" }`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAcuityAds, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected AcuityAds params: %s", validParam)
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
	`{ "accountid": "hash" }`,
	`{ "host": "", "accountid": "" }`,
	`{ "host": "/path", "accountid": "hash" }`,
	`{ "host": "//evil.com", "accountid": "hash" }`,
	`{ "host": "host/path", "accountid": "hash" }`,
	`{ "host": "host?query=1", "accountid": "hash" }`,
	`{ "host": "host#fragment", "accountid": "hash" }`,
	`{ "host": "user@host", "accountid": "hash" }`,
	`{ "host": "https://host.com", "accountid": "hash" }`,
	`{ "host": "host:notaport", "accountid": "hash" }`,
	`{ "host": "host:8080:extra", "accountid": "hash" }`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAcuityAds, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
