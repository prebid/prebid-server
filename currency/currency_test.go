package currency

import (
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/ptrutil"
	"github.com/stretchr/testify/assert"
)

func TestGetAuctionCurrencyRates(t *testing.T) {
	pbsRates := map[string]map[string]float64{
		"MXN": {
			"USD": 20.13,
			"EUR": 27.82,
			"JPY": 5.09, // "MXN" to "JPY" rate not found in customRates
		},
	}

	customRates := map[string]map[string]float64{
		"MXN": {
			"USD": 25.00, // different rate than in pbsRates
			"EUR": 27.82, // same as in pbsRates
			"GBP": 31.12, // not found in pbsRates at all
		},
	}

	expectedRateEngineRates := map[string]map[string]float64{
		"MXN": {
			"USD": 25.00, // rates engine will prioritize the value found in custom rates
			"EUR": 27.82, // same value in both the engine reads the custom entry first
			"JPY": 5.09,  // the engine will find it in the pbsRates conversions
			"GBP": 31.12, // the engine will find it in the custom conversions
		},
	}

	setupMockRateConverter := func(pbsRates map[string]map[string]float64) *RateConverter {
		if pbsRates == nil {
			return nil
		}

		jsonPbsRates, err := jsonutil.Marshal(pbsRates)
		if err != nil {
			t.Fatalf("Failed to marshal PBS rates: %v", err)
		}

		// Init mock currency conversion service
		mockCurrencyClient := &MockCurrencyRatesHttpClient{
			ResponseBody: `{"dataAsOf":"2018-09-12","conversions":` + string(jsonPbsRates) + `}`,
		}

		return NewRateConverter(
			mockCurrencyClient,
			"currency.fake.com",
			24*time.Hour,
		)
	}

	type args struct {
		currencyConverter *RateConverter
		requestRates      *openrtb_ext.ExtRequestCurrency
	}
	tests := []struct {
		name        string
		args        args
		assertRates map[string]map[string]float64
	}{
		{
			name: "valid ConversionRates, valid pbsRates, false UsePBSRates. Resulting rates identical to customRates",
			args: args{
				currencyConverter: setupMockRateConverter(pbsRates),
				requestRates: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     ptrutil.ToPtr(false),
				},
			},
			assertRates: customRates,
		},
		{
			name: "valid ConversionRates, valid pbsRates, true UsePBSRates. Resulting rates are a mix but customRates gets priority",
			args: args{
				currencyConverter: setupMockRateConverter(pbsRates),
				requestRates: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     ptrutil.ToPtr(true),
				},
			},
			assertRates: expectedRateEngineRates,
		},
		{
			name: "valid ConversionRates, nil pbsRates, false UsePBSRates. Resulting rates identical to customRates",
			args: args{
				currencyConverter: nil,
				requestRates: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     ptrutil.ToPtr(false),
				},
			},
			assertRates: customRates,
		},
		{
			name: "valid ConversionRates, nil pbsRates, true UsePBSRates. Resulting rates identical to customRates",
			args: args{
				currencyConverter: nil,
				requestRates: &openrtb_ext.ExtRequestCurrency{
					ConversionRates: customRates,
					UsePBSRates:     ptrutil.ToPtr(true),
				},
			},
			assertRates: customRates,
		},
		{
			name: "empty ConversionRates, valid pbsRates, false UsePBSRates. Because pbsRates cannot be used, disable currency conversion",
			args: args{
				currencyConverter: setupMockRateConverter(pbsRates),
				requestRates: &openrtb_ext.ExtRequestCurrency{
					// ConversionRates inCustomRates not initialized makes for a zero-length map
					UsePBSRates: ptrutil.ToPtr(false),
				},
			},
			assertRates: nil,
		},
		{
			name: "nil ConversionRates, valid pbsRates, true UsePBSRates. Resulting rates will be identical to pbsRates",
			args: args{
				currencyConverter: setupMockRateConverter(pbsRates),
				requestRates:      nil,
			},
			assertRates: pbsRates,
		},
		{
			name: "empty ConversionRates, nil pbsRates, false UsePBSRates. No conversion rates available, disable currency conversion",
			args: args{
				currencyConverter: setupMockRateConverter(pbsRates),
				requestRates: &openrtb_ext.ExtRequestCurrency{
					// ConversionRates inCustomRates not initialized makes for a zero-length map
					UsePBSRates: ptrutil.ToPtr(false),
				},
			},
			assertRates: nil,
		},

		{
			name: "empty ConversionRates, nil pbsRates, true UsePBSRates. No conversion rates available, disable currency conversion",
			args: args{
				currencyConverter: nil,
				requestRates: &openrtb_ext.ExtRequestCurrency{
					// ConversionRates inCustomRates not initialized makes for a zero-length map
					UsePBSRates: ptrutil.ToPtr(true),
				},
			},
			assertRates: nil,
		},
		{
			name: "nil customRates, nil pbsRates. No conversion rates available, disable currency conversion",
			args: args{
				currencyConverter: nil,
				requestRates:      nil,
			},
			assertRates: nil,
		},
		{
			name: "empty ConversionRates, valid pbsRates, true UsePBSRates. Resulting rates will be identical to pbsRates",
			args: args{
				currencyConverter: setupMockRateConverter(pbsRates),
				requestRates: &openrtb_ext.ExtRequestCurrency{
					// ConversionRates inCustomRates not initialized makes for a zero-length map
					UsePBSRates: ptrutil.ToPtr(true),
				},
			},
			assertRates: pbsRates,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.currencyConverter != nil {
				tt.args.currencyConverter.Run()
			}
			auctionRates := GetAuctionCurrencyRates(tt.args.currencyConverter, tt.args.requestRates)
			if tt.args.currencyConverter == nil && tt.args.requestRates == nil && tt.assertRates == nil {
				assert.Nil(t, auctionRates)
			} else if tt.assertRates == nil {
				rate, err := auctionRates.GetRate("USD", "MXN")
				assert.Error(t, err, tt.name)
				assert.Equal(t, float64(0), rate, tt.name)
			} else {
				for fromCurrency, rates := range tt.assertRates {
					for toCurrency, expectedRate := range rates {
						actualRate, err := auctionRates.GetRate(fromCurrency, toCurrency)
						assert.NoError(t, err, tt.name)
						assert.Equal(t, expectedRate, actualRate, tt.name)
					}
				}
			}
		})
	}
}
