package amx

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

var validBidParams = []string{
	`{"tagId":"sampleTagId", "adUnitId": "sampleAdUnitId"}`,
	`{"tagId":"sampleTagId", "adUnitId": ""}`,
	`{"adUnitId": ""}`,
	`{"adUnitId": "sampleAdUnitId"}`,
	`{"tagId":"sampleTagId"}`,
	`{"tagId":""}`,
	`{}`,
	`{"otherValue": "ignored"}`,
	`{"tagId": "sampleTagId", "otherValue": "ignored"}`,
	`{"otherValue": "ignored", "adUnitId": "sampleAdUnitId"}`,
}

var invalidBidParams = []string{
	`{"tagId":1234}`,
	`{"tagId": true}`,
	`{"adUnitId": true}`,
	`{"adUnitId": null}`,
	`{"adUnitId": null, "tagId": "sampleTagId"}`,
	`{"adUnitId": 1234, "tagId": "sampleTagId"}`,
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
