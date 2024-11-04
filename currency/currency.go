package currency

import (
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

func GetAuctionCurrencyRates(currencyConverter *RateConverter, requestRates *openrtb_ext.ExtRequestCurrency) Conversions {
	if currencyConverter == nil && requestRates == nil {
		return nil
	}

	if requestRates == nil {
		// No bidRequest.ext.currency field was found, use PBS rates as usual
		return currencyConverter.Rates()
	}

	// currencyConverter will never be nil, refer main.serve(), adding this check for future usecases
	if currencyConverter == nil {
		return NewRates(requestRates.ConversionRates)
	}

	// If bidRequest.ext.currency.usepbsrates is nil, we understand its value as true. It will be false
	// only if it's explicitly set to false
	usePbsRates := requestRates.UsePBSRates == nil || *requestRates.UsePBSRates

	if !usePbsRates {
		// At this point, we can safely assume the ConversionRates map is not empty because
		// validateCustomRates(bidReqCurrencyRates *openrtb_ext.ExtRequestCurrency) would have
		// thrown an error under such conditions.
		return NewRates(requestRates.ConversionRates)
	}

	// Both PBS and custom rates can be used, check if ConversionRates is not empty
	if len(requestRates.ConversionRates) == 0 {
		// Custom rates map is empty, use PBS rates only
		return currencyConverter.Rates()
	}

	// Return an AggregateConversions object that includes both custom and PBS currency rates but will
	// prioritize custom rates over PBS rates whenever a currency rate is found in both
	return NewAggregateConversions(NewRates(requestRates.ConversionRates), currencyConverter.Rates())
}
