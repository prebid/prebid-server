package info

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/v2/config"
	"github.com/stretchr/testify/assert"
)

func TestPrepareBiddersResponseAll(t *testing.T) {
	var (
		enabledCore   = config.BidderInfo{Disabled: false}
		enabledAlias  = config.BidderInfo{Disabled: false, AliasOf: "something"}
		disabledCore  = config.BidderInfo{Disabled: true}
		disabledAlias = config.BidderInfo{Disabled: true, AliasOf: "something"}
	)

	testCases := []struct {
		name                string
		givenBidders        config.BidderInfos
		givenRequestAliases map[string]string
		expected            string
	}{
		{
			name:                "none",
			givenBidders:        config.BidderInfos{},
			givenRequestAliases: nil,
			expected:            `[]`,
		},
		{
			name:                "core-one-enabled",
			givenBidders:        config.BidderInfos{"a": enabledCore},
			givenRequestAliases: nil,
			expected:            `["a"]`,
		},
		{
			name:                "core-one-disabled",
			givenBidders:        config.BidderInfos{"a": disabledCore},
			givenRequestAliases: nil,
			expected:            `["a"]`,
		},
		{
			name:                "core-one-mixed",
			givenBidders:        config.BidderInfos{"a": disabledCore, "b": enabledCore},
			givenRequestAliases: nil,
			expected:            `["a","b"]`,
		},
		{
			name:                "core-one-mixed-sorted",
			givenBidders:        config.BidderInfos{"z": enabledCore, "a": enabledCore},
			givenRequestAliases: nil,
			expected:            `["a","z"]`,
		},
		{
			name:                "alias-one",
			givenBidders:        config.BidderInfos{"a": enabledAlias},
			givenRequestAliases: nil,
			expected:            `["a"]`,
		},
		{
			name:                "alias-mixed",
			givenBidders:        config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			givenRequestAliases: nil,
			expected:            `["a","b","c","d"]`,
		},
		{
			name:                "alias-mixed-sorted",
			givenBidders:        config.BidderInfos{"z": enabledAlias, "a": enabledCore},
			givenRequestAliases: nil,
			expected:            `["a","z"]`,
		},
		{
			name:                "defaultrequest-one",
			givenBidders:        config.BidderInfos{"a": enabledCore},
			givenRequestAliases: map[string]string{"b": "a"},
			expected:            `["a","b"]`,
		},
		{
			name:                "defaultrequest-mixed",
			givenBidders:        config.BidderInfos{"a": enabledCore, "b": disabledCore},
			givenRequestAliases: map[string]string{"x": "a", "y": "b"},
			expected:            `["a","b","x","y"]`,
		},
		{
			name:                "defaultrequest-mixed-sorted",
			givenBidders:        config.BidderInfos{"z": enabledCore},
			givenRequestAliases: map[string]string{"a": "z"},
			expected:            `["a","z"]`,
		},
		{
			name:                "mixed",
			givenBidders:        config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			givenRequestAliases: map[string]string{"z": "a"},
			expected:            `["a","b","c","d","z"]`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := prepareBiddersResponseAll(test.givenBidders, test.givenRequestAliases)
			assert.NoError(t, err)
			assert.Equal(t, []byte(test.expected), result)
		})
	}
}

func TestPrepareBiddersResponseAllBaseOnly(t *testing.T) {
	var (
		enabledCore   = config.BidderInfo{Disabled: false}
		enabledAlias  = config.BidderInfo{Disabled: false, AliasOf: "something"}
		disabledCore  = config.BidderInfo{Disabled: true}
		disabledAlias = config.BidderInfo{Disabled: true, AliasOf: "something"}
	)

	testCases := []struct {
		name         string
		givenBidders config.BidderInfos
		expected     string
	}{
		{
			name:         "none",
			givenBidders: config.BidderInfos{},
			expected:     `[]`,
		},
		{
			name:         "core-one-enabled",
			givenBidders: config.BidderInfos{"a": enabledCore},
			expected:     `["a"]`,
		},
		{
			name:         "core-one-disabled",
			givenBidders: config.BidderInfos{"a": disabledCore},
			expected:     `["a"]`,
		},
		{
			name:         "core-one-mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": enabledCore},
			expected:     `["a","b"]`,
		},
		{
			name:         "core-one-mixed-sorted",
			givenBidders: config.BidderInfos{"z": enabledCore, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "alias-one",
			givenBidders: config.BidderInfos{"a": enabledAlias},
			expected:     `[]`,
		},
		{
			name:         "alias-mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			expected:     `["a","c"]`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := prepareBiddersResponseAllBaseOnly(test.givenBidders)
			assert.NoError(t, err)
			assert.Equal(t, []byte(test.expected), result)
		})
	}
}

