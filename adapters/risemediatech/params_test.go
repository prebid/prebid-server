package risemediatech

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the JSON schema")

	for _, p := range validParams {
		p := p // capture range variable
		t.Run(p, func(t *testing.T) {
			err := validator.Validate(openrtb_ext.BidderRiseMediaTech, json.RawMessage(p))
			if err != nil {
				t.Errorf("Schema rejected valid params: %s — error: %v", p, err)
			}
		})
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the JSON schema")

	for _, p := range invalidParams {
		p := p // capture range variable
		t.Run(p, func(t *testing.T) {
			err := validator.Validate(openrtb_ext.BidderRiseMediaTech, json.RawMessage(p))
			if err == nil {
				t.Errorf("Schema allowed invalid params: %s", p)
			}
		})
	}
}

var validParams = []string{
	`{"bidfloor": 0.01}`,
	`{"bidfloor": 2.5, "testMode": 1}`,
}

var invalidParams = []string{
	`{"bidfloor": "1.2"}`,
	`{"testMode": "yes"}`,
	`{"bidfloor": -5}`,
	`{"testMode": 9999}`,
}
