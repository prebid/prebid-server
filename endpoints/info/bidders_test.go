package info

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prebid/prebid-server/v3/config"
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
			name:         "core-many-mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": enabledCore},
			expected:     `["a","b"]`,
		},
		{
			name:         "core-many-sorted",
			givenBidders: config.BidderInfos{"z": enabledCore, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "alias-one-enabled",
			givenBidders: config.BidderInfos{"a": enabledAlias},
			expected:     `["a"]`,
		},
		{
			name:         "alias-one-disabled",
			givenBidders: config.BidderInfos{"a": disabledAlias},
			expected:     `["a"]`,
		},
		{
			name:         "alias-many-mixed",
			givenBidders: config.BidderInfos{"a": enabledAlias, "b": disabledAlias},
			expected:     `["a","b"]`,
		},
		{
			name:         "alias-many-sorted",
			givenBidders: config.BidderInfos{"z": enabledAlias, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			expected:     `["a","b","c","d"]`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := prepareBiddersResponseAll(test.givenBidders)
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
			name:         "core-many-mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": enabledCore},
			expected:     `["a","b"]`,
		},
		{
			name:         "core-many-sorted",
			givenBidders: config.BidderInfos{"z": enabledCore, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "alias-one-enabled",
			givenBidders: config.BidderInfos{"a": enabledAlias},
			expected:     `[]`,
		},
		{
			name:         "alias-one-disabled",
			givenBidders: config.BidderInfos{"a": disabledAlias},
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
			name:         "core-many-mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": enabledCore},
			expected:     `["b"]`,
		},
		{
			name:         "core-many-sorted",
			givenBidders: config.BidderInfos{"z": enabledCore, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "alias-one-enabled",
			givenBidders: config.BidderInfos{"a": enabledAlias},
			expected:     `["a"]`,
		},
		{
			name:         "alias-one-disabled",
			givenBidders: config.BidderInfos{"a": disabledAlias},
			expected:     `[]`,
		},
		{
			name:         "alias-many-mixed",
			givenBidders: config.BidderInfos{"a": enabledAlias, "b": disabledAlias},
			expected:     `["a"]`,
		},
		{
			name:         "alias-many-sorted",
			givenBidders: config.BidderInfos{"z": enabledAlias, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": disabledAlias, "c": enabledCore, "d": enabledAlias},
			expected:     `["c","d"]`,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			result, err := prepareBiddersResponseEnabledOnly(test.givenBidders)
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
			name:         "core-many",
			givenBidders: config.BidderInfos{"a": enabledCore, "b": enabledCore},
			expected:     `["a","b"]`,
		},
		{
			name:         "core-many-mixed",
			givenBidders: config.BidderInfos{"a": disabledCore, "b": enabledCore},
			expected:     `["b"]`,
		},
		{
			name:         "core-many-sorted",
			givenBidders: config.BidderInfos{"z": enabledCore, "a": enabledCore},
			expected:     `["a","z"]`,
		},
		{
			name:         "alias-one-enabled",
			givenBidders: config.BidderInfos{"a": enabledAlias},
			expected:     `[]`,
		},
		{
			name:         "alias-one-disabled",
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
			expectedBody:    `["a","b","c","d"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-false",
			givenURL:        "/info/bidders?enabledonly=false",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-false-caseinsensitive",
			givenURL:        "/info/bidders?enabledonly=fAlSe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-true",
			givenURL:        "/info/bidders?enabledonly=true",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "enabledonly-true-caseinsensitive",
			givenURL:        "/info/bidders?enabledonly=TrUe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b"]`,
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
			expectedBody:    `["a","b","c","d"]`,
			expectedHeaders: http.Header{"Content-Type": []string{"application/json"}},
		},
		{
			name:            "baseonly-false-caseinsensitive",
			givenURL:        "/info/bidders?baseadaptersonly=fAlSe",
			expectedStatus:  http.StatusOK,
			expectedBody:    `["a","b","c","d"]`,
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
			expectedBody:    `["a","b"]`,
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
			handler := NewBiddersEndpoint(bidders)

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