func TestPrepareBiddersResponseEnabledOnly(t *testing.T) {
	var (
		enabledCore   = config.BidderInfo{Disabled: false}
		enabledAlias  = config.BidderInfo{Disabled: false, AliasOf: "something"}
		disabledCore  = config.BidderInfo{Disabled: true}
		disabledAlias = config.BidderInfo{Disabled: true, AliasOf: "something"}
	)

	testCases := []struct {
		name                string
		givenBidders        config.BidderInfos
		givenRequestAliases map[string]string
		expected            string
	}{
		{
			name:                "none",
			givenBidders:        config.BidderInfos{},
			givenRequestAliases: nil,
			expected:            `[]`,
		},
		{
			name:                "core-one-enabled",
			givenBidders:        config.BidderInfos{"a": enabledCore},
			givenRequestAliases: nil,
			expected:            `["a"]`,
		},
		{
			name:                "core-one-disabled",
			givenBidders:        config.BidderInfos{"a": disabledCore},
			givenRequestAliases: nil,
			expected:            `[]`,
		},
		{
			name:                "core-one-mixed",
			givenBidders:        config.BidderInfos{"a": disabledCore, "b": enabledCore},
			givenRequestAliases: nil,
			expected:            `["b"]`,
		},
		{
			name:                "core-one-mixed-sorted",
			givenBidders:        config.BidderInfos{"z": enabledCore, "a": enabledCore},
			givenRequestAliases: nil,
			expected:            `["a","z"]`,
		},
		{
			name:                "alias-one",
			givenBidders:        config.BidderInfos{"a": enabledAlias},
			givenRequestAliases: nil,
			expected:            `["a"]`,
		},
		{
			name:                "alias-mixed",
			givenBidders:        config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			givenRequestAliases: nil,
			expected:            `["c","d"]`,
		},
		{
			name:                "alias-mixed-sorted",
			givenBidders:        config.BidderInfos{"z": enabledAlias, "a": enabledCore},
			givenRequestAliases: nil,
			expected:            `["a","z"]`,
		},
		{
			name:                "defaultrequest-one",
			givenBidders:        config.BidderInfos{"a": enabledCore},
			givenRequestAliases: map[string]string{"b": "a"},
			expected:            `["a","b"]`,
		},
		{
			name:                "defaultrequest-mixed",
			givenBidders:        config.BidderInfos{"a": enabledCore, "b": disabledCore},
			givenRequestAliases: map[string]string{"x": "a", "y": "b"},
			expected:            `["a","x"]`,
		},
		{
			name:                "defaultrequest-mixed-sorted",
			givenBidders:        config.BidderInfos{"z": enabledCore},
			givenRequestAliases: map[string]string{"a": "z"},
			expected:            `["a","z"]`,
		},
		{
			name:                "mixed",
			givenBidders:        config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			givenRequestAliases: map[string]string{"z": "a"},
			expected:            `["c","d"]`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := prepareBiddersResponseEnabledOnly(test.givenBidders, test.givenRequestAliases)
			assert.NoError(t, err)
			assert.Equal(t, []byte(test.expected), result)
		})
	}
}

func TestPrepareBiddersResponseEnabledOnlyBaseOnly(t *testing.T) {
	var (
		enabledCore   = config.BidderInfo{Disabled: false}
		enabledAlias  = config.BidderInfo{Disabled: false, AliasOf: "something"}
		disabledCore  = config.BidderInfo{Disabled: true}
		disabledAlias = config.BidderInfo{Disabled: true, AliasOf: "something"}
	)

	testCases := []struct {
		name         string
		givenBidders config.BidderInfos
		expected     string
	}{
		{
			name:         "none",
			givenBidders: config.BidderInfos{},
			expected:     `[]`,
		},
		{
			name:         "core-one-enabled",
			givenBidders: config.BidderInfos{"a": enabledCore},
			expected:     `["a"]`,
		},
		{
			name:         "core-one-disabled",
			givenBidders: config.BidderInfos{"a": disabledCore},
			expected:     `[]`,
		},
		{
			name:         "core-one-mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": enabledCore},
			expected:     `["b"]`,
		},
		{
			name:         "core-one-mixed-sorted",
			givenBidders: config.BidderInfos{"z": enabledCore, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "alias-one",
			givenBidders: config.BidderInfos{"a": enabledAlias},
			expected:     `[]`,
		},
		{
			name:         "alias-many",
			givenBidders: config.BidderInfos{"a": enabledAlias, "b": enabledAlias},
			expected:     `[]`,
		},
		{
			name:         "mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			expected:     `["c"]`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := prepareBiddersResponseEnabledOnlyBaseOnly(test.givenBidders)
			assert.NoError(t, err)
			assert.Equal(t, []byte(test.expected), result)
		})
	}
}

