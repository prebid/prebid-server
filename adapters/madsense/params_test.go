package madsense

import (
	"encoding/json"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
	"testing"
)

// This file actually intends to test static/bidder-params/madsense.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.madsense

// TestValidParams makes sure that the madsense schema accepts all imp.ext fields which we intend to support.
func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, validParam := range validParams {
		err := validator.Validate(openrtb_ext.BidderMadSense, json.RawMessage(validParam))
		require.NoError(t, err, "Schema rejected madsense params: %s", validParam)
	}
}

// TestInvalidParams makes sure that the madsense schema rejects all the imp.ext fields we don't support.
func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, invalidParam := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderMadSense, json.RawMessage(invalidParam))
		require.Error(t, err, "Schema allowed unexpected params: %s", invalidParam)
	}
}

var validParams = []string{
	`{"company_id": "9876543"}`,
}

var invalidParams = []string{
	``,
	`null`,
	`true`,
	`5`,
	`4.2`,
	`[]`,
	`{}`,
	`{"companyId": "987654a"}`,
	`{"companyId": "98765432"}`,
	`{"company_id": ""}`,
}
