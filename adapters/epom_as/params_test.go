package epom_as

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
		if err := validator.Validate(openrtb_ext.BidderEpomAs, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderEpomAs, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com:8080","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads-eu.example.co.uk","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","channel":"sports-uk"}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"section":"sport","tier":2,"premium":true}}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`[]`,
	`{}`,
	`{"host":"ads.example.com"}`,
	`{"placementKey":"a4f21c9e7b"}`,
	`{"host":"","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com","placementKey":""}`,
	`{"host":"ads.example.com","placementKey":42}`,
	`{"host":42,"placementKey":"a4f21c9e7b"}`,
	// A host must not be able to rewrite the outbound URL.
	`{"host":"https://ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com/collect","placementKey":"a4f21c9e7b"}`,
	`{"host":"user@ads.example.com","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com?x=1","placementKey":"a4f21c9e7b"}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":{"nested":{"a":1}}}`,
	`{"host":"ads.example.com","placementKey":"a4f21c9e7b","customParams":"not-an-object"}`,
}
