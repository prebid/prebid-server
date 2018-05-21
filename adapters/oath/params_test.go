package oath

import (
	"github.com/mxmCherry/openrtb"
	"github.com/prebid/prebid-server/openrtb_ext"
	"testing"
)

// This file actually intends to test static/bidder-params/oath.json
//
// These also validate the format of the external API: request.imp[i].ext.oath

// TestValidParams makes sure that the Oath schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderOath, openrtb.RawJSON(validParam)); err != nil {
			t.Errorf("Schema rejected oath params: %s", validParam)
		}
	}
}

// TestInvalidParams makes sure that the Oath schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderOath, openrtb.RawJSON(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"publisherId": "123", "publisherName": "foo.bar"}`,
	`{"publisherName": "abc", "headerbidding": false}`,
	`{"publisherId": "xxx", "publisherName": "xyz123", "headerbidding": true}`,
	`{"publisherId": "abc123", "publisherName": "testname", "headerbidding": false}`,
	`{"publisherName": "testpublisher"}`,
}

var invalidParams = []string{
	`{"publisherId": "123"}`,
	`{"publisherName": 100}`,
	`{"publisherId": "123", "headerbidding": false}`,
	`{"publisherId": "123", "publisherName": true}`,
	`{"publisherId": 111, "publisherName": "foo.bar"}`,
	`{"publisherName": 123, "headerbidding": true}`,
	`{"publisherName": "test", "headerbidding": "false"}`,
	`{"publisherName": "test", "publisherId": false}`,
	`{"publisherId": "123", "publisherName": 1, "headerbidding": true}`,
	`{"publisherId": 123, "publisherName": "foo.bar", "headerbidding": false}`,
	`{"publisherId": "123", "publisherName": "foo.bar", "headerbidding": "true"}`,
}
