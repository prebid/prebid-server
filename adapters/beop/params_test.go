package beop

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
		if err := validator.Validate(openrtb_ext.BidderBeop, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected adverxo params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBeop, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{ "pid": "aaaaaaaaaaaaaaaaaaaaaaaa"}`,
	`{ "nid": "aaaaaaaaaaaaaaaaaaaaaaaa", "ntpnid": "1234"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`{ "fid": 5 }`,
	`{ "pid": 5 }`,
	`{ "pid": 5 }`,
	`{ "nid": "", "nptnid": "" }`,
}
