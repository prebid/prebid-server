package sovrn

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range validParams {
		if err := validator.Validate(openrtb_ext.BidderSovrn, json.RawMessage(param)); err != nil {
			t.Errorf("Schema rejected sovrn params: %s", param)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")

	if err != nil {
		t.Fatalf("Failed to fetch json-schemas. %v", err)
	}

	for _, param := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSovrn, json.RawMessage(param)); err == nil {
			t.Errorf("Schema allowed sovrn params: %s", param)
		}
	}
}

var validParams = []string{
	`{"tagId":"1"}`,
	`{"tagid":"2"}`,
	`{"tagId":"1","bidfloor":"0.5"}`,
	`{"tagId":"1","bidfloor":0.5}`,
	`{"tagId":"1","bidfloor":"0.5", "adunitcode":"0.5"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`0`,
	`[]`,
	`{}`,
}
