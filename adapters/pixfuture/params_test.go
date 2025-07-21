package pixfuture

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema.")

	for _, p := range validParams {
		assert.NoErrorf(t, validator.Validate(openrtb_ext.BidderPixfuture, json.RawMessage(p)), "Schema rejected valid params: %s", p)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema")

	for _, p := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderPixfuture, json.RawMessage(p))
		assert.Errorf(t, err, "Schema allowed invalid params: %s", p)
	}
}

var validParams = []string{
	`{"pix_id": "123"}`,    // Minimum length satisfied
	`{"pix_id": "abcdef"}`, // Longer valid string
}

var invalidParams = []string{
	`{"pix_id": 123}`,  // Wrong type (integer)
	`{"pix_id": "ab"}`, // Too short (minLength: 3)
	`{}`,               // Missing required pix_id
	`{"pix_id": ""}`,   // Empty string (violates minLength)
}
