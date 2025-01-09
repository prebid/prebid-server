package vox

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderVox, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderVox, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"placementId": "64be6fe6685a271d37e900d2"}`,
	`{"placementId": "Any String Basically"}`,
	`{"placementId":""}`,
	`{"placementId":"id", "imageUrl":"http://site.com/img1.png"}`,
	`{"placementId":"id", "imageUrl":"http://site.com/img1.png", "displaySizes":["123x90", "1x1", "987x1111"]}`,
}

var invalidParams = []string{
	`{"placementId": 42}`,
	`{"placementId": null}`,
	`{"placementId": 3.1415}`,
	`{"placementId": true}`,
	`{"placementId": false}`,
	`{"placementId":"id", "imageUrl": null}`,
	`{"placementId":"id", "imageUrl": true}`,
	`{"placementId":"id", "imageUrl": []}`,
	`{"placementId":"id", "imageUrl": "http://some.url", "displaySizes": null}`,
	`{"placementId":"id", "imageUrl": "http://some.url", "displaySizes": {}}`,
	`{"placementId":"id", "imageUrl": "http://some.url", "displaySizes": "String"}`,
}
