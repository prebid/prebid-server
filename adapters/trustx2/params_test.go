package trustx2

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
	"testing"
)

// This file actually intends to test static/bidder-params/trustx2.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.trustx2

// TestValidParams makes sure that the trustx2 schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, validParam := range validParams {
		err := validator.Validate(openrtb_ext.BidderTrustX2, json.RawMessage(validParam))
		require.NoError(t, err, "Schema rejected trustx2 params: %s", validParam)
	}
}

// TestInvalidParams makes sure that the trustx2 schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, invalidParam := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderTrustX2, json.RawMessage(invalidParam))
		require.Error(t, err, "Schema allowed unexpected params: %s", invalidParam)
	}
}

var validParams = []string{
	`{"publisher_id": "pub", "placement_id": "plcm"}`,
}
var invalidParams = []string{
	`{"publisher_id": "pub"}`,
	`{"placement_id": "plcm"}`,
	`{"id", "placementId": "plc"}`,
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
}
