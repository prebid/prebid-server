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
	`{"host": "ads.example.com", "token": "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4"}`,
	`{"host": "ADS.Example.COM", "token": "tok_123-abc"}`,
	`{"host": "ads.example.com", "token": "tok", "floor": 0.5}`,
	`{"host": "ads.example.com", "token": "tok", "subid": "sports"}`,
	`{"host": "ads.example.com", "token": "tok", "subid": 42}`,
}

var invalidParams = []string{
	`{"host": "ads.example.com"}`,
	`{"token": "tok"}`,
	`{"host": "", "token": "tok"}`,
	`{"host": "ads.example.com", "token": ""}`,
	`{"host": "localhost", "token": "tok"}`,
	`{"host": "203.0.113.42", "token": "tok"}`,
	`{"host": "ads.example.com:8080", "token": "tok"}`,
	`{"host": "https://ads.example.com", "token": "tok"}`,
	`{"host": "ads.example.com/path", "token": "tok"}`,
	`{"host": "user@ads.example.com", "token": "tok"}`,
	`{"host": "ads.example.com", "token": "tok/../x"}`,
	`{"host": "ads.example.com", "token": 123}`,
	`{"host": "ads.example.com", "token": "tok", "floor": -1}`,
	`{"host": "ads.example.com", "token": "tok", "floor": "0.5"}`,
	`{"host": "ads.example.com", "token": "tok", "subid": 1.5}`,
	`nil`,
	``,
	`[]`,
	`true`,
}
