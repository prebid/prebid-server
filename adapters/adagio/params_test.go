package adagio

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

var validParams = []string{
	`{ "organizationId": "1234", "site": "adagio-io", "placement": "ban_atf" }`,
	`{ "organizationId": "1234", "site": "adagio-io", "placement": "ban_atf", "_unknown": "ban_atf"}`,
	`{ "organizationId": "1234", "site": "adagio-io", "placement": "ban_atf", "features": {"a": "a", "b": "b"}}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdagio, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Adagio params: %s", validParam)
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
	`{ "organizationId": "", "site": "", "placement": "" }`,
	`{ "organizationId": "", "site": "2", "placement": "3" }`,
	`{ "organizationId": "1", "site": "", "placement": "3" }`,
	`{ "organizationId": "1", "site": "2", "placement": "" }`,
	`{ "organizationId": 1, "site": "2", "placement": "3" }`,
	`{ "organizationId": "1234", "site": "adagio-io", "placement": "ban_atf", "features": {"a": "a", "notastring": true}}`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdagio, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
