package endpoints

import (
	"math/cmplx"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/v3/currency"
	"github.com/stretchr/testify/assert"
)

func TestCurrencyRatesEndpoint(t *testing.T) {
	// Setup:
	var testCases = []struct {
		inputConverter        rateConverter
		inputFetchingInterval time.Duration
		expectedBody          string
		expectedCode          int
		description           string
	}{
		{
			nil,
			time.Duration(0),
			`{"active": false}`,
			http.StatusOK,
			"case 1 - rate converter is nil",
		},
		{
			newRateConverterMock(
				"https://sync.test.com",
				time.Date(2019, 3, 2, 12, 54, 56, 651387237, time.UTC),
				newConversionMock(&map[string]map[string]float64{
					"USD": {
						"USD": 1.0,
					},
				}),
			),
			5 * time.Minute,
			`{
				"active": true,
				"source": "https://sync.test.com",
				"fetchingIntervalNs": 300000000000,
				"lastUpdated": "2019-03-02T12:54:56.651387237Z",
				"rates": {
					"USD": {
						"USD": 1
					}
				}
			 }`,
			http.StatusOK,
			"case 2 - rate converter is set and has some rates",
		},
		{
			newRateConverterMock(
				"",
				time.Time{},
				nil,
			),
			time.Duration(0),
			`{
				"active": true,
				"source": "",
				"fetchingIntervalNs": 0,
				"lastUpdated": "0001-01-01T00:00:00Z"
			 }`,
			http.StatusOK,
			"case 3 - rate converter is set and doesn't have any rates set",
		},
		{
			newRateConverterMockWithInfo(
				newUnmarshableConverterInfoMock(),
			),
			time.Duration(0),
			"",
			http.StatusInternalServerError,
			"case 4 - invalid rates input for marshaling",
		},
		{
			newRateConverterMockWithNilInfo(),
			time.Duration(0),
			`{
				"active": true
			 }`,
			http.StatusOK,
			"case 5 - rate converter is set but returns nil Infos",
		},
	}

	for _, tc := range testCases {

		handler := NewCurrencyRatesEndpoint(tc.inputConverter, tc.inputFetchingInterval)
		w := httptest.NewRecorder()

		// Execute:
		handler(w, nil)

		// Verify:
		assert.Equal(t, tc.expectedCode, w.Code, tc.description)
		if tc.expectedBody != "" {
			assert.JSONEq(t, tc.expectedBody, w.Body.String(), tc.description)
		} else {
			assert.Equal(t, tc.expectedBody, w.Body.String(), tc.description)
		}
	}
}

type conversionMock struct {
	rates *map[string]map[string]float64
}

func (m conversionMock) GetRates() *map[string]map[string]float64 {
	return m.rates
}

func newConversionMock(rates *map[string]map[string]float64) *conversionMock {
	return &conversionMock{
		rates: rates,
	}
}

type converterInfoMock struct {
	source         string
	lastUpdated    time.Time
	rates          *map[string]map[string]float64
	additionalInfo interface{}
}

func (m converterInfoMock) Source() string {
	return m.source
}

func (m converterInfoMock) LastUpdated() time.Time {
	return m.lastUpdated
}

func (m converterInfoMock) Rates() *map[string]map[string]float64 {
	return m.rates
}

func (m converterInfoMock) AdditionalInfo() interface{} {
	return m.additionalInfo
}

type unmarshableConverterInfoMock struct{}

func (m unmarshableConverterInfoMock) Source() string {
	return ""
}

func (m unmarshableConverterInfoMock) LastUpdated() time.Time {
	return time.Time{}
}

func (m unmarshableConverterInfoMock) Rates() *map[string]map[string]float64 {
	return nil
}

func (m unmarshableConverterInfoMock) AdditionalInfo() interface{} {
	cmplx.Sqrt(-5 + 12i)
	return cmplx.Sqrt(-5 + 12i)
}

func newUnmarshableConverterInfoMock() unmarshableConverterInfoMock {
	return unmarshableConverterInfoMock{}
}

type rateConverterMock struct {
	syncSourceURL       string
	rates               *conversionMock
	lastUpdated         time.Time
	rateConverterInfos  currency.ConverterInfo
	shouldReturnNilInfo bool
}

func (m rateConverterMock) GetInfo() currency.ConverterInfo {

	if m.shouldReturnNilInfo {
		return nil
	}

	if m.rateConverterInfos != nil {
		return m.rateConverterInfos
	}

	var rates *map[string]map[string]float64
	if m.rates == nil {
		rates = nil
	} else {
		rates = m.rates.GetRates()
	}
	return converterInfoMock{
		source:      m.syncSourceURL,
		lastUpdated: m.lastUpdated,
		rates:       rates,
	}
}

func newRateConverterMock(
	syncSourceURL string,
	lastUpdated time.Time,
	rates *conversionMock) rateConverterMock {
	return rateConverterMock{
		syncSourceURL: syncSourceURL,
		rates:         rates,
		lastUpdated:   lastUpdated,
	}
}

func newRateConverterMockWithInfo(rateConverterInfos currency.ConverterInfo) rateConverterMock {
	return rateConverterMock{
		rateConverterInfos: rateConverterInfos,
	}
}

func newRateConverterMockWithNilInfo() rateConverterMock {
	return rateConverterMock{
		shouldReturnNilInfo: true,
	}
}
