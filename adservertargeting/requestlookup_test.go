package adservertargeting

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
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
			description:       "valid request with correct ext.prebid, and empty ad server targeting",
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

func TestGetValueFromBidRequest(t *testing.T) {

	testCases := []struct {
		description       string
		inputPath         string
		expectedTargeting RequestTargetingData
		expectedError     bool
	}{
		{
			description:       "get existing value from query params",
			inputPath:         "ext.prebid.amp.data.amp-key",
			expectedTargeting: RequestTargetingData{SingleVal: json.RawMessage(`testAmpKey`)},
			expectedError:     false,
		},
		{
			description:       "get non-existing value from query params",
			inputPath:         "ext.prebid.amp.data.amp-key-not-existing",
			expectedTargeting: RequestTargetingData{},
			expectedError:     true,
		},
		{
			description: "get existing value from impressions",
			inputPath:   "imp.banner.h",
			expectedTargeting: RequestTargetingData{
				TargetingValueByImpId: map[string][]byte{
					"test_imp1": []byte(`350`),
					"test_imp2": []byte(`360`),
				},
			},
			expectedError: false,
		},
		{
			description: "get existing value from impressions ext",
			inputPath:   "imp.ext.appnexus.placementId",
			expectedTargeting: RequestTargetingData{
				TargetingValueByImpId: map[string][]byte{
					"test_imp1": []byte(`123`),
					"test_imp2": []byte(`456`),
				},
			},
			expectedError: false,
		},
		{
			description:       "get non-existing value from impressions",
			inputPath:         "imp.video",
			expectedTargeting: RequestTargetingData{},
			expectedError:     true,
		},
		{
			description: "get existing value from req",
			inputPath:   "site.page",
			expectedTargeting: RequestTargetingData{
				SingleVal: json.RawMessage(`test.com`),
			},
			expectedError: false,
		},
		{
			description:       "get non-existing value from req",
			inputPath:         "app",
			expectedTargeting: RequestTargetingData{},
			expectedError:     true,
		},
	}

	reqCache := requestCache{
		resolvedReq: json.RawMessage(reqFullValid),
	}

	u, _ := url.Parse(testUrl)
	params := u.Query()

	for _, test := range testCases {
		resTargetingData, err := getValueFromBidRequest(&reqCache, test.inputPath, params)

		assert.Equal(t, test.expectedTargeting, resTargetingData, "incorrect targeting data returned")

		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}

func TestGetValueFromQueryParam(t *testing.T) {
	u, _ := url.Parse(testUrl)
	params := u.Query()

	testCases := []struct {
		description   string
		inputPath     string
		expectedValue json.RawMessage
		expectedError bool
	}{
		{
			description:   "get existing value from query params",
			inputPath:     "ext.prebid.amp.data.amp-key",
			expectedValue: json.RawMessage(`testAmpKey`),
			expectedError: false,
		},
		{
			description:   "get non-existing value from query params",
			inputPath:     "ext.prebid.amp.data.amp-key-not-existing",
			expectedValue: nil,
			expectedError: true,
		},
		{
			description:   "get value from not query params path",
			inputPath:     "ext.data.amp-key-not-existing",
			expectedValue: nil,
			expectedError: false,
		},
	}

	for _, test := range testCases {
		res, err := getValueFromQueryParam(test.inputPath, params)

		assert.Equal(t, test.expectedValue, res, "incorrect value found")
		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}

func TestGetValueFromImp(t *testing.T) {

	testCases := []struct {
		description       string
		inputPath         string
		inputRequest      json.RawMessage
		expectedTargeting map[string][]byte
		expectedError     bool
	}{

		{
			description:  "get existing value from impressions",
			inputPath:    "imp.banner.h",
			inputRequest: json.RawMessage(reqFullValid),
			expectedTargeting: map[string][]byte{
				"test_imp1": []byte(`350`),
				"test_imp2": []byte(`360`),
			},
			expectedError: false,
		},
		{
			description: "get existing value from impressions ext",
			inputPath:   "imp.ext.appnexus.placementId",
			expectedTargeting: map[string][]byte{
				"test_imp1": []byte(`123`),
				"test_imp2": []byte(`456`),
			},
			inputRequest:  json.RawMessage(reqFullValid),
			expectedError: false,
		},
		{
			description:       "get non-existing value from impressions",
			inputPath:         "imp.video",
			expectedTargeting: map[string][]byte(nil),
			inputRequest:      json.RawMessage(reqFullValid),
			expectedError:     true,
		},
		{
			description:       "get value from invalid impressions",
			inputPath:         "imp.video",
			expectedTargeting: map[string][]byte(nil),
			inputRequest:      json.RawMessage(reqFullInvalid),
			expectedError:     true,
		},
		{
			description:       "get value from request without impressions",
			inputPath:         "imp.video",
			expectedTargeting: map[string][]byte(nil),
			inputRequest:      json.RawMessage(`{}`),
			expectedError:     true,
		},
	}

	for _, test := range testCases {

		reqCache := requestCache{
			resolvedReq: test.inputRequest,
		}

		resData, err := getValueFromImp(test.inputPath, &reqCache)

		assert.Equal(t, test.expectedTargeting, resData, "incorrect imp data returned")

		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}

func TestGetValueFromRequestJson(t *testing.T) {

	testCases := []struct {
		description   string
		inputPath     string
		expectedValue json.RawMessage
		expectedError bool
	}{

		{
			description:   "get existing value from req",
			inputPath:     "site.page",
			expectedValue: json.RawMessage(`test.com`),
			expectedError: false,
		},
		{
			description:   "get non-existing value from req",
			inputPath:     "app",
			expectedValue: json.RawMessage(nil),
			expectedError: true,
		},
	}

	reqCache := requestCache{
		resolvedReq: json.RawMessage(reqFullValid),
	}

	for _, test := range testCases {
		resValue, err := getDataFromRequestJson(test.inputPath, &reqCache)

		assert.Equal(t, test.expectedValue, resValue, "incorrect request data returned")

		if test.expectedError {
			assert.Error(t, err, "expected error not returned")
		} else {
			assert.NoError(t, err, "unexpected error returned")
		}
	}
}
