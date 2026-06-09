package bidmachine

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderBidmachine, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Bidmachine params: %s \n Error: %s", validParam, err)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderBidmachine, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"seller_id":"1", "host":"host", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host.example.com", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host.example.com:8080", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host-example.test", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"Host.Example.com", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"localhost:3000", "path":"auction/rtb/v2"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"some_random_field":""}`,
	`{"seller_id":"", "host":"host", "path":"auction/rtb/v2"}`,
	`{"host":"host", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host", "path":""}`,
	`{"seller_id":"1", "host":"host"}`,
	`{"seller_id":"1", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"/path", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"//evil.com", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host/path", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host?query=1", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host#fragment", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"user@host", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"https://host.com", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host:notaport", "path":"auction/rtb/v2"}`,
	`{"seller_id":"1", "host":"host:8080:extra", "path":"auction/rtb/v2"}`,
}
