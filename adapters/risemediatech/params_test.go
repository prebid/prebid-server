package risemediatech

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the JSON schema")

	tests := []struct {
		name  string
		input string
	}{
		{"Valid bidfloor only", `{"bidfloor": 0.01}`},
		{"Valid bidfloor with testMode", `{"bidfloor": 2.5, "testMode": 1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.NoError(t, validator.Validate(openrtb_ext.BidderRiseMediaTech, json.RawMessage(tt.input)))
		})
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the JSON schema")

	tests := []struct {
		name  string
		input string
	}{
		{"Invalid bidfloor type", `{"bidfloor": "1.2"}`},
		{"Invalid testMode type", `{"testMode": "yes"}`},
		{"Negative bidfloor", `{"bidfloor": -5}`},
		{"Invalid testMode value", `{"testMode": 9999}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Error(t, validator.Validate(openrtb_ext.BidderRiseMediaTech, json.RawMessage(tt.input)))
		})
	}
}
