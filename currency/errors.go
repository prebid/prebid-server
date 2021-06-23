package currency

import "fmt"

// ConversionRateNotFoundError is thrown by the currency.Conversions GetRate(from string, to string) method
// when the conversion rate between the two currencies, nor its reciprocal, can be found.
type ConversionRateNotFoundError struct {
	FromCur, ToCur string
}

func (err ConversionRateNotFoundError) Error() string {
	return fmt.Sprintf("Currency conversion rate not found: '%s' => '%s'", err.FromCur, err.ToCur)
}
