package currency

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGroupedGetRate(t *testing.T) {

	// Setup:
	customRates := NewRates(map[string]map[string]float64{
		"USD": {
			"GBP": 3.00,
			"EUR": 2.00,
		},
	})

	pbsRates := NewRates(map[string]map[string]float64{
		"USD": {
			"GBP": 4.00,
			"MXN": 10.00,
		},
	})
	aggregateConversions := NewAggregateConversions(customRates, pbsRates)

	// Test cases:
	type aTest struct {
		desc         string
		from         string
		to           string
		expectedRate float64
	}

	testGroups := []struct {
		expectedError error
		testCases     []aTest
	}{
		{
			expectedError: nil,
			testCases: []aTest{
				{"Found in both, return custom rate", "USD", "GBP", 3.00},
				{"Found in both, return inverse custom rate", "GBP", "USD", 1 / 3.00},
				{"Found in custom rates only", "USD", "EUR", 2.00},
				{"Found in PBS rates only", "USD", "MXN", 10.00},
				{"Found in PBS rates only, return inverse", "MXN", "USD", 1 / 10.00},
				{"Same currency, return unitary rate", "USD", "USD", 1},
			},
		},
		{
			expectedError: errors.New("currency: tag is not well-formed"),
			testCases: []aTest{
				{"From-currency three-digit code malformed", "XX", "EUR", 0},
				{"To-currency three-digit code malformed", "GBP", "", 0},
				{"Both currencies malformed", "", "", 0},
			},
		},
		{
			expectedError: errors.New("currency: tag is not a recognized currency"),
			testCases: []aTest{
				{"From-currency three-digit code not found", "FOO", "EUR", 0},
				{"To-currency three-digit code not found", "GBP", "BAR", 0},
			},
		},
		{
			expectedError: ConversionNotFoundError{FromCur: "GBP", ToCur: "EUR"},
			testCases: []aTest{
				{"Valid three-digit currency codes, but conversion rate not found", "GBP", "EUR", 0},
			},
		},
	}

	for _, group := range testGroups {
		for _, tc := range group.testCases {
			// Execute:
			rate, err := aggregateConversions.GetRate(tc.from, tc.to)

			// Verify:
			assert.Equal(t, tc.expectedRate, rate, "conversion rate doesn't match the expected rate: %s\n", tc.desc)
			if group.expectedError != nil {
				assert.Error(t, err, "error doesn't match expected: %s\n", tc.desc)
			} else {
				assert.NoError(t, err, "err should be nil: %s\n", tc.desc)
			}
		}
	}
}
