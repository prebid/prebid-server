package goadserver

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, validParam := range validParams {
		err := validator.Validate(openrtb_ext.BidderGoadserver, json.RawMessage(validParam))
		assert.NoErrorf(t, err, "Schema rejected goadserver params: %s", validParam)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, invalidParam := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderGoadserver, json.RawMessage(invalidParam))
		assert.Errorf(t, err, "Schema allowed unexpected params: %s", invalidParam)
	}
}

var validParams = []string{
	`{"token": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"}`,
	`{"token": "tok_123-abc"}`,
	`{"token": "tok", "floor": 0.5}`,
	`{"token": "tok", "subid": "sports"}`,
	`{"token": "tok", "subid": 42}`,
}

var invalidParams = []string{
	`{}`,
	`{"token": ""}`,
	`{"token": "tok/../x"}`,
	`{"token": "tok with spaces"}`,
	`{"token": 123}`,
	`{"token": "tok", "floor": -1}`,
	`{"token": "tok", "floor": "0.5"}`,
	`{"token": "tok", "subid": 1.5}`,
	`nil`,
	``,
	`[]`,
	`true`,
}
