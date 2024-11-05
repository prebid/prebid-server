package currency

import (
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/errortypes"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
	"github.com/prebid/prebid-server/v3/util/timeutil"
)

// RateConverter holds the currencies conversion rates dictionary
type RateConverter struct {
	httpClient          httpClient
	staleRatesThreshold time.Duration
	syncSourceURL       string
	rates               atomic.Value // Should only hold Rates struct
	lastUpdated         atomic.Value // Should only hold time.Time
	constantRates       Conversions
	time                timeutil.Time
}

// NewRateConverter returns a new RateConverter
func NewRateConverter(
	httpClient httpClient,
	syncSourceURL string,
	staleRatesThreshold time.Duration,
) *RateConverter {
	return &RateConverter{
		httpClient:          httpClient,
		staleRatesThreshold: staleRatesThreshold,
		syncSourceURL:       syncSourceURL,
		rates:               atomic.Value{},
		lastUpdated:         atomic.Value{},
		constantRates:       NewConstantRates(),
		time:                &timeutil.RealTime{},
	}
}

// fetch allows to retrieve the currencies rates from the syncSourceURL provided
func (rc *RateConverter) fetch() (*Rates, error) {
	request, err := http.NewRequest("GET", rc.syncSourceURL, nil)
	if err != nil {
		return nil, err
	}

	response, err := rc.httpClient.Do(request)
	if err != nil {
		return nil, err
	}

	if response.StatusCode >= 400 {
		message := fmt.Sprintf("The currency rates request failed with status code %d", response.StatusCode)
		return nil, &errortypes.BadServerResponse{Message: message}
	}

	defer response.Body.Close()

	bytesJSON, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	updatedRates := &Rates{}
	err = jsonutil.UnmarshalValid(bytesJSON, updatedRates)
	if err != nil {
		return nil, err
	}

	return updatedRates, err
}

// Update updates the internal currencies rates from remote sources
func (rc *RateConverter) update() error {
	rates, err := rc.fetch()
	if err == nil {
		rc.rates.Store(rates)
		rc.lastUpdated.Store(rc.time.Now())
	} else {
		if rc.checkStaleRates() {
			rc.clearRates()
			glog.Errorf("Error updating conversion rates, falling back to constant rates: %v", err)
		} else {
			glog.Errorf("Error updating conversion rates: %v", err)
		}
	}

	return err
}

func (rc *RateConverter) Run() error {
	return rc.update()
}

// LastUpdated returns time when currencies rates were updated
func (rc *RateConverter) LastUpdated() time.Time {
	if lastUpdated := rc.lastUpdated.Load(); lastUpdated != nil {
		return lastUpdated.(time.Time)
	}
	return time.Time{}
}

// Rates returns current conversions rates
func (rc *RateConverter) Rates() Conversions {
	// atomic.Value field rates is an empty interface and will be of type *Rates the first time rates are stored
	// or nil if the rates have never been stored
	if rates := rc.rates.Load(); rates != (*Rates)(nil) && rates != nil {
		return rates.(*Rates)
	}
	return rc.constantRates
}

// clearRates sets the rates to nil
func (rc *RateConverter) clearRates() {
	// atomic.Value field rates must be of type *Rates so we cast nil to that type
	rc.rates.Store((*Rates)(nil))
}

// checkStaleRates checks if loaded third party conversion rates are stale
func (rc *RateConverter) checkStaleRates() bool {
	if rc.staleRatesThreshold <= 0 {
		return false
	}

	currentTime := rc.time.Now().UTC()
	if lastUpdated := rc.lastUpdated.Load(); lastUpdated != nil {
		delta := currentTime.Sub(lastUpdated.(time.Time).UTC())
		if delta.Seconds() > rc.staleRatesThreshold.Seconds() {
			return true
		}
	}
	return false
}

// GetInfo returns setup information about the converter
func (rc *RateConverter) GetInfo() ConverterInfo {
	var rates *map[string]map[string]float64 = rc.Rates().GetRates()
	return converterInfo{
		source:      rc.syncSourceURL,
		lastUpdated: rc.LastUpdated(),
		rates:       rates,
	}
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Conversions allows to get a conversion rate between two currencies.
// if one of the currency string is not a currency or if there is not conversion between those
// currencies, then an err is returned and rate is 0.
type Conversions interface {
	GetRate(from string, to string) (float64, error)
	GetRates() *map[string]map[string]float64
}
