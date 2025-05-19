package matterfull

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
		if err := validator.Validate(openrtb_ext.BidderMatterfull, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected Matterfull params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderMatterfull, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"pid": "LUN2gcJFHRwysZVTm8p3"}`,
}

var invalidParams = []string{
<<<<<<< HEAD
	`{"publisher": "34563434"}`,
=======
	`{"publisher": "19f1b372c7548ec1fe734d2c9f8dc688"}`,
>>>>>>> da4549a5 (New Adapter: Matterfull)
	`nil`,
	``,
	`[]`,
	`true`,
}
