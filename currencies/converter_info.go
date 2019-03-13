package currencies

import "time"

// ConverterInfo holds information about converter setup
type ConverterInfo interface {
	Source() string
	FetchingInterval() time.Duration
	LastUpdated() time.Time
	Rates() *map[string]map[string]float64
	AdditionalInfo() interface{}
}

type converterInfo struct {
	source           string
	fetchingInterval time.Duration
	lastUpdated      time.Time
	rates            *map[string]map[string]float64
	additionalInfo   interface{}
}

// Source returns converter's URL source
func (ci converterInfo) Source() string {
	return ci.source
}

// FetchingInterval returns converter's fetching interval in nanoseconds
func (ci converterInfo) FetchingInterval() time.Duration {
	return ci.fetchingInterval
}

// LastUpdated returns converter's last updated time
func (ci converterInfo) LastUpdated() time.Time {
	return ci.lastUpdated
}

// Rates returns converter's internal rates
func (ci converterInfo) Rates() *map[string]map[string]float64 {
	return ci.rates
}

// AdditionalInfo returns converter's additional infos
func (ci converterInfo) AdditionalInfo() interface{} {
	return ci.additionalInfo
}
