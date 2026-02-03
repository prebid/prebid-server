package trustx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
)

// This file actually intends to test static/bidder-params/trustx.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.trustx

// TestValidParams makes sure that the trustx schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, validParam := range validParams {
		err := validator.Validate(openrtb_ext.BidderTrustX, json.RawMessage(validParam))
		require.NoError(t, err, "Schema rejected trustx params: %s", validParam)
	}
}

// TestInvalidParams makes sure that the trustx schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, invalidParam := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderTrustX, json.RawMessage(invalidParam))
		require.Error(t, err, "Schema allowed unexpected params: %s", invalidParam)
	}
}

var validParams = []string{
	`{}`,
	`{"uid": 1234}`,
	`{"uid": 1234, "keywords":{"site": {}, "user": {}}}`,
}
var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{"uid": "invalid_type"}`,
	`{"keywords": "invalid_type"}`,
}
