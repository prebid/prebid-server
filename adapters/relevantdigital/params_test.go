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
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host.example.com"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host.example.com:8080"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host-example.test"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "Host.Example.com"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "localhost:3000"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "https://foo.relevant-digital.com"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "http://foo.relevant-digital.com"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"accountId": 123, "placementId": 123, "pbsHost": ""}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host", "useSourceBidderCode": "somethingInvalid"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "/path"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "//evil.com"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host/path"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host?query=1"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host#fragment"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "user@host"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host:notaport"}`,
	`{"accountId": "5fcf49f83a64ba6602b5be7e", "placementId": "63b68275b4f35962c8eec9b1_5fcf49f83a64ba6602b5be9a", "pbsHost": "host:8080:extra"}`,
}