func TestBiddersHandler(t *testing.T) {
	var (
		enabledCore   = config.BidderInfo{Disabled: false}
		enabledAlias  = config.BidderInfo{Disabled: false, AliasOf: "something"}
		disabledCore  = config.BidderInfo{Disabled: true}
		disabledAlias = config.BidderInfo{Disabled: true, AliasOf: "something"}
	)

	bidders := config.BidderInfos{"a": enabledCore, "b": enabledAlias, "c": disabledCore, "d": disabledAlias}
	aliases := map[string]string{"x": "a", "y": "c"}

	testCases := []struct {
		name            string
		givenURL        string
		expectedStatus  int
		expectedBody    string
		expectedHeaders http.Header
	}{
		{
			name:            "simple",
			givenURL:        "/info/bidders",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-false",
			givenURL:        "/info/bidders?enabledonly=false",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-false-caseinsensitive",
			givenURL:        "/info/bidders?enabledonly=fAlSe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-true",
			givenURL:        "/info/bidders?enabledonly=true",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","x"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-true-caseinsensitive",
			givenURL:        "/info/bidders?enabledonly=TrUe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","x"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-invalid",
			givenURL:        "/info/bidders?enabledonly=foo",
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `Invalid value for 'enabledonly' query param, must be of boolean type`,
			expectedHeaders: http.Header{},
		},
		{
			name:            "enabledonly-missing",
			givenURL:        "/info/bidders?enabledonly=",
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `Invalid value for 'enabledonly' query param, must be of boolean type`,
			expectedHeaders: http.Header{},
		},
		{
			name:            "baseonly-false",
			givenURL:        "/info/bidders?baseadaptersonly=false",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "baseonly-false-caseinsensitive",
			givenURL:        "/info/bidders?baseadaptersonly=fAlSe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d","x","y"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "baseonly-true",
			givenURL:        "/info/bidders?baseadaptersonly=true",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","c"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "baseonly-true-caseinsensitive",
			givenURL:        "/info/bidders?baseadaptersonly=TrUe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","c"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "baseonly-invalid",
			givenURL:        "/info/bidders?baseadaptersonly=foo",
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `Invalid value for 'baseadaptersonly' query param, must be of boolean type`,
			expectedHeaders: http.Header{},
		},
		{
			name:            "baseonly-missing",
			givenURL:        "/info/bidders?baseadaptersonly=",
			expectedStatus:  http.StatusBadRequest,
			expectedBody:    `Invalid value for 'baseadaptersonly' query param, must be of boolean type`,
			expectedHeaders: http.Header{},
		},
		{
			name:            "enabledonly-true-baseonly-false",
			givenURL:        "/info/bidders?enabledonly=true&baseadaptersonly=false",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","x"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-false-baseonly-true",
			givenURL:        "/info/bidders?enabledonly=false&baseadaptersonly=true",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","c"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-true-baseonly-true",
			givenURL:        "/info/bidders?enabledonly=true&baseadaptersonly=true",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			handler := NewBiddersEndpoint(bidders, aliases)

			request := httptest.NewRequest("GET", test.givenURL, nil)

			responseRecorder := httptest.NewRecorder()
			handler(responseRecorder, request, nil)

			result := responseRecorder.Result()
			assert.Equal(t, result.StatusCode, test.expectedStatus)

			resultBody, _ := io.ReadAll(result.Body)
			assert.Equal(t, []byte(test.expectedBody), resultBody)

			resultHeaders := result.Header
			assert.Equal(t, test.expectedHeaders, resultHeaders)
		})
	}
}
