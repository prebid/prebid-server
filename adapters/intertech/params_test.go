package intertech

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderIntertech, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderIntertech, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"page_id": 123123, "imp_id": 123}`,
	// `{"placement_id": "A-123123-123"}`,
	// `{"placement_id": "B-A-123123-123"}`,
	`{"placement_id": "123123-123"}`,
}

var invalidParams = []string{
	`{"pageId": 123123, "impId": 123}`,
	`{"page_id": "123123", "imp_id": "123"}`,
	`{"page_id": "123123", "imp_id": "123", "placement_id": "123123"}`,
	`{"page_id": "123123"}`,
	`{"imp_id": "123"}`,
	`{"placement_id": 123123}`,
	`{"placement_id": "123123"}`,
	`{"placement_id": "A-123123"}`,
	`{"placement_id": "B-A-123123"}`,
	`{}`,
}

func TestValidPlacementIdMapper(t *testing.T) {
	for ext, expectedPlacementId := range validPlacementIds {
		val, err := mapExtToPlacementID(ext)

		assert.Equal(t, &expectedPlacementId, val)
		assert.NoError(t, err)
	}
}

var validPlacementIds = map[openrtb_ext.ExtImpIntertech]intertechPlacementID{
	{PlacementID: "111-222"}:  {PageID: "111", ImpID: "222"},
	{PageID: 111, ImpID: 222}: {PageID: "111", ImpID: "222"},
}
