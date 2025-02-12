package currency

import (
	"testing"

	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
)

func TestValidateCustomRates(t *testing.T) {
	boolTrue := true
	boolFalse := false

	testCases := []struct {
		desc               string
		inBidReqCurrencies *openrtb_ext.ExtRequestCurrency
		outCurrencyError   error
	}{
		{
			desc:               "nil input, no errors expected",
			inBidReqCurrencies: nil,
			outCurrencyError:   nil,
		},
		{
			desc: "empty custom currency rates but UsePBSRates is set to false, we don't return error nor warning",
			inBidReqCurrencies: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: map[string]map[string]float64{},
				UsePBSRates:     &boolFalse,
			},
			outCurrencyError: nil,
		},
		{
			desc: "empty custom currency rates but UsePBSRates is set to true, no need to return error because we can use PBS rates",
			inBidReqCurrencies: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: map[string]map[string]float64{},
				UsePBSRates:     &boolTrue,
			},
			outCurrencyError: nil,
		},
		{
			desc: "UsePBSRates is nil and defaults to true, bidExt fromCurrency is invalid, expect bad input error",
			inBidReqCurrencies: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: map[string]map[string]float64{
					"FOO": {
						"GBP": 1.2,
						"MXN": 0.05,
						"JPY": 0.95,
					},
				},
			},
			outCurrencyError: &errortypes.BadInput{Message: "currency code FOO is not recognized or malformed"},
		},
		{
			desc: "UsePBSRates set to false, bidExt fromCurrency is invalid, expect bad input error",
			inBidReqCurrencies: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: map[string]map[string]float64{
					"FOO": {
						"GBP": 1.2,
						"MXN": 0.05,
						"JPY": 0.95,
					},
				},
				UsePBSRates: &boolFalse,
			},
			outCurrencyError: &errortypes.BadInput{Message: "currency code FOO is not recognized or malformed"},
		},
		{
			desc: "UsePBSRates set to false, some of the bidExt 'to' Currencies are invalid, expect bad input error when parsing the first invalid currency code",
			inBidReqCurrencies: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: map[string]map[string]float64{
					"USD": {
						"FOO": 10.0,
						"MXN": 0.05,
					},
				},
				UsePBSRates: &boolFalse,
			},
			outCurrencyError: &errortypes.BadInput{Message: "currency code FOO is not recognized or malformed"},
		},
		{
			desc: "UsePBSRates set to false, some of the bidExt 'from' and 'to' currencies are invalid, expect bad input error when parsing the first invalid currency code",
			inBidReqCurrencies: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: map[string]map[string]float64{
					"FOO": {
						"MXN": 0.05,
						"CAD": 0.95,
					},
				},
				UsePBSRates: &boolFalse,
			},
			outCurrencyError: &errortypes.BadInput{Message: "currency code FOO is not recognized or malformed"},
		},
		{
			desc: "All 3-digit currency codes exist, expect no error",
			inBidReqCurrencies: &openrtb_ext.ExtRequestCurrency{
				ConversionRates: map[string]map[string]float64{
					"USD": {
						"MXN": 0.05,
					},
					"MXN": {
						"JPY": 10.0,
						"EUR": 10.95,
					},
				},
				UsePBSRates: &boolFalse,
			},
		},
	}

	for _, tc := range testCases {
		actualErr := ValidateCustomRates(tc.inBidReqCurrencies)

		assert.Equal(t, tc.outCurrencyError, actualErr, tc.desc)
	}
}
