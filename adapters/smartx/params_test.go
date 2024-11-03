package smartx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

var validParams = []string{
	`{"tagId":"Nu68JuOWAvrbzoyrOR9a7A", "publisherId":"11986", "siteId":"22860"}`,
	`{"tagId":"Nu68JuOWAvrbzoyrOR9a7A", "publisherId":"11986", "appId":"22860"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSmartx, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected smartx params: %s", validParam)
		}
	}
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"anyparam": "anyvalue"}`,
	`{"tagId":"Nu68JuOWAvrbzoyrOR9a7A"}`,
	`{"publisherId":"11986"}`,
	`{"siteId":"22860"}`,
	`{"appId":"22860"}`,
	`{"tagId":"Nu68JuOWAvrbzoyrOR9a7A", "publisherId":"11986"}`,
	`{"tagId":"Nu68JuOWAvrbzoyrOR9a7A", "siteId":"22860"}`,
	`{"tagId":"Nu68JuOWAvrbzoyrOR9a7A", "appId":"22860"}`,
	`{"publisherId":"11986", "appId":"22860"}`,
	`{"publisherId":"11986", "appId":"22860"}`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSmartHub, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
