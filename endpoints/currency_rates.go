package endpoints

import (
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/prebid/prebid-server/v3/currency"
	"github.com/prebid/prebid-server/v3/util/jsonutil"
)

// currencyRatesInfo holds currency rates information.
type currencyRatesInfo struct {
	Active           bool                           `json:"active"`
	Source           *string                        `json:"source,omitempty"`
	FetchingInterval *time.Duration                 `json:"fetchingIntervalNs,omitempty"`
	LastUpdated      *time.Time                     `json:"lastUpdated,omitempty"`
	Rates            *map[string]map[string]float64 `json:"rates,omitempty"`
	AdditionalInfo   interface{}                    `json:"additionalInfo,omitempty"`
}

type rateConverter interface {
	GetInfo() currency.ConverterInfo
}

// newCurrencyRatesInfo creates a new CurrencyRatesInfo instance.
func newCurrencyRatesInfo(rateConverter rateConverter, fetchingInterval time.Duration) currencyRatesInfo {

	currencyRatesInfo := currencyRatesInfo{
		Active: false,
	}

	if rateConverter == nil {
		return currencyRatesInfo
	}

	currencyRatesInfo.Active = true

	infos := rateConverter.GetInfo()
	if infos == nil {
		return currencyRatesInfo
	}

	source := infos.Source()
	currencyRatesInfo.Source = &source

	currencyRatesInfo.FetchingInterval = &fetchingInterval

	lastUpdated := infos.LastUpdated()
	currencyRatesInfo.LastUpdated = &lastUpdated

	currencyRatesInfo.Rates = infos.Rates()
	currencyRatesInfo.AdditionalInfo = infos.AdditionalInfo()

	return currencyRatesInfo
}

// NewCurrencyRatesEndpoint returns current currency rates applied by the PBS server.
func NewCurrencyRatesEndpoint(rateConverter rateConverter, fetchingInterval time.Duration) http.HandlerFunc {
	currencyRateInfo := newCurrencyRatesInfo(rateConverter, fetchingInterval)

	return func(w http.ResponseWriter, _ *http.Request) {
		jsonOutput, err := jsonutil.Marshal(currencyRateInfo)
		if err != nil {
			glog.Errorf("/currency/rates Critical error when trying to marshal currencyRateInfo: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonOutput)
	}
}
