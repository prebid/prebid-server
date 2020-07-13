package currencies_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prebid/prebid-server/currencies"
	"github.com/stretchr/testify/assert"
)

func getMockRates() []byte {
	return []byte(`{
		"dataAsOf":"2018-09-12",
		"conversions":{
			"USD":{
				"GBP":0.77208
			},
			"GBP":{
				"USD":1.2952
			}
		}
	}`)
}

func formatTestErrorMsg(desc string, message string, args ...interface{}) string {
	message = fmt.Sprintf(message, args)
	return fmt.Sprintf("Test Case: %s \n Message: %s", desc, message)
}

func TestReadWriteRates(t *testing.T) {
	// Setup
	mockServerHandler := func(mockResponse []byte, mockCode int) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(mockCode)
			rw.Write([]byte(mockResponse))
		})
	}

	tests := []struct {
		description       string
		giveFrozenTime    time.Time
		giveMockUrl       string
		giveMockResponse  []byte
		giveMockCode      int
		wantUpdateErr     bool
		wantConstantRates bool
		wantLastUpdated   time.Time
		wantDataAsOf      time.Time
		wantConversions   map[string]map[string]float64
	}{
		{
			description:       "Fetching currency rates successfully",
			giveFrozenTime:    time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  getMockRates(),
			giveMockCode:      200,
			wantUpdateErr:     false,
			wantConstantRates: false,
			wantLastUpdated:   time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			wantDataAsOf:      time.Date(2018, time.September, 12, 0, 0, 0, 0, time.UTC),
			wantConversions:   map[string]map[string]float64{"USD": {"GBP": 0.77208}, "GBP": {"USD": 1.2952}},
		},
		{
			description:       "Currency rates endpoint returns empty response",
			giveFrozenTime:    time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  []byte("{}"),
			giveMockCode:      200,
			wantUpdateErr:     false,
			wantConstantRates: false,
			wantLastUpdated:   time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			wantDataAsOf:      time.Time{},
			wantConversions:   map[string]map[string]float64(nil),
		},
		{
			description:       "Currency rates endpoint returns nil response",
			giveFrozenTime:    time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  nil,
			giveMockCode:      200,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
		{
			description:       "Currency rates endpoint returns non-2xx status code",
			giveFrozenTime:    time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  []byte(`{"message": "Not Found"}`),
			giveMockCode:      404,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
		{
			description:       "Currency rates endpoint returns invalid json response",
			giveFrozenTime:    time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  []byte(`{"message": Invalid-JSON-No-Surrounding-Quotes}`),
			giveMockCode:      200,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
		{
			description:       "Currency rates endpoint url is invalid",
			giveFrozenTime:    time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockUrl:       "invalidurl",
			giveMockResponse:  getMockRates(),
			giveMockCode:      200,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
	}

	for _, tt := range tests {
		mockedHttpServer := httptest.NewServer(mockServerHandler(tt.giveMockResponse, tt.giveMockCode))
		defer mockedHttpServer.Close()

		var url string
		if len(tt.giveMockUrl) > 0 {
			url = tt.giveMockUrl
		} else {
			url = mockedHttpServer.URL
		}
		currencyConverter := currencies.NewRateConverter(
			&http.Client{},
			url,
			time.Duration(24)*time.Hour,
			time.Duration(24)*time.Hour,
			currencies.NewMockClockAt(tt.giveFrozenTime),
		)
		err := currencyConverter.Run()

		if tt.wantUpdateErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		if tt.wantConstantRates {
			assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, tt.description)
		} else {
			rates := currencyConverter.Rates().(*currencies.Rates)
			assert.Equal(t, tt.wantConversions, (*rates).Conversions, tt.description)
			assert.Equal(t, tt.wantDataAsOf, (*rates).DataAsOf, tt.description)
		}

		lastUpdated := currencyConverter.LastUpdated()
		assert.Equal(t, tt.wantLastUpdated, lastUpdated, tt.description)
	}
}

func TestRateStaleness(t *testing.T) {
	callCnt := 0
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			callCnt++
			if callCnt == 2 || callCnt >= 5 {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(getMockRates()))
			} else {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{"message": "Not Found"}`))
			}
		}),
	)

	defer mockedHttpServer.Close()

	expectedRates := &currencies.Rates{
		DataAsOf: time.Date(2018, time.September, 12, 0, 0, 0, 0, time.UTC),
		Conversions: map[string]map[string]float64{
			"USD": {
				"GBP": 0.77208,
			},
			"GBP": {
				"USD": 1.2952,
			},
		},
	}

	frozenTime := time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC)
	mockClock := currencies.NewMockClockAt(frozenTime)

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(30)*time.Second, // stale rates threshold
		mockClock,
	)

	// First Update call results in error
	err1 := currencyConverter.Run()
	assert.NotNil(t, err1)

	// Verify constant rates are used and last update ts is not set
	assert.Equal(t, &currencies.ConstantRates{}, currencyConverter.Rates(), "Rates should return constant rates")
	assert.Equal(t, time.Time{}, currencyConverter.LastUpdated(), "LastUpdated return is incorrect")

	// Second Update call is successful and yields valid rates
	err2 := currencyConverter.Run()
	assert.Nil(t, err2)

	// Verify rates are valid and last update timestamp is set
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, frozenTime, currencyConverter.LastUpdated(), "LastUpdated should be set")

	// Advance time so the rates fall just short of being considered stale
	twentyNineSec := time.Duration(29) * time.Second
	mockClock.Advance(twentyNineSec)

	// Third Update call results in error
	err3 := currencyConverter.Run()
	assert.NotNil(t, err3)

	// Verify rates are valid and last update ts is set
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, frozenTime, currencyConverter.LastUpdated(), "LastUpdated should be set")

	// Advance time just past the threshold so the rates are considered stale
	twoSec := time.Duration(2) * time.Second
	mockClock.Advance(twoSec)

	// Fourth Update call results in error
	err4 := currencyConverter.Run()
	assert.NotNil(t, err4)

	// Verify constant rates are used and last update ts is set
	assert.Equal(t, &currencies.ConstantRates{}, currencyConverter.Rates(), "Rates should return constant rates")
	assert.Equal(t, frozenTime, currencyConverter.LastUpdated(), "LastUpdated return is incorrect")

	// Fifth Update call is successful and yields valid rates
	err5 := currencyConverter.Run()
	assert.Nil(t, err5)

	// Verify rates are valid and last update ts has changed
	thirtyOneSec := time.Duration(31) * time.Second
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, (frozenTime.Add(thirtyOneSec)), currencyConverter.LastUpdated(), "LastUpdated should be set")
}

func TestRatesAreNeverStale(t *testing.T) {
	callCnt := 0
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			callCnt++
			if callCnt == 1 {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(getMockRates()))
			} else {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{"message": "Not Found"}`))
			}
		}),
	)

	defer mockedHttpServer.Close()

	expectedRates := &currencies.Rates{
		DataAsOf: time.Date(2018, time.September, 12, 0, 0, 0, 0, time.UTC),
		Conversions: map[string]map[string]float64{
			"USD": {
				"GBP": 0.77208,
			},
			"GBP": {
				"USD": 1.2952,
			},
		},
	}

	frozenTime := time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC)
	mockClock := currencies.NewMockClockAt(frozenTime)

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(0)*time.Millisecond, // stale rates threshold
		mockClock,
	)

	// First Update call is successful and yields valid rates
	err1 := currencyConverter.Run()
	assert.Nil(t, err1)

	// Verify rates are valid and last update timestamp is correct
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, frozenTime, currencyConverter.LastUpdated(), "LastUpdated should be set")

	// Advance time so the rates fall just short of being considered stale
	twentyFourHours := time.Duration(24) * time.Hour
	mockClock.Advance(twentyFourHours)

	// Second Update call results in error but rates from a day ago are still valid
	err2 := currencyConverter.Run()
	assert.NotNil(t, err2)

	// Verify rates are valid and last update ts is correct
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, frozenTime, currencyConverter.LastUpdated(), "LastUpdated should be set")
}
