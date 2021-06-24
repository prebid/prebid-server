package salunamedia

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

var validParams = []string{
	`{ "key": "2", "type": "network"}`,
	`{ "key": "1"}`,
	`{ "key": "33232", "type": "publisher"}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSaLunaMedia, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected sa_lunamedia params: %s", validParam)
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
	`{ "anyparam": "anyvalue" }`,
	`{ "type": "network" }`,
	`{ "key": "asddsfd", "type": "any"}`,
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSaLunaMedia, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}
