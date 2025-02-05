package yandex

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	if err != nil {
		t.Fatalf("Failed to fetch the json schema. %v", err)
	}

	for _, p := range validParams {
		if err := validator.Validate(openrtb_ext.BidderYandex, json.RawMessage(p)); err != nil {
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
		if err := validator.Validate(openrtb_ext.BidderYandex, json.RawMessage(p)); err == nil {
			t.Errorf("Schema allowed invalid params: %s", p)
		}
	}
}

var validParams = []string{
	`{"page_id": 123123, "imp_id": 123}`,
	`{"placement_id": "A-123123-123"}`,
	`{"placement_id": "B-A-123123-123"}`,
	`{"placement_id": "123123-123"}`,
}

var invalidParams = []string{
	`{"pageId": 123123, "impId": 123}`,
	`{"page_id": 0, "imp_id": 0}`,
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

func TestInvalidPlacementIdMapper(t *testing.T) {
	for _, ext := range invalidPlacementIds {
		_, err := mapExtToPlacementID(ext)

		assert.Error(t, err)
	}
}

var validPlacementIds = map[openrtb_ext.ExtImpYandex]yandexPlacementID{
	{PlacementID: "A-12345-1"}:      {PageID: "12345", ImpID: "1"},
	{PlacementID: "B-A-123123-123"}: {PageID: "123123", ImpID: "123"},
	{PlacementID: "111-222"}:        {PageID: "111", ImpID: "222"},
	{PageID: 111, ImpID: 222}:       {PageID: "111", ImpID: "222"},
}

var invalidPlacementIds = []openrtb_ext.ExtImpYandex{
	{PlacementID: "123123"},
	{PlacementID: "A-123123"},
	{PlacementID: "B-A-123123"},
	{PlacementID: "C-B-A-123123"},
}
