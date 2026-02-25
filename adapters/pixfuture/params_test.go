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

	tests := []struct {
		name string
		json string
	}{
		{
			name: "Minimum length satisfied",
			json: `{"pix_id": "123"}`,
		},
		{
			name: "Longer valid string",
			json: `{"pix_id": "abcdef"}`,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			assert.NoErrorf(t, validator.Validate(openrtb_ext.BidderPixfuture, json.RawMessage(tt.json)), "Schema rejected valid params: %s", tt.json)
		})
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema.")

	tests := []struct {
		name string
		json string
	}{
		{
			name: "Wrong type (integer)",
			json: `{"pix_id": 123}`,
		},
		{
			name: "Too short (minLength: 3)",
			json: `{"pix_id": "ab"}`,
		},
		{
			name: "Missing required pix_id",
			json: `{}`,
		},
		{
			name: "Empty string (violates minLength)",
			json: `{"pix_id": ""}`,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			err := validator.Validate(openrtb_ext.BidderPixfuture, json.RawMessage(tt.json))
			assert.Errorf(t, err, "Schema allowed invalid params: %s", tt.json)
		})
	}
}
