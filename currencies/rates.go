package currencies

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Rates holds data as represented on https://cdn.jsdelivr.net/gh/prebid/currency-file@1/latest.json
// note that `DataAsOfRaw` field is needed when parsing remote JSON as the date format if not standard and requires
// custom parsing to be properly set as Golang time.Time
type Rates struct {
	DataAsOf    time.Time                     `json:"dataAsOf"`
	Conversions map[string]map[string]float64 `json:"conversions"`
}

// NewRates creates a new Rates object holding currencies rates
func NewRates(dataAsOf time.Time, conversions map[string]map[string]float64) *Rates {
	return &Rates{
		DataAsOf:    dataAsOf,
		Conversions: conversions,
	}
}

// UnmarshalJSON unmarshal raw JSON bytes to Rates object
func (r *Rates) UnmarshalJSON(b []byte) error {
	c := &struct {
		DataAsOf    string                        `json:"dataAsOf"`
		Conversions map[string]map[string]float64 `json:"conversions"`
	}{}
	if err := json.Unmarshal(b, &c); err != nil {
		return err
	}

	r.Conversions = c.Conversions

	layout := "2006-01-02"
	if date, err := time.Parse(layout, c.DataAsOf); err == nil {
		r.DataAsOf = date
	}

	return nil
}

// GetRate returns the conversion rate between two currencies
// returns an error in case the conversion rate between the two given currencies is not in the currencies rates map
func (r *Rates) GetRate(from string, to string) (float64, error) {
	if r.Conversions != nil {
		if conversion, present := r.Conversions[from][to]; present == true {
			return conversion, nil
		}
		return 0, fmt.Errorf("conversion %s->%s not present in rates dictionary", from, to)
	}
	return 0, errors.New("rates are nil")
}
