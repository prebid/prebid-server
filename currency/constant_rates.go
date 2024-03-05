package currency

import (
	"golang.org/x/text/currency"
)

// ConstantRates doesn't do any currency conversions and accepts only conversions where
// both currencies (from and to) are the same.
// If not the same currencies, it returns an error.
type ConstantRates struct{}

// NewConstantRates creates a new ConstantRates object holding currencies rates
func NewConstantRates() *ConstantRates {
	return &ConstantRates{}
}

// GetRate returns 1 if both currencies are the same.
// If not, it will return an error.
func (r *ConstantRates) GetRate(from string, to string) (float64, error) {
	fromUnit, err := currency.ParseISO(from)
	if err != nil {
		return 0, err
	}
	toUnit, err := currency.ParseISO(to)
	if err != nil {
		return 0, err
	}

	if fromUnit.String() != toUnit.String() {
		return 0, ConversionNotFoundError{FromCur: fromUnit.String(), ToCur: toUnit.String()}
	}

	return 1, nil
}

// GetRates returns current rates
func (r *ConstantRates) GetRates() *map[string]map[string]float64 {
	return nil
}
