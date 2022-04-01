package currency

import (
	"errors"

	"golang.org/x/text/currency"
)

// Rates holds data as represented on https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json
// note that `DataAsOfRaw` field is needed when parsing remote JSON as the date format if not standard and requires
// custom parsing to be properly set as Golang time.Time
type Rates struct {
	Conversions map[string]map[string]float64 `json:"conversions"`
}

// NewRates creates a new Rates object holding currencies rates
func NewRates(conversions map[string]map[string]float64) *Rates {
	return &Rates{
		Conversions: conversions,
	}
}

// GetRate returns the conversion rate between two currencies or:
//  - An error if one of the currency strings is not well-formed
//  - An error if any of the currency strings is not a recognized currency code.
//  - A ConversionNotFoundError in case the conversion rate between the two
//    given currencies is not in the currencies rates map
func (r *Rates) GetRate(from, to string) (float64, error) {
	var err error
	fromUnit, err := currency.ParseISO(from)
	if err != nil {
		return 0, err
	}
	toUnit, err := currency.ParseISO(to)
	if err != nil {
		return 0, err
	}
	if fromUnit.String() == toUnit.String() {
		return 1, nil
	}
	if r.Conversions != nil {
		if conversion, present := r.Conversions[fromUnit.String()][toUnit.String()]; present {
			// In case we have an entry FROM -> TO
			return conversion, nil
		} else if conversion, present := r.Conversions[toUnit.String()][fromUnit.String()]; present {
			// In case we have an entry TO -> FROM
			return 1 / conversion, nil
		}
		return 0, ConversionNotFoundError{FromCur: fromUnit.String(), ToCur: toUnit.String()}
	}
	return 0, errors.New("rates are nil")
}

// GetRates returns current rates
func (r *Rates) GetRates() *map[string]map[string]float64 {
	return &r.Conversions
}
