package adswag

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
		if err := validator.Validate(openrtb_ext.BidderAdswag, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderAdswag, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"publisherId":"pub-3b4912c7-b428-40b0-bbfb-30dbe888ceb8"}`,
	`{"publisherId":"prebid-test"}`,
	`{"publisherId":"prebid-test","placementId":"prebid-test-display"}`,
	// Client-side convenience params (bidFloor, video) may ride along from a
	// shared Prebid.js config; the schema tolerates them and the adapter
	// ignores them (floors and video params arrive in the OpenRTB request).
	`{"publisherId":"prebid-test","bidFloor":0.5}`,
}

var invalidParams = []string{
	``,
	`null`,
	`{}`,
	`{"publisherId":1}`,
	`{"publisherId":""}`,
	`{"placementId":"prebid-test-display"}`,
	`{"publisherId":"prebid-test","placementId":42}`,
	`{"publisherId":"prebid-test","placementId":""}`,
}
