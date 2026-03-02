package matterfull

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
		err := validator.Validate(openrtb_ext.BidderMatterfull, json.RawMessage(validParam))
		assert.NoErrorf(t, err, "Schema rejected Matterfull params: %s", validParam)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json-schemas")

	for _, invalidParam := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderMatterfull, json.RawMessage(invalidParam))
		assert.Errorf(t, err, "Schema allowed unexpected params: %s", invalidParam)
	}
}

var validParams = []string{
	`{"pid": "LUN2gcJFHRwysZVTm8p3"}`,
}

var invalidParams = []string{
	`{"publisher": "34563434"}`,
	`nil`,
	``,
	`[]`,
	`true`,
}
