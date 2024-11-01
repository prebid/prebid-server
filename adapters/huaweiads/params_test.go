package huaweiads

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
		if err := validator.Validate(openrtb_ext.BidderHuaweiAds, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected huaweiads params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderGumGum, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"slotid": "m8x9x3rzff","adtype": "banner","publisherid": "123","signkey": "2f910deac52ff34f0d80585d8664c55e3422ff3c6aeb5e1cf2ff94f1ac6a9642","keyid": "41","clienttime": "2018-11-02 16:34:07.981+1300"}`,
	`{"slotid": "m8x9x3rzff","adtype": "banner","publisherid": "123","signkey": "2f910deac52ff34f0d80585d8664c55e3422ff3c6aeb5e1cf2ff94f1ac6a9642","keyid": "41"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	``,
	`{}`,
	`[]`,
	`true`,
	`2`,
	`{"slotid": "","adtype": "banner","publisherid": "123","signkey": "2f910deac52ff34f0d80585d8664c55e3422ff3c6aeb5e1cf2ff94f1ac6a9642","keyid": "41","clienttime": "2018-11-02 16:34:07.981+1300"}`,
	`{"slotid": "m8x9x3rzff","adtype": "","publisherid": "123","signkey": "2f910deac52ff34f0d80585d8664c55e3422ff3c6aeb5e1cf2ff94f1ac6a9642","keyid": "41","clienttime": "2018-11-02 16:34:07.981+1300"}`,
	`{"slotid": "m8x9x3rzff","adtype": "banner","publisherid": "","signkey": "2f910deac52ff34f0d80585d8664c55e3422ff3c6aeb5e1cf2ff94f1ac6a9642","keyid": "41","clienttime": "2018-11-02 16:34:07.981+1300"}`,
	`{"slotid": "m8x9x3rzff","adtype": "banner","publisherid": "123","signkey": "","keyid": "41","clienttime": "2018-11-02 16:34:07.981+1300"}`,
	`{"slotid": "m8x9x3rzff","adtype": "banner","publisherid": "123","signkey": "2f910deac52ff34f0d80585d8664c55e3422ff3c6aeb5e1cf2ff94f1ac6a9642","keyid": "","clienttime": "2018-11-02 16:34:07.981+1300"}`,
}
