package currencies_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/currencies"
)

func TestUnMarshallRates(t *testing.T) {

	// Setup:
	testCases := []struct {
		ratesJSON     string
		expectedRates currencies.Rates
		expectsError  bool
	}{
		{
			ratesJSON: `{
				"dataAsOf":"2018-09-12",
				"conversions":{
					"USD":{
						"GBP":0.7662523901
					},
					"GBP":{
						"USD":1.3050530256
					}
				}
			}`,
			expectedRates: currencies.Rates{
				DataAsOf: time.Date(2018, time.September, 12, 0, 0, 0, 0, time.UTC),
				Conversions: map[string]map[string]float64{
					"USD": {
						"GBP": 0.7662523901,
					},
					"GBP": {
						"USD": 1.3050530256,
					},
				},
			},
			expectsError: false,
		},
		{
			ratesJSON: `{
				"dataAsOf":"",
				"conversions":{
					"USD":{
						"GBP":0.7662523901
					},
					"GBP":{
						"USD":1.3050530256
					}
				}
			}`,
			expectedRates: currencies.Rates{
				DataAsOf: time.Time{},
				Conversions: map[string]map[string]float64{
					"USD": {
						"GBP": 0.7662523901,
					},
					"GBP": {
						"USD": 1.3050530256,
					},
				},
			},
			expectsError: false,
		},
		{
			ratesJSON: `{
				"dataAsOf":"blabla",
				"conversions":{
					"USD":{
						"GBP":0.7662523901
					},
					"GBP":{
						"USD":1.3050530256
					}
				}
			}`,
			expectedRates: currencies.Rates{
				DataAsOf: time.Time{},
				Conversions: map[string]map[string]float64{
					"USD": {
						"GBP": 0.7662523901,
					},
					"GBP": {
						"USD": 1.3050530256,
					},
				},
			},
			expectsError: false,
		},
		{
			ratesJSON: `{
				"dataAsOf":"blabla",
				"conversions":{
					"USD":{
						"GBP":0.7662523901,
					},
					"GBP":{
						"USD":1.3050530256,
					}
				}
			}`,
			expectedRates: currencies.Rates{},
			expectsError:  true,
		},
	}

	for _, tc := range testCases {
		// Execute:
		updatedRates := currencies.Rates{}
		err := json.Unmarshal([]byte(tc.ratesJSON), &updatedRates)

		// Verify:
		assert.Equal(t, err != nil, tc.expectsError)
		assert.Equal(t, tc.expectedRates, updatedRates, "Rates weren't the expected ones")
	}
}

func TestGetRate(t *testing.T) {

	// Setup:
	rates := currencies.NewRates(time.Now(), map[string]map[string]float64{
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
	rates := currencies.NewRates(time.Now(), map[string]map[string]float64{
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
	rates := currencies.NewRates(time.Time{}, nil)

	// Execute:
	rate, err := rates.GetRate("USD", "EUR")

	// Verify:
	assert.NotNil(t, err, "err shouldn't be nil")
	assert.Equal(t, float64(0), rate, "rate should be 0")
}

func TestGetRate_NotValidISOCurrency(t *testing.T) {

	// Setup:
	rates := currencies.NewRates(time.Time{}, nil)

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
