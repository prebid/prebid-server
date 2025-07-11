package oprx

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
		if err := validator.Validate(openrtb_ext.BidderOprx, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderOprx, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"key": "some key", "placement_id": 1234567890, "type": "banner"}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "width": 1234}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "height": 5678}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "bid_floor": 1.23}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "npi": "some npi"}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "ndc": "some ndc"}`,
}

var invalidParams = []string{
	`{"key": 1234567, "placement_id": 1234567890, "type": "banner"}`,
	`{"placement_id": "1234567890", "key": "some key", "type": "banner"}`,
	`{"key": "some key", "placement_id": 1234567890, "type": 12}`,
	`{"key": "some key"}`,
	`{"placement_id": 1234567890, "type": 12}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "width": "1234"}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "height": "5678"}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "bid_floor": "1.23"}`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "npi": 1223456`,
	`{"key": "some key", "placement_id": 1234567890, "type": "banner", "ndc": 876543}`,
}

// Key         string  `json:"key"`
// 	PlacementID int     `json:"placement_id"`
// 	Width       int     `json:"width"`
// 	Height      int     `json:"height"`
// 	BidFloor    float64 `json:"bid_floor"`
// 	Npi         string  `json:"npi"`
// 	Ndc         string  `json:"ndc"`
// 	Type        string  `json:"type"`
