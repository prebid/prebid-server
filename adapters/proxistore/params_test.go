package proxistore

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v4/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema. %v", err)

	for _, p := range validParams {
		err := validator.Validate(openrtb_ext.BidderProxistore, json.RawMessage(p))
		assert.NoError(t, err, "Schema rejected valid params: %s", p)
	}
}

func TestInvalidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	require.NoError(t, err, "Failed to fetch the json schema. %v", err)

	for _, p := range invalidParams {
		err := validator.Validate(openrtb_ext.BidderProxistore, json.RawMessage(p))
		assert.Error(t, err, "Schema allowed invalid params: %s", p)
	}
}

var validParams = []string{
	`{"website": "example.com", "language": "fr"}`,
	`{"website": "test-site", "language": "en"}`,
	`{"website": "publisher.be", "language": "nl"}`,
}

var invalidParams = []string{
	`{}`,
	`{"website": "example.com"}`,
	`{"language": "fr"}`,
	`{"website": "", "language": "fr"}`,
	`{"website": "example.com", "language": ""}`,
	`{"website": 123, "language": "fr"}`,
	`{"website": "example.com", "language": 456}`,
	`null`,
}
