package openrtb_ext_test

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestInvalidDeviceExt(t *testing.T) {
	var s openrtb_ext.ExtDevice
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":105}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":true,"minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":null,"minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":"75","minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")

	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":85}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":85,"minheightperc":-5}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":85,"minheightperc":false}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
	assert.EqualError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":85,"minheightperc":"75"}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
}

func TestValidDeviceExt(t *testing.T) {
	var s openrtb_ext.ExtDevice
	assert.NoError(t, json.Unmarshal([]byte(`{"prebid":{}}`), &s))
	assert.Nil(t, s.Prebid.Interstitial)
	assert.NoError(t, json.Unmarshal([]byte(`{}`), &s))
	assert.Nil(t, s.Prebid.Interstitial)
	assert.NoError(t, json.Unmarshal([]byte(`{"prebid":{"interstitial":{"minwidthperc":75,"minheightperc":60}}}`), &s))
	assert.EqualValues(t, 75, s.Prebid.Interstitial.MinWidthPerc)
	assert.EqualValues(t, 60, s.Prebid.Interstitial.MinHeightPerc)
}
