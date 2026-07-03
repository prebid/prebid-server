package superedge

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
		if err := validator.Validate(openrtb_ext.BidderSuperEdge, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected superEdge params: %s with err: %v", validParam, err)
		}
	}
}

// TestInvalidParams makes sure that the superEdge schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderSuperEdge, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"sk": "7f096f84f44f4adfa7602f037179c98b"}`,
	`{"sk": "1e9ead5397ae44d78c6792bc7cddc050"}`,
	`{"sk": "27bb74d57068406ebcbb29ab9bfeb9b9"}`,
	`{"sk": "0c3356713c184ca186779eecdd5aff5d"}`,
}

var invalidParams = []string{
	`{}`,
	`{"tn": "0c3356713c184ca186779eecdd5aff5d"}`,
	`{"region": "APAC"}`,
	`{"region": "US"}`,
	`{"tn": "27bb74d57068406ebcbb29ab9bfeb9b9"}`,
}
