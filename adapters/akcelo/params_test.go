package akcelo

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema. %v", err)

	for _, p := range validParams {
		err := validator.Validate(openrtb_ext.BidderAkcelo, json.RawMessage(p))
		assert.NoError(t, err, "Schema rejected valid params: %s", p)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema. %v", err)

	for _, p := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderAkcelo, json.RawMessage(p))
		assert.Error(t, err, "Schema allowed invalid params: %s", p)
	}
}

var validParams = []string{
	`{"adUnitId": 123, "siteId": 456}`,
	`{"adUnitId": 123, "siteId": 456, "test": 0}`,
	`{"adUnitId": 123, "siteId": 456, "test": 1}`,
}

var invalidParams = []string{
	`{"adUnitId": 123}`,
	`{"siteId": 456}`,
	`{"siteId": 456, "test": 1}`,
}
