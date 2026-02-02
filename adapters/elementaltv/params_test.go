package elementaltv

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoErrorf(t, err, "Failed to fetch the json schema: %v", err)
	for _, p := range validParams {
		err := validator.Validate(openrtb_ext.BidderElementalTV, json.RawMessage(p))
		require.NoError(t, err, "Schema rejected valid params: %s", p)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoErrorf(t, err, "Failed to fetch the json schema: %v", err)

	for _, p := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderElementalTV, json.RawMessage(p))
		require.Error(t, err, "Schema allowed invalid params: %s", p)
	}
}

var validParams = []string{
	`{"adunit":"123"}`,
	`{"adunit":"SSP:123"}`,
	`{"adunit":"5346"}`}

var invalidParams = []string{
	`{}`,
	`{"adunitt":"123"}`,
}
