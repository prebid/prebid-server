package currency

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnMarshallRates(t *testing.T) {
	// Setup:
	testCases := []struct {
		desc          string
		ratesJSON     string
		expectedRates Rates
		expectsError  bool
		expectedError error
	}{
		{
			desc:          "malformed JSON object, return error",
			ratesJSON:     `malformed`,
			expectedRates: Rates{},
			expectsError:  true,
			expectedError: errors.New("invalid character 'm' looking for beginning of value"),
		},
		{
			desc: "Valid JSON field defining valid conversion object. Expect no error",
			ratesJSON: `{
				"conversions":{
					"USD":{
						"GBP":0.7662523901
					},
					"GBP":{
						"USD":1.3050530256
					}
				}
			}`,
			expectedRates: Rates{
				Conversions: map[string]map[string]float64{
					"USD": {
						"GBP": 0.7662523901,
					},
					"GBP": {
						"USD": 1.3050530256,
					},
				},
			},
			expectsError:  false,
			expectedError: nil,
		},
		{
			desc: "Valid JSON field defines a conversions map with repeated entries, expect error",
			ratesJSON: `{
				"conversions":{
					"USD":{
						"GBP":0.7662523901,
						"MXN":20.00
					},
					"USD":{
						"GBP":0.7662523901
					},
				}
			}`,
			expectedRates: Rates{},
			expectsError:  true,
			expectedError: errors.New("invalid character '}' looking for beginning of object key string"),
		},
	}

	for _, tc := range testCases {
		// Execute:
		updatedRates := Rates{}
		err := json.Unmarshal([]byte(tc.ratesJSON), &updatedRates)

		// Verify:
		assert.Equal(t, err != nil, tc.expectsError, tc.desc)
		if tc.expectsError {
			assert.Equal(t, err.Error(), tc.expectedError.Error(), tc.desc)
		}
		assert.Equal(t, tc.expectedRates, updatedRates, tc.desc)
	}
}

func TestGetRate(t *testing.T) {

	// Setup:
	rates := NewRates(map[string]map[string]float64{
		"USD": {
			"GBP": 0.77208,
		},
		"GBP": {
			"USD": 1.2952,
		},
	})

	testCases := []struct {
		from         string
		to           string
		expectedRate float64
		hasError     bool
	}{
		{from: "USD", to: "GBP", expectedRate: 0.77208, hasError: false},
		{from: "GBP", to: "USD", expectedRate: 1.2952, hasError: false},
		{from: "GBP", to: "EUR", expectedRate: 0, hasError: true},
		{from: "CNY", to: "EUR", expectedRate: 0, hasError: true},
		{from: "", to: "EUR", expectedRate: 0, hasError: true},
		{from: "CNY", to: "", expectedRate: 0, hasError: true},
		{from: "", to: "", expectedRate: 0, hasError: true},
		{from: "USD", to: "USD", expectedRate: 1, hasError: false},
	}

	for _, tc := range testCases {
		// Execute:
		rate, err := rates.GetRate(tc.from, tc.to)

		// Verify:
		if tc.hasError {
			assert.NotNil(t, err, "err shouldn't be nil")
			assert.Equal(t, float64(0), rate, "rate should be 0")
		} else {
			assert.Nil(t, err, "err should be nil")
			assert.Equal(t, tc.expectedRate, rate, "rate doesn't match the expected one")
		}
	}
}

func TestGetRate_ReverseConversion(t *testing.T) {

	// Setup:
	rates := NewRates(map[string]map[string]float64{
		"USD": {
			"GBP": 0.77208,
		},
		"EUR": {
			"USD": 0.88723,
		},
	})

	testCases := []struct {
		from         string
		to           string
		expectedRate float64
		description  string
	}{
		{
			from:         "USD",
			to:           "GBP",
			expectedRate: 0.77208,
			description:  "case 1 - Rate is present directly and will be returned as is",
		},
		{
			from:         "EUR",
			to:           "USD",
			expectedRate: 0.88723,
			description:  "case 2 - Rate is present directly and will be returned as is (2)",
		},
		{
			from:         "GBP",
			to:           "USD",
			expectedRate: 1 / 0.77208,
			description:  "case 3 - Rate is not present but the reverse one is, will return the computed rate from the reverse entry",
		},
		{
			from:         "USD",
			to:           "EUR",
			expectedRate: 1 / 0.88723,
			description:  "case 4 - Rate is not present but the reverse one is, will return the computed rate from the reverse entry (2)",
		},
	}

	for _, tc := range testCases {
		// Execute:
		rate, err := rates.GetRate(tc.from, tc.to)

		// Verify:
		assert.Nil(t, err, "err should be nil: "+tc.description)
		assert.Equal(t, tc.expectedRate, rate, "rate doesn't match the expected one: "+tc.description)
	}
}

func TestGetRate_EmptyRates(t *testing.T) {

	// Setup:
	rates := NewRates(nil)

	// Execute:
	rate, err := rates.GetRate("USD", "EUR")

	// Verify:
	assert.NotNil(t, err, "err shouldn't be nil")
	assert.Equal(t, float64(0), rate, "rate should be 0")
}

func TestGetRate_NotValidISOCurrency(t *testing.T) {

	// Setup:
	rates := NewRates(nil)

	testCases := []struct {
		from         string
		to           string
		expectedRate float64
		hasError     bool
	}{
		{from: "foo", to: "foo", expectedRate: 0, hasError: true},
		{from: "bar", to: "foo", expectedRate: 0, hasError: true},
	}

	for _, tc := range testCases {
		// Execute:
		rate, err := rates.GetRate(tc.from, tc.to)

		// Verify:
		if tc.hasError {
			assert.NotNil(t, err, "err shouldn't be nil")
			assert.Equal(t, float64(0), rate, "rate should be 0")
		} else {
			assert.Nil(t, err, "err should be nil")
			assert.Equal(t, tc.expectedRate, rate, "rate doesn't match the expected one")
		}
	}
}
