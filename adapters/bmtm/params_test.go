package bmtm

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	assert.NoError(t, err, fmt.Sprintf("Failed to fetch the json-schemas: %s", err.Error()))

	for _, validParam := range validParams {
		err := validator.Validate(openrtb_ext.BidderBmtm, json.RawMessage(validParam))
		assert.NoError(t, err, fmt.Sprintf("Schema rejected brightMountainMedia params: %s", validParam))
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	assert.NoError(t, err, fmt.Sprintf("Failed to fetch the json-schemas: %s", err.Error()))

	for _, invalidParam := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderBmtm, json.RawMessage(invalidParam))
		assert.Error(t, err, fmt.Sprintf("Schema allowed unexpected params: %s", invalidParam))
	}
}

var validParams = []string{
	`{"placement_id": 329}`,
	`{"placement_id": 12450}`,
}

var invalidParams = []string{
	`{"placement_id": "548d4e75w7a5d8e1w7w5r7ee7"}`,
	`{"placement_id": "42"}`,
}
