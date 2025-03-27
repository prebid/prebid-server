package currency

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"golang.org/x/text/currency"
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
			expectedError: errors.New("expect { or n, but found m"),
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
			desc: "Valid JSON field defines a conversions map with repeated entries, last one wins",
			ratesJSON: `{
				"conversions":{
					"USD":{
						"GBP":0.7662523901,
						"MXN":20.00
					},
					"USD":{
						"GBP":0.4815162342
					}
				}
			}`,
			expectedRates: Rates{
				Conversions: map[string]map[string]float64{
					"USD": {
						"GBP": 0.4815162342,
					},
				},
			},
			expectsError:  false,
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		// Execute:
		updatedRates := Rates{}
		err := jsonutil.UnmarshalValid([]byte(tc.ratesJSON), &updatedRates)

		// Verify:
		assert.Equal(t, err != nil, tc.expectsError, tc.desc)
		if tc.expectsError {
			assert.Equal(t, tc.expectedError.Error(), err.Error(), tc.desc)
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

func TestGetRate_FindIntermediateConversionRate(t *testing.T) {
	rates := NewRates(map[string]map[string]float64{
		"USD": {
			"SEK": 10.23842,
			"NOK": 10.47089,
		},
		"EUR": {
			"THB": 35.23842,
			"ZAR": 18.47089,
		},
	})

	testCases := []struct {
		description  string
		from         string
		to           string
		expectedRate float64
		hasError     bool
	}{
		{
			description:  "in_same_intermediate_USD_currency",
			from:         "NOK",
			to:           "SEK",
			expectedRate: 0.9777984488424574,
		},
		{
			description:  "in_same_intermediate_USD_currency_inverse",
			from:         "SEK",
			to:           "NOK",
			expectedRate: 1 / 0.9777984488424574,
		},
		{
			description:  "in_same_intermediate_EUR_currency",
			from:         "THB",
			to:           "ZAR",
			expectedRate: 0.5241690745498806,
		},
		{
			description:  "in_same_intermediate_EUR_currency_inverse",
			from:         "ZAR",
			to:           "THB",
			expectedRate: 1 / 0.5241690745498806,
		},
		{
			description:  "in_different_intermediate_currencies",
			from:         "NOK",
			to:           "ZAR",
			expectedRate: 0,
			hasError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			fromUnit, err := currency.ParseISO(tc.from)
			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}
			toUnit, err := currency.ParseISO(tc.to)
			if err != nil {
				t.Errorf("Expected no error, but got: %v", err)
			}

			rate, err := FindIntermediateConversionRate(rates, fromUnit, toUnit)

			if tc.hasError {
				assert.NotNil(t, err)
				assert.Equal(t, float64(0), rate)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tc.expectedRate, rate)
			}
		})
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
