package ucfunnel

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file actually intends to test static/bidder-params/ucfunnel.json
//
// These also validate the format of the external API: request.imp[i].ext.ucfunnel

// TestValidParams makes sure that the ucfunnel schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderUcfunnel, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected ucfunnel params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the ucfunnel schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderUcfunnel, json.RawMessage(invalidParam)); err != nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"adunitid": "ad-83444226E44368D1E32E49EEBE6D29","partnerid": "par-2EDDB423AA24474188B843EE4842932"}`,
}

var invalidParams = []string{
	`{"adunitid": "","partnerid": ""}`,
}
