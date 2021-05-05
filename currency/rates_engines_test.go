package currency

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateEnginesGetRate(t *testing.T) {

	// Setup:
	customRates := NewRates(time.Now(), map[string]map[string]float64{
		"USD": {
			"GBP": 0.77208,
			"EUR": 0.80,
		},
	})

	pbsRates := NewRates(time.Now(), map[string]map[string]float64{
		"USD": {
			"GBP": 0.50,
			"MXN": 10.31,
		},
	})
	rateEngines := NewRateEngines(customRates, pbsRates)

	// Test cases:
	type aTest struct {
		desc         string
		from         string
		to           string
		expectedRate float64
	}

	testGroups := []struct {
		expectError bool
		testCases   []aTest
	}{
		{
			expectError: false,
			testCases: []aTest{
				{"Found in both, return custom rate", "USD", "GBP", 0.77208},
				{"Found in both, return inverse custom rate", "GBP", "USD", 1 / 0.77208},
				{"Found in custom rates only", "USD", "EUR", 0.80},
				{"Found in PBS rates only", "USD", "MXN", 10.31},
				{"Found in PBS rates only, return inverse", "MXN", "USD", 1 / 10.31},
				{"Same currency, return unitary rate", "USD", "USD", 1},
			},
		},
		{
			expectError: true,
			testCases: []aTest{
				{"From-currency three-digit code not found", "FOO", "EUR", 0},
				{"From-currency three-digit code malformed", "XX", "EUR", 0},
				{"To-currency three-digit code not found", "GBP", "BAR", 0},
				{"To-currency three-digit code malformed", "GBP", "", 0},
				{"Both currencies malformed", "", "", 0},
				{"Valid three-digit currency codes, but conversion rate not found", "GBP", "EUR", 0},
			},
		},
	}

	for _, group := range testGroups {
		for _, tc := range group.testCases {
			// Execute:
			rate, err := rateEngines.GetRate(tc.from, tc.to)

			// Verify:
			if group.expectError {
				assert.NotNilf(t, err, "err shouldn't be nil: %s\n", tc.desc)
				assert.Equal(t, float64(0), rate, "rate should be 0: %s\n", tc.desc)
			} else {
				assert.Nil(t, err, "err should be nil: %s\n", tc.desc)
				assert.Equal(t, tc.expectedRate, rate, "rate doesn't match the expected: %s\n", tc.desc)
			}
		}
	}
}
