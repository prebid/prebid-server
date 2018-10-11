package rhythmone

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, validParam := range validParams {
		if err := validator.Validate(openrtb_ext.BidderRhythmone, json.RawMessage(validParam)); err != nil {
			t.Errorf("Schema rejected rhythmone params: %s", validParam)
		}
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json-schemas. %v", err)
	}

	for _, invalidParam := range invalidParams {
		if err := validator.Validate(openrtb_ext.BidderRhythmone, json.RawMessage(invalidParam)); err == nil {
			t.Errorf("Schema allowed unexpected params: %s", invalidParam)
		}
	}
}

var validParams = []string{
	`{"placementId":"123", "zone":"12345", "path":"34567"}`,
}

var invalidParams = []string{
	`{"placementId":"123", "zone":"12345", "path":34567}`,
	`{"placementId":"123", "zone":12345, "path":"34567"}`,
	`{"placementId":123, "zone":"12345", "path":"34567"}`,
	`{"placementId":123, "zone":12345, "path":34567}`,
	`{"placementId":123, "zone":12345, "path":"34567"}`,
	`{"appId":"123", "bidfloor":0.01}`,
	`{"publisherName": 100}`,
	`{"placementId": 1234}`,
	`{"zone": true}`,
	``,
	`null`,
	`nil`,
	`true`,
	`9`,
	`[]`,
	`{}`,
}
