package currency

import "strings"

const (
	CustomRates = iota
	PBSRates
)

type RateEngines struct {
	conversions []Conversions
}

// NewRates creates a new Rates object holding currencies rates
func NewRateEngines(customRates, pbsRates Conversions) *RateEngines {
	return &RateEngines{
		conversions: []Conversions{customRates, pbsRates},
	}
}

// GetRate returns the conversion rate between two currencies prioritizing
// the customRates currency rate over that of the PBS currency rate service
// returns an error if both Conversions objects return error.
func (re *RateEngines) GetRate(from string, to string) (float64, error) {

	rate, err := re.conversions[CustomRates].GetRate(from, to)
	if err == nil || !strings.HasPrefix(err.Error(), `Currency conversion rate not found`) {
		// valid custom conversion rate was found, return this
		// value because custom rates take priority over PBS rates
		return rate, err
	}

	// because the custom rates' GetRate() call returned an error other than "conversion
	// rate not found", there's nothing wrong with the 3 letter currency code so let's
	// try the PBS rates instead
	return re.conversions[PBSRates].GetRate(from, to)
}

// No need to call GetRates on RateEngines
func (r *RateEngines) GetRates() *map[string]map[string]float64 {
	return nil
}
