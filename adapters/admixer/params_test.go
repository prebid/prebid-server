package admixer

import (
	"encoding/json"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file actually intends to test static/bidder-params/admixer.json
//
// These also validate the format of the external API: request.imp[i].ext.admixer

// TestValidParams makes sure that the admixer schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderAdmixer, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected admixer params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the admixer schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderAdmixer, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"zone": "9FF668A2-4122-462E-AAF8-36EA3A54BA21"}`,
	`{"zone": "9ff668a2-4122-462e-aaf8-36ea3a54ba21"}`,
	`{"zone": "9FF668A2-4122-462E-AAF8-36EA3A54BA21", "customFloor": 0.1}`,
	`{"zone": "9FF668A2-4122-462E-AAF8-36EA3A54BA21", "customParams": {"foo": "bar"}}`,
	`{"zone": "9ff668a2-4122-462e-aaf8-36ea3a54ba21", "customFloor": 0.1, "customParams": {"foo": ["bar", "baz"]}}`,
	`{"zone": "9FF668A24122462EAAF836EA3A54BA21"}`,
	`{"zone": "9FF668A24122462EAAF836EA3A54BA212"}`,
}

var invalidParams = []string{
	`{"zone": "123"}`,
	`{"zone": ""}`,
	`{"zone": "ZFF668A2-4122-462E-AAF8-36EA3A54BA21"}`,
	`{"zone": "9FF668A2-4122-462E-AAF8-36EA3A54BA211"}`,
	`{"zone": "123", "customFloor": "0.1"}`,
	`{"zone": "9FF668A2-4122-462E-AAF8-36EA3A54BA21",  "customFloor": -0.1}`,
	`{"zone": "9FF668A2-4122-462E-AAF8-36EA3A54BA21",  "customParams": "foo: bar"}`,
	`{"zone": "9FF668A24122462EAAF836EA3A54BA2"}`,
	`{"zone": "9FF668A24122462EAAF836EA3A54BA2112336"}`,
}
