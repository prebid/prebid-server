package amx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var validBidParams = []string{
	`{"tagId":"sampleTagId"}`,
	`{"tagId":""}`,
	`{}`,
	`{"otherValue": "ignored"}`,
}

var invalidBidParams = []string{
	`{"tagId":1234}`,
	`{"tagId": true}`,
}

func TestValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	assert.Nil(t, err)
	for _, params := range validBidParams {
		assert.Nil(t, validator.Validate(openrtb_ext.BidderAMX, json.RawMessage(params)))
	}
}

func TestInValidParams(t *testing.T) {
	validator, err := openrtb_ext.NewBidderParamsValidator("../../static/bidder-params")
	assert.Nil(t, err)
	for _, params := range invalidBidParams {
		assert.NotNil(t, validator.Validate(openrtb_ext.BidderAMX, json.RawMessage(params)))
	}
}
