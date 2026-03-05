package rtbstack

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
		if err := validator.Validate(openrtb_ext.BidderRTBStack, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected rtbstack params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRTBStack, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"route":"https://testsite.us-adx-admixer.rtb-stack.com/prebid?client=c4527281-5aa5-4c8e-bc53-a80bb3f99470&endpoint=309&ssp=145","tagId":"12345"}`,
	`{"route":"https://site.eu-adx-admixer.rtb-stack.com/prebid?client=abc&endpoint=1&ssp=2","tagId":"tag1"}`,
	`{"route":"https://site.asia-adx-admixer.rtb-stack.com/prebid?client=abc&endpoint=1&ssp=2","tagId":"tag1","customParams":{"foo":"bar"}}`,
	`{"route":"https://example.us-adx-admixer.rtb-stack.com/prebid?client=abc&endpoint=1&ssp=2","tagId":"tag1","customParams":{}}`,
	`{"route":"https://example.us-adx-admixer.rtb-stack.com/prebid?client=abc&endpoint=1&ssp=2","tagId":"tag1","customParams":{"nested":{"key":"value"}}}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"route":"https://testsite.us-adx-admixer.rtb-stack.com/prebid?client=abc&endpoint=1&ssp=2"}`,
	`{"tagId":"12345"}`,
	`{"route":"","tagId":"12345"}`,
	`{"route":123,"tagId":"12345"}`,
	`{"route":"https://testsite.us-adx-admixer.rtb-stack.com/prebid?client=abc&endpoint=1&ssp=2","tagId":123}`,
}
