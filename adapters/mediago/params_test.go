package mediago

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderMediaGo, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected mediago params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the mediago schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderMediaGo, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"token": "f9f2b1ef23fe2759c2cad0953029a94b"}`,
	`{"token": "f9f2b1ef23fe2759c2cad0953029a94b", "region": "APAC"}`,
	`{"token": "f9f2b1ef23fe2759c2cad0953029a94b", "region": "US"}`,
	`{"token": "f9f2b1ef23fe2759c2cad0953029a94b", "region": "EU"}`,
}

var invalidParams = []string{
	`{}`,
	`{"tn": "f9f2b1ef23fe2759c2cad0953029a94b"}`,
	`{"region": "APAC"}`,
	`{"region": "US"}`,
	`{"tn": "f9f2b1ef23fe2759c2cad0953029a94b", "region": "EU"}`,
}
