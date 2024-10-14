package currency

import "fmt"

// ConversionNotFoundError is thrown by the currency.Conversions GetRate(from string, to string) method
// when the conversion rate between the two currencies, nor its reciprocal, can be found.
type ConversionNotFoundError struct {
	FromCur, ToCur string
}

func (err ConversionNotFoundError) Error() string {
	return fmt.Sprintf("Currency conversion rate not found: '%s' => '%s'", err.FromCur, err.ToCur)
}
