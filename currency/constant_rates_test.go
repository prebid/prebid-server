package currency

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRate_ConstantRates(t *testing.T) {

	// Setup:
	rates := NewConstantRates()

	testCases := []struct {
		from         string
		to           string
		expectedRate float64
		hasError     bool
	}{
		{from: "USD", to: "GBP", expectedRate: 0, hasError: true},
		{from: "GBP", to: "USD", expectedRate: 0, hasError: true},
		{from: "GBP", to: "EUR", expectedRate: 0, hasError: true},
		{from: "CNY", to: "EUR", expectedRate: 0, hasError: true},
		{from: "", to: "EUR", expectedRate: 0, hasError: true},
		{from: "CNY", to: "", expectedRate: 0, hasError: true},
		{from: "", to: "", expectedRate: 0, hasError: true},
		{from: "USD", to: "USD", expectedRate: 1, hasError: false},
		{from: "EUR", to: "EUR", expectedRate: 1, hasError: false},
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

func TestGetRate_ConstantRates_NotValidISOCurrency(t *testing.T) {

	// Setup:
	rates := NewConstantRates()

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
