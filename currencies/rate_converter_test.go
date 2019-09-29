package currencies_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
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

func TestNewRateConverter(t *testing.T) {
	// Setup:
	mockServerHandler := func(mockResponse []byte, code int) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(code)
			rw.Write(mockResponse)
		})
	}

	testCases := []struct {
		mockResponse     []byte
		mockResponseCode int
		updateInterval   int // In seconds
		fetchURL         string
		expectedTicks    int
		description      string
	}{
		{
			mockResponse:     getMockRates(),
			mockResponseCode: 200,
			updateInterval:   1,
			expectedTicks:    1,
			description:      "Fetching currency rates successfully",
		},
		{
			mockResponse:     []byte(""),
			mockResponseCode: 200,
			updateInterval:   100,
			description:      "Currency rates endpoint returns empty response",
		},
		{
			mockResponse:     []byte("`invalid-json:"),
			mockResponseCode: 200,
			updateInterval:   100,
			description:      "Currency rates endpoint returns invalid json",
		},
		{
			mockResponse:     nil,
			mockResponseCode: 200,
			updateInterval:   100,
			description:      "Currency rates endpoint returns nil response",
		},
		{
			mockResponseCode: 404,
			updateInterval:   4,
			description:      "Currency rates endpoint returns 404",
		},
		{
			mockResponseCode: 400,
			updateInterval:   1,
			description:      "Currency rates endpoint returns 400",
		},
		{
			mockResponse:     getMockRates(),
			mockResponseCode: 200,
			description:      "Fetch interval set to 0",
		},
		{
			fetchURL:    "invalid-url",
			description: "Invalid currency rates endpoint",
		},
	}

	for _, test := range testCases {
		mockedHttpServer := httptest.NewServer(mockServerHandler(test.mockResponse, test.mockResponseCode))

		if test.fetchURL == "" {
			test.fetchURL = mockedHttpServer.URL
		}
		// Execute:
		beforeExecution := time.Now()
		currencyConverter := currencies.NewRateConverter(
			&http.Client{},
			test.fetchURL,
			time.Duration(test.updateInterval)*time.Second,
		)

		// Verify:
		if test.expectedTicks == 0 {
			// Check for the next 200 milliseconds that no fetch was triggered
			time.Sleep(200 * time.Millisecond)

			assert.Equal(t, (time.Time{}), currencyConverter.LastUpdated(), formatTestErrorMsg(test.description, "LastUpdated() shouldn't be set"))
			assert.Equal(t, &currencies.ConstantRates{}, currencyConverter.Rates(), formatTestErrorMsg(test.description, "Conversions.Rates weren't the expected ones"))
			continue
		}

		lastUpdated := currencyConverter.LastUpdated()
		assert.NotEqual(t, (time.Time{}), lastUpdated, formatTestErrorMsg(test.description, "LastUpdated() should return a time set"))
		assert.True(t, lastUpdated.After(beforeExecution), formatTestErrorMsg(test.description, "LastUpdated() should be after last update"))

		rates := &currencies.Rates{}
		err := json.Unmarshal(test.mockResponse, rates)
		assert.NoError(t, err, formatTestErrorMsg(test.description, "JSON unmarshalling of conversions failed"))

		assert.Equal(t, rates, currencyConverter.Rates(), formatTestErrorMsg(test.description, "Rates() doesn't return expected rates"))

		// Check for the next 1 second that no fetch was triggered
		time.Sleep(500 * time.Millisecond)
		assert.Equal(t, lastUpdated, currencyConverter.LastUpdated(), formatTestErrorMsg(test.description, "Currency rates were fetched before the requested duration was passed"))
	}
}

