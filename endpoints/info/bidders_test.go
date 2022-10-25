package info

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/config"
	"github.com/stretchr/testify/assert"
)

func TestPrepareBiddersResponseAll(t *testing.T) {
	var (
		enabled  = config.BidderInfo{Disabled: false}
		disabled = config.BidderInfo{Disabled: true}
	)

	testCases := []struct {
		description  string
		givenBidders config.BidderInfos
		givenAliases map[string]string
		expected     string
	}{
		{
			description:  "None",
			givenBidders: config.BidderInfos{},
			givenAliases: nil,
			expected:     `[]`,
		},
		{
			description:  "Core Bidders Only - One - Enabled",
			givenBidders: config.BidderInfos{"a": enabled},
			givenAliases: nil,
			expected:     `["a"]`,
		},
		{
			description:  "Core Bidders Only - One - Disabled",
			givenBidders: config.BidderInfos{"a": disabled},
			givenAliases: nil,
			expected:     `["a"]`,
		},
		{
			description:  "Core Bidders Only - Many",
			givenBidders: config.BidderInfos{"a": enabled, "b": enabled},
			givenAliases: nil,
			expected:     `["a","b"]`,
		},
		{
			description:  "Core Bidders Only - Many - Mixed",
			givenBidders: config.BidderInfos{"a": disabled, "b": enabled},
			givenAliases: nil,
			expected:     `["a","b"]`,
		},
		{
			description:  "Core Bidders Only - Many - Sorted",
			givenBidders: config.BidderInfos{"b": enabled, "a": enabled},
			givenAliases: nil,
			expected:     `["a","b"]`,
		},
		{
			description:  "With Aliases - One",
			givenBidders: config.BidderInfos{"a": enabled},
			givenAliases: map[string]string{"b": "a"},
			expected:     `["a","b"]`,
		},
		{
			description:  "With Aliases - Many",
			givenBidders: config.BidderInfos{"a": enabled, "b": disabled},
			givenAliases: map[string]string{"x": "a", "y": "b"},
			expected:     `["a","b","x","y"]`,
		},
		{
			description:  "With Aliases - Sorted",
			givenBidders: config.BidderInfos{"z": enabled},
			givenAliases: map[string]string{"a": "z"},
			expected:     `["a","z"]`,
		},
	}

	for _, test := range testCases {
		result, err := prepareBiddersResponseAll(test.givenBidders, test.givenAliases)

		assert.NoError(t, err, test.description)
		assert.Equal(t, []byte(test.expected), result, test.description)
	}
}

func TestPrepareBiddersResponseEnabledOnly(t *testing.T) {
	var (
		enabled  = config.BidderInfo{Disabled: false}
		disabled = config.BidderInfo{Disabled: true}
	)

	testCases := []struct {
		description  string
		givenBidders config.BidderInfos
		givenAliases map[string]string
		expected     string
	}{
		{
			description:  "None",
			givenBidders: config.BidderInfos{},
			givenAliases: nil,
			expected:     `[]`,
		},
		{
			description:  "Core Bidders Only - One - Enabled",
			givenBidders: config.BidderInfos{"a": enabled},
			givenAliases: nil,
			expected:     `["a"]`,
		},
		{
			description:  "Core Bidders Only - One - Disabled",
			givenBidders: config.BidderInfos{"a": disabled},
			givenAliases: nil,
			expected:     `[]`,
		},
		{
			description:  "Core Bidders Only - Many",
			givenBidders: config.BidderInfos{"a": enabled, "b": enabled},
			givenAliases: nil,
			expected:     `["a","b"]`,
		},
		{
			description:  "Core Bidders Only - Many - Mixed",
			givenBidders: config.BidderInfos{"a": disabled, "b": enabled},
			givenAliases: nil,
			expected:     `["b"]`,
		},
		{
			description:  "Core Bidders Only - Many - Sorted",
			givenBidders: config.BidderInfos{"b": enabled, "a": enabled},
			givenAliases: nil,
			expected:     `["a","b"]`,
		},
		{
			description:  "With Aliases - One",
			givenBidders: config.BidderInfos{"a": enabled},
			givenAliases: map[string]string{"b": "a"},
			expected:     `["a","b"]`,
		},
		{
			description:  "With Aliases - Many",
			givenBidders: config.BidderInfos{"a": enabled, "b": disabled},
			givenAliases: map[string]string{"x": "a", "y": "b"},
			expected:     `["a","x"]`,
		},
		{
			description:  "With Aliases - Sorted",
			givenBidders: config.BidderInfos{"z": enabled},
			givenAliases: map[string]string{"a": "z"},
			expected:     `["a","z"]`,
		},
	}

	for _, test := range testCases {
		result, err := prepareBiddersResponseEnabledOnly(test.givenBidders, test.givenAliases)

		assert.NoError(t, err, test.description)
		assert.Equal(t, []byte(test.expected), result, test.description)
	}
}

func TestBiddersHandler(t *testing.T) {
	var (
		enabled  = config.BidderInfo{Disabled: false}
		disabled = config.BidderInfo{Disabled: true}
	)

	bidders := config.BidderInfos{"a": enabled, "b": disabled}
	aliases := map[string]string{"x": "a", "y": "b"}

	testCases := []struct {
		description     string
		givenURL        string
		expectedStatus  int
		expectedBody    string
		expectedHeaders http.Header
	}{
		{
			description:     "No Query Parameters - Backwards Compatibility",
			givenURL:        "/info/bidders",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			description:     "Enabled Only - False",
			givenURL:        "/info/bidders?enabledonly=false",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			description:     "Enabled Only - False - Case Insensitive",
			givenURL:        "/info/bidders?enabledonly=fAlSe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			description:     "Enabled Only - True",
			givenURL:        "/info/bidders?enabledonly=true",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","x"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			description:     "Enabled Only - True - Case Insensitive",
			givenURL:        "/info/bidders?enabledonly=TrUe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","x"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			description:     "Enabled Only - Invalid",
			givenURL:        "/info/bidders?enabledonly=foo",
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `Invalid value for 'enabledonly' query param, must be of boolean type`,
			expectedHeaders: http.Header{},
		},
		{
			description:     "Enabled Only - Missing Value",
			givenURL:        "/info/bidders?enabledonly=",
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `Invalid value for 'enabledonly' query param, must be of boolean type`,
			expectedHeaders: http.Header{},
		},
	}

	for _, test := range testCases {
		handler := NewBiddersEndpoint(bidders, aliases)

		request := httptest.NewRequest("GET", test.givenURL, nil)

		responseRecorder := httptest.NewRecorder()
		handler(responseRecorder, request, nil)

		result := responseRecorder.Result()
		assert.Equal(t, result.StatusCode, test.expectedStatus)

		resultBody, _ := ioutil.ReadAll(result.Body)
		assert.Equal(t, []byte(test.expectedBody), resultBody)

		resultHeaders := result.Header
		assert.Equal(t, test.expectedHeaders, resultHeaders)
	}
}
