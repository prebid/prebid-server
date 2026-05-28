package relevantdigital

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderRelevantDigital, json.RawMessage(p)); err != nil {
			t.Errorf("Schema rejected valid params: %s", p)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRelevantDigital, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId" : "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost" : "some-host" }`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId" : "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost" : "some-host", "useSourceBidderCode": true }`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId" : "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost" : "some-host", "useSourceBidderCode": false }`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId" : "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost" : "https://foo.relevant-digital.com" }`,
}

var invalidParams = []string{
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId" : "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost" : "127.0.0.1:6060/debug/pprof#" }`,
	`{"accountId": 123, "placementId" : 123, "pbsHost" : ""}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId" : "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost" : "some-host", "useSourceBidderCode": "somethingInvalid" }`,
}
