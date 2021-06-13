package currency

import "fmt"

// ConversionRateNotFound is thrown by the currency.Conversions GetRate(from string, to string) method
// when the conversion rate between the two currencies, nor its reciprocal, can be found.
type ConversionRateNotFound struct {
	FromCur, ToCur string
}

func (err ConversionRateNotFound) Error() string {
	return fmt.Sprintf("Currency conversion rate not found: '%s' => '%s'", err.FromCur, err.ToCur)
}
