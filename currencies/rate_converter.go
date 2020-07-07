package currencies

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
)

// RateConverter holds the currencies conversion rates dictionary
type RateConverter struct {
	httpClient          httpClient
	done                chan bool
	updateNotifier      chan<- int
	fetchingInterval    time.Duration
	staleRatesThreshold time.Duration
	syncSourceURL       string
	rates               atomic.Value // Should only hold Rates struct
	lastUpdated         atomic.Value // Should only hold time.Time
	constantRates       Conversions
}

// NewRateConverter returns a new RateConverter
func NewRateConverter(
	httpClient httpClient,
	syncSourceURL string,
	fetchingInterval time.Duration,
	staleRatesThreshold time.Duration,
) *RateConverter {
	return NewRateConverterWithNotifier(
		httpClient,
		syncSourceURL,
		fetchingInterval,
		staleRatesThreshold,
		nil, // no notifier channel specified, won't send any notifications
	)
}

// NewRateConverterDefault returns a RateConverter with default values.
// By default there will be no currencies conversions done.
// `currencies.ConstantRate` will be used.
func NewRateConverterDefault() *RateConverter {
	return NewRateConverter(&http.Client{}, "", time.Duration(0), time.Duration(0))
}

// NewRateConverterWithNotifier returns a new RateConverter
// it allow to pass an update chan in which the number of ticks will be passed after each tick
// allowing clients to listen on updates
// Do not use this method
func NewRateConverterWithNotifier(
	httpClient httpClient,
	syncSourceURL string,
	fetchingInterval time.Duration,
	staleRatesThreshold time.Duration,
	updateNotifier chan<- int,
) *RateConverter {
	rc := &RateConverter{
		httpClient:          httpClient,
		done:                make(chan bool),
		updateNotifier:      updateNotifier,
		fetchingInterval:    fetchingInterval,
		staleRatesThreshold: staleRatesThreshold,
		syncSourceURL:       syncSourceURL,
		rates:               atomic.Value{},
		lastUpdated:         atomic.Value{},
		constantRates:       NewConstantRates(),
	}

	// In case host do not want to support currency lookup
	// we just stop here and do nothing
	if rc.fetchingInterval == time.Duration(0) {
		return rc
	}

	rc.Update()                   // Make sure to populate data before to return the converter
	go rc.startPeriodicFetching() // Start periodic ticking

	return rc
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

	defer response.Body.Close()

	bytesJSON, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	updatedRates := &Rates{}
	err = json.Unmarshal(bytesJSON, updatedRates)
	if err != nil {
		return nil, err
	}

	return updatedRates, err
}

// Update updates the internal currencies rates from remote sources
func (rc *RateConverter) Update() error {
	rates, err := rc.fetch()
	if err == nil {
		rc.rates.Store(rates)
		rc.lastUpdated.Store(time.Now())
	} else {
		if rc.CheckStaleRates() {
			rc.ClearRates()
			glog.Errorf("Error updating conversion rates, falling back to constant rates: %v", err)
		} else {
			glog.Errorf("Error updating conversion rates: %v", err)
		}
	}

	return err
}

// startPeriodicFetching starts the periodic fetching at the given interval
// triggers a first fetch when called before the first tick happen in order to initialize currencies rates map
// returns a chan in which the number of data updates everytime a new update was done
func (rc *RateConverter) startPeriodicFetching() {

	ticker := time.NewTicker(rc.fetchingInterval)
	updatesTicksCount := 0

	for {
		select {
		case <-ticker.C:
			// Retries are handled by clients directly.
			rc.Update()
			updatesTicksCount++
			if rc.updateNotifier != nil {
				rc.updateNotifier <- updatesTicksCount
			}
		case <-rc.done:
			if ticker != nil {
				ticker.Stop()
				ticker = nil
			}
			return
		}
	}
}

// StopPeriodicFetching stops the periodic fetching while keeping the latest currencies rates map
func (rc *RateConverter) StopPeriodicFetching() {
	rc.done <- true
	close(rc.done)
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

// ClearRates sets the rates to nil
func (rc *RateConverter) ClearRates() {
	// atomic.Value field rates must be of type *Rates so we cast nil to that type
	rc.rates.Store((*Rates)(nil))
}

// CheckStaleRates checks if loaded third party conversion rates are stale
func (rc *RateConverter) CheckStaleRates() bool {
	if rc.staleRatesThreshold <= 0 {
		return false
	}
	currentTime := time.Now().UTC()
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
	var rates *map[string]map[string]float64
	if rc.Rates() != nil {
		rates = rc.Rates().GetRates()
	}
	return converterInfo{
		source:           rc.syncSourceURL,
		fetchingInterval: rc.fetchingInterval,
		lastUpdated:      rc.LastUpdated(),
		rates:            rates,
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
