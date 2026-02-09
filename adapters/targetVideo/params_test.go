package targetVideo

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, validParam := range validParams {
		assert.NoError(t, validator.Validate(openrtb_ext.BidderTargetVideo, json.RawMessage(validParam)),
			"Schema rejected targetVideo params: %s", validParam)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, invalidParam := range invalidParams {
		assert.Error(t, validator.Validate(openrtb_ext.BidderTargetVideo, json.RawMessage(invalidParam)),
			"Schema should have rejected unexpected params: %s", invalidParam)
	}
}

var validParams = []string{
	`{"placementId":846}`,
	`{"placementId":"846"}`,
}

var invalidParams = []string{
	`null`,
	`nil`,
	`undefined`,
	`{"placementId": "%9"}`,
	`{"publisherId": "as9""}`,
	`{"placementId": true}`,
	`{"placementId": ""}`,
}
