package currency

// AggregateConversions contains both the request-defined currency rate
// map found in request.ext.prebid.currency and the currencies conversion
// rates fetched with the RateConverter object defined in rate_converter.go
// It implements the Conversions interface.
type AggregateConversions struct {
	customRates, serverRates Conversions
}

// NewAggregateConversions expects both customRates and pbsRates to not be nil
func NewAggregateConversions(customRates, pbsRates Conversions) *AggregateConversions {
	return &AggregateConversions{
		customRates: customRates,
		serverRates: pbsRates,
	}
}

// GetRate returns the conversion rate between two currencies prioritizing
// the customRates currency rate over that of the PBS currency rate service
// returns an error if both Conversions objects return error.
func (re *AggregateConversions) GetRate(from string, to string) (float64, error) {
	rate, err := re.customRates.GetRate(from, to)
	if err == nil {
		return rate, nil
	} else if _, isMissingRateErr := err.(ConversionNotFoundError); !isMissingRateErr {
		// other error, return the error
		return 0, err
	}

	// because the custom rates' GetRate() call returned an error other than "conversion
	// rate not found", there's nothing wrong with the 3 letter currency code so let's
	// try the PBS rates instead
	return re.serverRates.GetRate(from, to)
}

// GetRates is not implemented for AggregateConversions . There is no need to call
// this function for this scenario.
func (r *AggregateConversions) GetRates() *map[string]map[string]float64 {
	return nil
}
