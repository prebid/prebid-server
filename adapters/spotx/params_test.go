package spotx

import (
	"encoding/json"
	"testing"

	"github.com/PubMatic-OpenWrap/prebid-server/openrtb_ext"
)

func TestSpotxParams(t *testing.T) {
	testValidParams(t)
	testInvalidParams(t)
}

func testValidParams(t *testing.T) {

	params := []string{
		`{"channel_id": "12345", "ad_unit": "instream"}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": true}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": true, "ad_volume": 0.4}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": true, "ad_volume": 0.4, "price_floor": 10}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": true, "ad_volume": 0.4, "price_floor": 10, "hide_skin": false}`,
	}
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Error loading json schema for spotx paramaters: %v", err)
	}

	for _, param := range params {
		if err := validator.Validate(openrtb_ext.BidderSpotX, json.RawMessage(param)); err != nil {
			t.Errorf("Params schema mismatch - %s: %v", param, err)
		}
	}
}

// TestInvalidParams makes sure that the 33Across schema rejects all the imp.ext fields we don't support.
func testInvalidParams(t *testing.T) {
	params := []string{
		`{"channel_id": "1234", "ad_unit": "instream", "secure": true, "ad_volume": 0.4, "price_floor": 10, "hide_skin": false}`,
		`{"channel_id": "12345", "ad_unit": "outstream1", "secure": true, "ad_volume": 0.4, "price_floor": 10, "hide_skin": false}`,
		`{"ad_unit": "instream", "secure": true, "ad_volume": 0.4, "price_floor": 10, "hide_skin": false}`,
		`{"channel_id": "12345", "secure": true, "ad_volume": 0.4, "price_floor": 10, "hide_skin": false}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": 1, "ad_volume": 0.4, "price_floor": 10, "hide_skin": false}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": true, "ad_volume": "0.4", "price_floor": 10, "hide_skin": false}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": true, "ad_volume": 0.4, "price_floor": 10.12, "hide_skin": false}`,
		`{"channel_id": "12345", "ad_unit": "instream", "secure": true, "ad_volume": 0.4, "price_floor": 10, "hide_skin": 0}`,
	}
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Error loading json schema for spotx paramaters: %v", err)
	}

	for _, param := range params {
		if err := validator.Validate(openrtb_ext.BidderSpotX, json.RawMessage(param)); err == nil {
			t.Errorf("Unexpexted params schema match - %s", param)
		}
	}
}
