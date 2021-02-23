package currency

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/prebid/prebid-server/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestUnMarshallRates(t *testing.T) {

	// Setup:
	testCases := []struct {
		ratesJSON     string
		expectedRates Rates
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
			expectedRates: Rates{
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
			expectedRates: Rates{
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
			expectedRates: Rates{
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
			expectedRates: Rates{},
			expectsError:  true,
		},
	}

	for _, tc := range testCases {
		// Execute:
		updatedRates := Rates{}
		err := json.Unmarshal([]byte(tc.ratesJSON), &updatedRates)

		// Verify:
		assert.Equal(t, err != nil, tc.expectsError)
		assert.Equal(t, tc.expectedRates, updatedRates, "Rates weren't the expected ones")
	}
}

func TestGetRate(t *testing.T) {

	// Setup:
	rates := NewRates(time.Now(), map[string]map[string]float64{
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
	rates := NewRates(time.Now(), map[string]map[string]float64{
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
	rates := NewRates(time.Time{}, nil)

	// Execute:
	rate, err := rates.GetRate("USD", "EUR")

	// Verify:
	assert.NotNil(t, err, "err shouldn't be nil")
	assert.Equal(t, float64(0), rate, "rate should be 0")
}

func TestUpdateRates(t *testing.T) {

	boolTrue, boolFalse := true, false

	pbsRates := map[string]map[string]float64{
		"MXN": {
			"USD": 20.13,
			"EUR": 27.82,
			"JPY": 5.09, // "MXN" to "JPY" rate not found in customRates
		},
		// Euro exchange rates not found in customRates
		"EUR": {
			"JPY": 0.05,
			"MXN": 0.05,
			"USD": 0.92,
		},
	}

	customRates := map[string]map[string]float64{
		// "MXN" can also be found in pbsRates but maps to some rate values
		// that are different. This map also includes some currencies not found in pbsRates
		"MXN": {
			"USD": 25.00, // different rate than in pbsRates
			"EUR": 27.82, // same as in pbsRates
			"GBP": 31.12, // not found in pbsRates at all
		},
		// Currency can not be found in pbsRates add this entry as is
		"USD": {
			"GBP": 1.2,
			"MXN": 0.05,
			"CAN": 0.95,
		},
	}

	testCases := []struct {
		desc              string
		inPbsRates        map[string]map[string]float64
		inBidExtCurrency  *openrtb_ext.ExtRequestCurrency
		outResultingRates map[string]map[string]float64
	}{
		{
			desc:       "Valid Conversions objects, UsePBSRates set to false. Resulting rates will be identical to customRates",
			inPbsRates: pbsRates,
			inBidExtCurrency: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: customRates,
				UsePBSRates:     &boolFalse,
			},
			outResultingRates: customRates,
		},
		{
			desc:       "Valid Conversions objects, UsePBSRates set to true. Resulting rates will keep field values found in pbsRates but not found in customRates and the rest will be added or overwritten with customRates' values",
			inPbsRates: pbsRates,
			inBidExtCurrency: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: customRates,
				UsePBSRates:     &boolTrue,
			},
			outResultingRates: map[string]map[string]float64{
				// Currency was found in both pbsRates and customRates and got updated
				"MXN": {
					"USD": 25.00, //updated with customRates' value
					"EUR": 27.82, //same in pbsRates than in customRates, no update
					"GBP": 31.12, //added because it was found in customRates and not in pbsRates
					"JPY": 5.09,  //kept from pbsRates as it wasn't found in customRates
				},
				// Currency added as is because it wasn't found in pbsRates
				"USD": {
					"GBP": 1.2,
					"MXN": 0.05,
					"CAN": 0.95,
				},
				// customRates didn't have exchange rates for this currency, entry
				// kept from pbsRates
				"EUR": {
					"JPY": 0.05,
					"MXN": 0.05,
					"USD": 0.92,
				},
			},
		},
		{
			desc:       "pbsRates are nil, UsePBSRates set to false. Resulting rates will be identical to customRates",
			inPbsRates: nil,
			inBidExtCurrency: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: customRates,
				UsePBSRates:     &boolFalse,
			},
			outResultingRates: customRates,
		},
		{
			desc:       "pbsRates are nil, UsePBSRates set to true. Resulting rates will be identical to customRates",
			inPbsRates: nil,
			inBidExtCurrency: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: customRates,
				UsePBSRates:     &boolTrue,
			},
			outResultingRates: customRates,
		},
		{
			desc:       "customRates empty, UsePBSRates set to false. Resulting rates will be identical to pbsRates",
			inPbsRates: pbsRates,
			inBidExtCurrency: &openrtb_ext.ExtRequestCurrency{
				// ConversionRates inCustomRates not initialized makes for a zero-length map
				UsePBSRates: &boolFalse,
			},
			outResultingRates: pbsRates,
		},
		{
			desc:       "customRates empty, UsePBSRates set to true. Resulting rates will be identical to pbsRates",
			inPbsRates: pbsRates,
			inBidExtCurrency: &openrtb_ext.ExtRequestCurrency{
				// ConversionRates inCustomRates not initialized makes for a zero-length map
				UsePBSRates: &boolTrue,
			},
			outResultingRates: pbsRates,
		},
	}

	for _, tc := range testCases {
		// Test setup:
		rates := NewRates(time.Time{}, tc.inPbsRates)

		// Run test
		rates.UpdateRates(tc.inBidExtCurrency)

		// Assertions
		assert.Equal(t, tc.outResultingRates, *rates.GetRates(), tc.desc)
	}
}

func TestGetRate_NotValidISOCurrency(t *testing.T) {

	// Setup:
	rates := NewRates(time.Time{}, nil)

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
