package adservertargeting

import (
	"encoding/json"
	"github.com/prebid/openrtb/v17/openrtb2"
	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAdServerTargeting(t *testing.T) {

	testCases := []struct {
		description       string
		inputRequestExt   json.RawMessage
		expectedTargeting []openrtb_ext.AdServerTarget
		expectedError     bool
	}{
		{
			description:       "valid request with no ext.prebid",
			inputRequestExt:   json.RawMessage(``),
			expectedTargeting: nil,
			expectedError:     false,
		},
		{
			description:       "valid request with correct ext.prebid, no ad server targeting",
			inputRequestExt:   json.RawMessage(`{"prebid":{}}`),
			expectedTargeting: nil,
			expectedError:     false,
		},
		{
			description:       "valid request with correct ext.prebid, no ad server targeting",
			inputRequestExt:   json.RawMessage(`{"prebid":{"adservertargeting":[]}}`),
			expectedTargeting: []openrtb_ext.AdServerTarget{},
			expectedError:     false,
		},
		{
			description: "valid request with correct ext.prebid, and with ad server targeting",
			inputRequestExt: json.RawMessage(`{"prebid":{"adservertargeting":[
					{"key": "adt_key",
                    "source": "bidrequest",
                    "value": "ext.prebid.data"}
				]}}`),
			expectedTargeting: []openrtb_ext.AdServerTarget{
				{Key: "adt_key", Source: "bidrequest", Value: "ext.prebid.data"},
			},
			expectedError: false,
		},
	}

	for _, test := range testCases {
		request := &openrtb_ext.RequestWrapper{
			BidRequest: &openrtb2.BidRequest{ID: "req_id", Ext: test.inputRequestExt},
		}

		actualTargeting, err := getAdServerTargeting(request)

		assert.Equal(t, test.expectedTargeting, actualTargeting, "targeting data doesn't match")
		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}

func TestValidateAdServerTargeting(t *testing.T) {
	testCases := []struct {
		description       string
		inputTargeting    []openrtb_ext.AdServerTarget
		expectedTargeting []openrtb_ext.AdServerTarget
		expectedWarnings  []openrtb_ext.ExtBidderMessage
	}{
		{
			description: "valid targeting object",
			inputTargeting: []openrtb_ext.AdServerTarget{
				{Key: "adt_key", Source: "bidrequest", Value: "ext.prebid.data"},
			},
			expectedTargeting: []openrtb_ext.AdServerTarget{
				{Key: "adt_key", Source: "bidrequest", Value: "ext.prebid.data"},
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage(nil),
		},
		{
			description: "invalid targeting object: key",
			inputTargeting: []openrtb_ext.AdServerTarget{
				{Key: "", Source: "bidrequest", Value: "ext.prebid.data"},
			},
			expectedTargeting: []openrtb_ext.AdServerTarget(nil),
			expectedWarnings: []openrtb_ext.ExtBidderMessage{
				{Code: 10007, Message: "Key is empty for the ad server targeting object at index 0"},
			},
		},
		{
			description: "invalid targeting object: source",
			inputTargeting: []openrtb_ext.AdServerTarget{
				{Key: "adt_key", Source: "incorrect", Value: "ext.prebid.data"},
			},
			expectedTargeting: []openrtb_ext.AdServerTarget(nil),
			expectedWarnings: []openrtb_ext.ExtBidderMessage{
				{Code: 10007, Message: "Incorrect source for the ad server targeting object at index 0"},
			},
		},
		{
			description: "invalid targeting object: value",
			inputTargeting: []openrtb_ext.AdServerTarget{
				{Key: "adt_key", Source: "static", Value: ""},
			},
			expectedTargeting: []openrtb_ext.AdServerTarget(nil),
			expectedWarnings: []openrtb_ext.ExtBidderMessage{
				{Code: 10007, Message: "Value is empty for the ad server targeting object at index 0"},
			},
		},
		{
			description: "valid and invalid targeting object",
			inputTargeting: []openrtb_ext.AdServerTarget{
				{Key: "adt_key1", Source: "static", Value: "valid"},
				{Key: "adt_key2", Source: "static", Value: ""},
			},
			expectedTargeting: []openrtb_ext.AdServerTarget{
				{Key: "adt_key1", Source: "static", Value: "valid"},
			},
			expectedWarnings: []openrtb_ext.ExtBidderMessage{
				{Code: 10007, Message: "Value is empty for the ad server targeting object at index 1"},
			},
		},
	}

	for _, test := range testCases {
		actualTargeting, actualWarnings := validateAdServerTargeting(test.inputTargeting)
		assert.Equal(t, test.expectedTargeting, actualTargeting, "incorrect targeting data")
		assert.Equal(t, test.expectedWarnings, actualWarnings, "incorrect warnings")
	}
}