func TestNewRateConverterWithNotifier(t *testing.T) {
	// Setup:
	rates := &currencies.Rates{}
	err := json.Unmarshal(getMockRates(), rates)
	assert.NoError(t, err, "JSON unmarshalling of conversions failed")

	mockServerHandler := func(mockResponse []byte, code int) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.WriteHeader(code)
			rw.Write(mockResponse)
		})
	}

	testCases := []struct {
		mockResponse     []byte
		mockResponseCode int
		updateInterval   int
		fetchURL         string
		expectedTicks    int
		description      string
	}{
		{
			mockResponse:     getMockRates(),
			mockResponseCode: 200,
			updateInterval:   100,
			expectedTicks:    5,
			description:      "Fetching currency rates successfully",
		},
		{
			mockResponse:     []byte(""),
			mockResponseCode: 200,
			updateInterval:   100,
			description:      "Currency rates endpoint returns empty response",
		},
		{
			mockResponse:     []byte("`invalid-json:"),
			mockResponseCode: 200,
			updateInterval:   100,
			description:      "Currency rates endpoint returns invalid json",
		},
		{
			mockResponseCode: 503,
			updateInterval:   1,
			description:      "Unable to reach currency rates endpoint",
		},
		{
			mockResponse:     getMockRates(),
			mockResponseCode: 200,
			description:      "Fetch interval set to 0",
		},
		{
			fetchURL:    "invalid-url",
			description: "Invalid currency rates endpoint",
		},
	}

	for _, test := range testCases {
		ticksTimes := []time.Time{}
		ticks := make(chan int)
		mockedHttpServer := httptest.NewServer(mockServerHandler(test.mockResponse, test.mockResponseCode))
		if test.fetchURL == "" {
			test.fetchURL = mockedHttpServer.URL
		}
		updateInterval := time.Duration(test.updateInterval) * time.Millisecond

		// Execute:
		currencyConverter := currencies.NewRateConverterWithNotifier(
			&http.Client{},
			test.fetchURL,
			updateInterval,
			ticks,
		)

		// Verify:

		// If test.expectedTicks == 0 meaning we expect back constantRates
		if test.expectedTicks == 0 {
			assert.NotNil(t, currencyConverter.Rates(), formatTestErrorMsg(test.description, "Rates shouldn't be nil"))
			assert.Equal(t, &currencies.ConstantRates{}, currencyConverter.Rates(), formatTestErrorMsg(test.description, "Conversions.Rates weren't the expected ones"))
			continue
		}

		errorMargin := 0.1 // 10% error margin

		// At each ticks, do couple checks
		for ticksCount := range ticks {
			ticksTimes = append(ticksTimes, time.Now())
			if len(ticksTimes) > 1 {
				intervalDuration := ticksTimes[len(ticksTimes)-1].Truncate(time.Millisecond).Sub(ticksTimes[len(ticksTimes)-2].Truncate(time.Millisecond))
				intervalDiff := float64(float64(intervalDuration.Nanoseconds()) / float64(updateInterval.Nanoseconds()))
				assert.False(t, intervalDiff > float64(errorMargin*100), formatTestErrorMsg(test.description, "Interval between ticks should be: %d but was: %d", updateInterval, intervalDuration))
			}

			assert.NotNil(t, currencyConverter.Rates(), formatTestErrorMsg(test.description, "Rates shouldn't be nil"))
			assert.NotEqual(t, currencyConverter.LastUpdated(), (time.Time{}), formatTestErrorMsg(test.description, "LastUpdated should be set"))
			assert.Equal(t, rates, currencyConverter.Rates(), formatTestErrorMsg(test.description, "Conversions.Rates weren't the expected ones"))

			if ticksCount == test.expectedTicks {
				currencyConverter.StopPeriodicFetching()
				break
			}
		}
		lastFetched := time.Now()

		// Check for the next 1 second that no fetch was triggered
		time.Sleep(1 * time.Second)
		assert.False(t, currencyConverter.LastUpdated().After(lastFetched), formatTestErrorMsg(test.description, "LastUpdated() shouldn't be after `lastFetched` since the periodic fetching is stopped"))
	}
}

func TestRates(t *testing.T) {

	// Setup:
	testCases := []struct {
		from         string
		to           string
		expectedRate float64
		hasError     bool
	}{
		{from: "USD", to: "GBP", expectedRate: 0.77208, hasError: false},
		{from: "GBP", to: "USD", expectedRate: 1.2952, hasError: false},
		{from: "GBP", to: "EUR", expectedRate: 0, hasError: true},
		{from: "CNY", to: "EUR", expectedRate: 0, hasError: true},
		{from: "", to: "EUR", expectedRate: 0, hasError: true},
		{from: "CNY", to: "", expectedRate: 0, hasError: true},
		{from: "", to: "", expectedRate: 0, hasError: true},
	}

	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(getMockRates()))
		}),
	)

	// Execute:
	ticks := make(chan int)
	currencyConverter := currencies.NewRateConverterWithNotifier(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		ticks,
	)
	rates := currencyConverter.Rates()

	// Let the currency converter ticks 1 time before to stop it
	select {
	case <-ticks:
		currencyConverter.StopPeriodicFetching()
	}

	// Verify:
	assert.NotNil(t, rates, "rates shouldn't be nil")
	for _, tc := range testCases {
		rate, err := rates.GetRate(tc.from, tc.to)

		if tc.hasError {
			assert.NotNil(t, err, "err shouldn't be nil")
			assert.Equal(t, float64(0), rate, "rate should be 0")
		} else {
			assert.Nil(t, err, "err should be nil")
			assert.Equal(t, tc.expectedRate, rate, "rate doesn't match the expected one")
		}
	}
}

func TestRace(t *testing.T) {

	// This test is checking that no race conditions appear in rate converter.
	// It simulate multiple clients (in different goroutines) asking for updates
	// and rates while the rate converter is also updating periodically.

	// Setup:
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(getMockRates()))
		}),
	)

	// Execute:

	// Create a rate converter which will be fetching new values every 10 ms
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(10)*time.Millisecond,
	)
	defer currencyConverter.StopPeriodicFetching()

	// Create 50 clients asking for updates and rates conversion at random intervals
	// from 1ms to 50ms for 10 seconds
	var wg sync.WaitGroup
	clientsCount := 50
	wg.Add(clientsCount)
	dones := make([]chan bool, clientsCount)

	for c := 0; c < clientsCount; c++ {
		dones[c] = make(chan bool)
		go func(done chan bool, clientNum int) {
			randomTickInterval := time.Duration(clientNum+1) * time.Millisecond
			clientTicker := time.NewTicker(randomTickInterval)
			for {
				select {
				case tickTime := <-clientTicker.C:
					// Either ask for an Update() or for GetRate()
					// based on the tick ms
					tickMs := tickTime.UnixNano() / int64(time.Millisecond)
					if tickMs%2 == 0 {
						err := currencyConverter.Update()
						assert.Nil(t, err)
					} else {
						rate, err := currencyConverter.Rates().GetRate("USD", "GBP")
						assert.Nil(t, err)
						assert.Equal(t, float64(0.77208), rate)
					}
				case <-done:
					wg.Done()
					return
				}
			}
		}(dones[c], c)
	}

	time.Sleep(10 * time.Second)
	// Sending stop signals to all clients
	for i := range dones {
		dones[i] <- true
	}
	wg.Wait()
}
