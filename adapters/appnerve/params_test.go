package appnerve

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"testing"
)

func TestParameters(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{`{"sourceId":"test-source"}`, `{"placementId":"test-source"}`} {
		if err := validator.Validate(openrtb_ext.BidderAppnerve, json.RawMessage(value)); err != nil {
			t.Fatalf("rejected %s: %v", value, err)
		}
	}
	for _, value := range []string{`{}`, `null`, `{"sourceId":42}`, `{"sourceId":""}`, `{"sourceId":"   "}`} {
		if err := validator.Validate(openrtb_ext.BidderAppnerve, json.RawMessage(value)); err == nil {
			t.Fatalf("accepted %s", value)
		}
	}
}
