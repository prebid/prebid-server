package currencies_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prebid/prebid-server/currencies"
	"github.com/prebid/prebid-server/util/timeutil"
	"github.com/stretchr/testify/assert"
)

type MockRunner struct {
	updateNotifier chan<- int
	runCount       int
}

func (mcc *MockRunner) Run() error { return nil }
func (mcc *MockRunner) Notify()    { mcc.runCount++; mcc.updateNotifier <- mcc.runCount }

func TestStop(t *testing.T) {
	// Setup:
	ticks := make(chan int)
	mockRunner := &MockRunner{updateNotifier: ticks}
	interval := time.Duration(1) * time.Millisecond
	ticker := currencies.NewTickerTask(interval, mockRunner)

	// Execute:
	expectedTicks := 2
	ticker.Start()

	// Let the currency converter fetch 2 times before stopping it
	for ticksCount := range ticks {
		if ticksCount >= expectedTicks {
			ticker.Stop()
			break
		}
	}

	// Verify:
	// Give the ticker enough time to tick again if it didn't stop
	time.Sleep(2 * time.Millisecond)

	// Verify no additional data was received because the ticker was stopped
	var moreTicks bool
	select {
	case <-ticks:
		moreTicks = true
	default:
		moreTicks = false
	}
	assert.Equal(t, moreTicks, false)
}

func TestIntervalNotSet(t *testing.T) {

	// Setup:
	calledURLs := []string{}
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			calledURLs = append(calledURLs, req.RequestURI)
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(getMockRates()))
		}),
	)

	// Execute:
	interval := time.Duration(0)
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		interval,
		time.Duration(24)*time.Hour,
	)
	ticker := currencies.NewTickerTask(interval, currencyConverter)
	ticker.Start()

	// Verify:
	// Check for the next 1 second that no fetch was triggered
	time.Sleep(1 * time.Second)

	assert.Equal(t, 0, len(calledURLs), "sync URL shouldn't have been called but was called %d times", 0, len(calledURLs))
	assert.Equal(t, (time.Time{}), currencyConverter.LastUpdated(), "LastUpdated() shouldn't be set")
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")
	assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")
}

func TestInit(t *testing.T) {

	// Setup:
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(getMockRates()))
		}),
	)

	// Execute:
	expectedTicks := 5
	ticksTimes := []time.Time{}
	ticks := make(chan int)
	interval := time.Duration(100) * time.Millisecond
	currencyConverter := currencies.NewRateConverterWithNotifier(
		&http.Client{},
		mockedHttpServer.URL,
		interval,
		time.Duration(24)*time.Hour,
		timeutil.NewRealClock(),
		ticks,
	)
	ticker := currencies.NewTickerTask(interval, currencyConverter)
	ticker.Start()

	// Verify:
	errorMargin := 0.1 // 10% error margin
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

	// At each ticks, do couple checks
	for ticksCount := range ticks {
		ticksTimes = append(ticksTimes, time.Now())
		if len(ticksTimes) > 1 {
			intervalDuration := ticksTimes[len(ticksTimes)-1].Truncate(time.Millisecond).Sub(ticksTimes[len(ticksTimes)-2].Truncate(time.Millisecond))
			intervalDiff := float64(float64(intervalDuration.Nanoseconds()) / float64(interval.Nanoseconds()))
			assert.False(t, intervalDiff > float64(errorMargin*100), "Interval between ticks should be: %d but was: %d", interval, intervalDuration)
		}

		assert.NotEqual(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated should be set")
		assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
		assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")

		if ticksCount == expectedTicks {
			ticker.Stop()
			return
		}
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
		{from: "USD", to: "USD", expectedRate: 1, hasError: false},
	}

	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(getMockRates()))
		}),
	)

	// Execute:
	ticks := make(chan int)
	interval := time.Duration(100) * time.Millisecond
	currencyConverter := currencies.NewRateConverterWithNotifier(
		&http.Client{},
		mockedHttpServer.URL,
		interval,
		time.Duration(24)*time.Hour,
		timeutil.NewRealClock(),
		ticks,
	)
	ticker := currencies.NewTickerTask(interval, currencyConverter)
	ticker.Start()

	rates := currencyConverter.Rates()

	// Let the currency converter ticks 1 time before to stop it
	select {
	case <-ticks:
		ticker.Stop()
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
	// Using an HTTP client mock preventing any http client overload while using
	// very small update intervals (less than 50ms) in this test.
	// See #722
	mockedHttpClient := &mockHttpClient{
		responseBody: `{
			"dataAsOf":"2018-09-12",
			"conversions":{
				"USD":{
					"GBP":0.77208
				},
				"GBP":{
					"USD":1.2952
				}
			}
		}`,
	}

	// Execute:

	// Create a rate converter which will be fetching new values every 10 ms
	interval := time.Duration(10) * time.Millisecond
	currencyConverter := currencies.NewRateConverter(
		mockedHttpClient,
		"currency.fake.com",
		interval,
		time.Duration(24)*time.Hour,
	)
	ticker := currencies.NewTickerTask(interval, currencyConverter)
	ticker.Start()
	defer ticker.Stop()

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
						err := currencyConverter.Run()
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

// mockHttpClient is a simple http client mock returning a constant response body
type mockHttpClient struct {
	responseBody string
}

func (m *mockHttpClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Body:       ioutil.NopCloser(strings.NewReader(m.responseBody)),
	}, nil
}
