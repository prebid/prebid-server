package exgeniee

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// This file intends to test static/bidder-params/ex_geniee.json
//
// These also validate the format of the external API: request.imp[i].ext.prebid.bidder.ex_geniee

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, validParam := range validParams {
		err := validator.Validate(testBidderName, json.RawMessage(validParam))
		assert.NoError(t, err, "Schema rejected ex_geniee params: %s", validParam)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, invalidParam := range invalidParams {
		err := validator.Validate(testBidderName, json.RawMessage(invalidParam))
		assert.Error(t, err, "Schema allowed unexpected params: %s", invalidParam)
	}
}

var validParams = []string{
	`{"partnerId": 1, "placementId": "slot-1"}`,
	// The schema only refuses an empty placementId: Geniee Exchange validates the
	// value itself, so a value outside its documented format still passes here.
	`{"partnerId": 12345, "placementId": "top banner&id=9"}`,
	`{"partnerId": 4321}`,
}

var invalidParams = []string{
	`null`,
	`true`,
	`[]`,
	`{}`,
	`{"placementId": "slot-1"}`,
	`{"partnerId": 0, "placementId": "slot-1"}`,
	`{"partnerId": -1, "placementId": "slot-1"}`,
	`{"partnerId": 1.5, "placementId": "slot-1"}`,
	`{"partnerId": "4321", "placementId": "slot-1"}`,
	`{"partnerId": true, "placementId": "slot-1"}`,
	`{"partnerId": 4321, "placementId": 1}`,
	`{"partnerId": 4321, "placementId": ""}`,
}
