package openrtb_ext

import (
	"encoding/json"
	"testing"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/stretchr/testify/assert"
)

func TestInvalidDeviceExt(t *testing.T) {
	var s ExtDevice
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":105}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":true,"minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":null,"minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":"75","minheightperc":0}}}`), &s), "request.device.ext.prebid.interstitial.minwidthperc must be a number between 0 and 100")

	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":85}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":85,"minheightperc":-5}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":85,"minheightperc":false}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
	assert.EqualError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":85,"minheightperc":"75"}}}`), &s), "request.device.ext.prebid.interstitial.minheightperc must be a number between 0 and 100")
}

func TestValidDeviceExt(t *testing.T) {
	var s ExtDevice
	assert.NoError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{}}`), &s))
	assert.Nil(t, s.Prebid.Interstitial)
	assert.NoError(t, jsonutil.UnmarshalValid([]byte(`{}`), &s))
	assert.Nil(t, s.Prebid.Interstitial)
	assert.NoError(t, jsonutil.UnmarshalValid([]byte(`{"prebid":{"interstitial":{"minwidthperc":75,"minheightperc":60}}}`), &s))
	assert.EqualValues(t, 75, s.Prebid.Interstitial.MinWidthPerc)
	assert.EqualValues(t, 60, s.Prebid.Interstitial.MinHeightPerc)
}

func TestIsKnownIOSAppTrackingStatus(t *testing.T) {
	valid := []int64{0, 1, 2, 3}
	invalid := []int64{-1, 4}

	for _, v := range valid {
		assert.True(t, IsKnownIOSAppTrackingStatus(v))
	}

	for _, v := range invalid {
		assert.False(t, IsKnownIOSAppTrackingStatus(v))
	}
}

func TestParseDeviceExtATTS(t *testing.T) {
	authorized := IOSAppTrackingStatusAuthorized

	tests := []struct {
		description    string
		givenExt       json.RawMessage
		expectedStatus *IOSAppTrackingStatus
		expectedError  string
	}{
		{
			description:    "Nil",
			givenExt:       nil,
			expectedStatus: nil,
		},
		{
			description:    "Empty",
			givenExt:       json.RawMessage(``),
			expectedStatus: nil,
		},
		{
			description:    "Empty Object",
			givenExt:       json.RawMessage(`{}`),
			expectedStatus: nil,
		},
		{
			description:    "Valid",
			givenExt:       json.RawMessage(`{"atts":3}`),
			expectedStatus: &authorized,
		},
		{
			description:    "Invalid Value",
			givenExt:       json.RawMessage(`{"atts":5}`),
			expectedStatus: nil,
			expectedError:  "invalid status",
		},
		{
			// This test case produces an error with the standard Go library, but jsonparser doesn't
			// return an error for malformed JSON. It treats this case the same as not being found.
			description:    "Malformed - Standard Test Case",
			givenExt:       json.RawMessage(`malformed`),
			expectedStatus: nil,
		},
		{
			description:    "Malformed - Wrong Type",
			givenExt:       json.RawMessage(`{"atts":"1"}`),
			expectedStatus: nil,
			expectedError:  "Value is not a number: 1",
		},
	}

	for _, test := range tests {
		status, err := ParseDeviceExtATTS(test.givenExt)

		if test.expectedError == "" {
			assert.NoError(t, err, test.description+":err")
		} else {
			assert.EqualError(t, err, test.expectedError, test.description+":err")
		}

		assert.Equal(t, test.expectedStatus, status, test.description+":status")
	}
}
