package tadvertising_blis

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSpec struct {
	name    string
	json    string
	wantErr bool
}

func testParams(t *testing.T, specs []testSpec) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema")

	for _, spec := range specs {
		t.Run(spec.name, func(t *testing.T) {
			err := validator.Validate(openrtb_ext.BidderTAdvertisingBlis, json.RawMessage(spec.json))
			if spec.wantErr {
				assert.Error(t, err, "Expected error but got none for: %s", spec.json)
			} else {
				assert.NoError(t, err, "Unexpected error for: %s", spec.json)
			}
		})
	}
}

func TestValidParams(t *testing.T) {
	testParams(t, []testSpec{
		{
			name: "Valid params with publisherId",
			json: `{"publisherId": "1427ab10f2e448057ed3b422"}`,
		},
	})
}

func TestInvalidParams(t *testing.T) {
	testParams(t, []testSpec{
		{
			name:    "Empty params",
			json:    `{}`,
			wantErr: true,
		},
		{
			name:    "Empty publisherId",
			json:    `{"publisherId": ""}`,
			wantErr: true,
		},
		{
			name:    "Invalid publisherId type",
			json:    `{"publisherId": 123}`,
			wantErr: true,
		},
		{
			name:    "Too long publisherId",
			json:    `{"publisherId": "111111111111111111111111111111111"}`,
			wantErr: true,
		},
	})
}
