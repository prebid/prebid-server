package currency

import (
	"fmt"

	"golang.org/x/text/currency"

	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
)

// ValidateCustomRates throws a bad input error if any of the 3-digit currency codes found in
// the bidRequest.ext.prebid.currency field is invalid, malfomed or does not represent any actual
// currency. No error is thrown if bidRequest.ext.prebid.currency is invalid or empty.
func ValidateCustomRates(bidReqCurrencyRates *openrtb_ext.ExtRequestCurrency) error {
	if bidReqCurrencyRates == nil {
		return nil
	}

	for fromCurrency, rates := range bidReqCurrencyRates.ConversionRates {
		// Check if fromCurrency is a valid 3-letter currency code
		if _, err := currency.ParseISO(fromCurrency); err != nil {
			return &errortypes.BadInput{Message: fmt.Sprintf("currency code %s is not recognized or malformed", fromCurrency)}
		}

		// Check if currencies mapped to fromCurrency are valid 3-letter currency codes
		for toCurrency := range rates {
			if _, err := currency.ParseISO(toCurrency); err != nil {
				return &errortypes.BadInput{Message: fmt.Sprintf("currency code %s is not recognized or malformed", toCurrency)}
			}
		}
	}
	return nil
}
