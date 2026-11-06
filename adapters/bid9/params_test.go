package bid9

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
		if err := validator.Validate(openrtb_ext.BidderBid9, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderBid9, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"channelId": "6ad9d5605c6d395f530e7799"}`,
	`{"channelId": "a b&c"}`,
	`{"channelId": "c", "publisherId": "p"}`,
	`{"channelId": "c", "bidFloor": 0.5}`,
	`{"channelId": "c", "bidFloor": 0.5, "bidFloorCur": "EUR"}`,
	`{"channelId": "c", "publisherId": "p", "bidFloor": 1, "bidFloorCur": "USD"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`"c"`,
	`[]`,
	`{}`,
	`{"channelId": ""}`,
	`{"channelId": 123}`,
	`{"channelId": {"id": "c"}}`,
	`{"placementId": "c"}`,
	`{"publisherId": "p"}`,
	`{"channelId": "c", "publisherId": ""}`,
	`{"channelId": "c", "publisherId": 5}`,
	`{"channelId": "c", "bidFloor": 0}`,
	`{"channelId": "c", "bidFloor": -1}`,
	`{"channelId": "c", "bidFloor": "0.5"}`,
	`{"channelId": "c", "bidFloorCur": "usd"}`,
	`{"channelId": "c", "bidFloorCur": "EURO"}`,
	`{"channelId": "c", "bidFloorCur": 978}`,
}
