package optout

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatal(err)
	}
	valid := []string{
		`{"publisher":"152","adSlot":"home"}`,
		`{"publisher":"152","adSlot":"home","id":"x","includeOrtb2":true,"customs":{"section":"news","number":3}}`,
	}
	for _, p := range valid {
		if err := validator.Validate("optout", json.RawMessage(p)); err != nil {
			t.Errorf("valid params rejected %s: %v", p, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatal(err)
	}
	invalid := []string{
		`{}`, `{"publisher":"152"}`, `{"adSlot":"home"}`,
		`{"publisher":"","adSlot":"home"}`, `{"publisher":"152","adSlot":""}`,
		`{"publisher":123,"adSlot":"home"}`, `{"publisher":"152","adSlot":123}`,
		`{"publisher":"152","adSlot":"home","includeOrtb2":"yes"}`,
		`{"publisher":"152","adSlot":"home","customs":"not an object"}`,
	}
	for _, p := range invalid {
		if err := validator.Validate("optout", json.RawMessage(p)); err == nil {
			t.Errorf("invalid params accepted: %s", p)
		}
	}
}
