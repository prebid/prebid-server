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

func TestFetch_Success(t *testing.T) {

	// Setup:
	calledURLs := []string{}
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			calledURLs = append(calledURLs, req.RequestURI)
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(getMockRates()))
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

	// Execute:
	beforeExecution := time.Now()
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(24)*time.Hour,
		time.Duration(24)*time.Hour,
	)

	// Verify:
	assert.Equal(t, 1, len(calledURLs), "sync URL should have been called %d times but was %d", 1, len(calledURLs))
	assert.NotEqual(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated() should return a time set")
	assert.True(t, currencyConverter.LastUpdated().After(beforeExecution), "LastUpdated() should be after last update")
	rates := currencyConverter.Rates()
	assert.NotNil(t, rates, "Rates() should not return nil")
	assert.Equal(t, expectedRates, rates, "Rates() doesn't return expected rates")
	assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")
}

func TestFetch_Fail404(t *testing.T) {

	// Setup:
	calledURLs := []string{}
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			calledURLs = append(calledURLs, req.RequestURI)
			rw.WriteHeader(http.StatusNotFound)
		}),
	)

	defer mockedHttpServer.Close()

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(24)*time.Hour,
		time.Duration(24)*time.Hour,
	)

	// Verify:
	assert.Equal(t, 1, len(calledURLs), "sync URL should have been called %d times but was %d", 1, len(calledURLs))
	assert.Equal(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated() shouldn't return a time set")
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")
	assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")
}

func TestFetch_FailErrorHttpClient(t *testing.T) {

	// Setup:
	calledURLs := []string{}
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			calledURLs = append(calledURLs, req.RequestURI)
			rw.WriteHeader(http.StatusBadRequest)
		}),
	)

	defer mockedHttpServer.Close()

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(24)*time.Hour,
		time.Duration(24)*time.Hour,
	)

	// Verify:
	assert.Equal(t, 1, len(calledURLs), "sync URL should have been called %d times but was %d", 1, len(calledURLs))
	assert.Equal(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated() shouldn't return a time set")
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")
	assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")
}

func TestFetch_FailBadSyncURL(t *testing.T) {

	// Setup:

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		"justaweirdurl",
		time.Duration(24)*time.Hour,
		time.Duration(24)*time.Hour,
	)

	// Verify:
	assert.Equal(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated() shouldn't return a time set")
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")
	assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")
}

func TestFetch_FailBadJSON(t *testing.T) {

	// Setup:
	calledURLs := []string{}
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			calledURLs = append(calledURLs, req.RequestURI)
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(
				`{
					"dataAsOf":"2018-09-12",
					"conversions":{
						"USD":{
							"GBP":0.77208
						},
						"GBP":{
							"USD":1.2952
						},
						"badJsonHere"
					}
				}`,
			))
		}),
	)

	defer mockedHttpServer.Close()

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(24)*time.Hour,
		time.Duration(24)*time.Hour,
	)

	// Verify:
	assert.Equal(t, 1, len(calledURLs), "sync URL should have been called %d times but was %d", 1, len(calledURLs))
	assert.Equal(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated() shouldn't return a time set")
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")
	assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")
}

func TestFetch_InvalidRemoteResponseContent(t *testing.T) {

	// Setup:
	calledURLs := []string{}
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			calledURLs = append(calledURLs, req.RequestURI)
			rw.WriteHeader(http.StatusOK)
			rw.Write(nil)
		}),
	)

	defer mockedHttpServer.Close()

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(24)*time.Hour,
		time.Duration(24)*time.Hour,
	)

	// Verify:
	assert.Equal(t, 1, len(calledURLs), "sync URL should have been called %d times but was %d", 1, len(calledURLs))
	assert.Equal(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated() shouldn't return a time set")
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
	currencyConverter := currencies.NewRateConverterWithNotifier(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(24)*time.Hour,
		ticks,
	)

	// Verify:
	expectedIntervalDuration := time.Duration(100) * time.Millisecond
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
			intervalDiff := float64(float64(intervalDuration.Nanoseconds()) / float64(expectedIntervalDuration.Nanoseconds()))
			assert.False(t, intervalDiff > float64(errorMargin*100), "Interval between ticks should be: %d but was: %d", expectedIntervalDuration, intervalDuration)
		}

		assert.NotEqual(t, currencyConverter.LastUpdated(), (time.Time{}), "LastUpdated should be set")
		assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
		assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")

		if ticksCount == expectedTicks {
			currencyConverter.StopPeriodicFetching()
			return
		}
	}
}

func TestStop(t *testing.T) {

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
	expectedTicks := 2
	ticks := make(chan int)
	currencyConverter := currencies.NewRateConverterWithNotifier(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(24)*time.Hour,
		ticks,
	)

	// Let the currency converter fetch 5 times before stopping it
	for ticksCount := range ticks {
		if ticksCount == expectedTicks {
			currencyConverter.StopPeriodicFetching()
			break
		}
	}
	lastFetched := time.Now()

	// Verify:
	// Check for the next 1 second that no fetch was triggered
	time.Sleep(1 * time.Second)

	assert.False(t, currencyConverter.LastUpdated().After(lastFetched), "LastUpdated() shouldn't be after `lastFetched` since the periodic fetching is stopped")
}

func TestInitWithZeroDuration(t *testing.T) {

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
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(0),
		time.Duration(24)*time.Hour,
	)

	// Verify:
	// Check for the next 1 second that no fetch was triggered
	time.Sleep(1 * time.Second)

	assert.Equal(t, 0, len(calledURLs), "sync URL shouldn't have been called but was called %d times", 0, len(calledURLs))
	assert.Equal(t, (time.Time{}), currencyConverter.LastUpdated(), "LastUpdated() shouldn't be set")
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")
	assert.NotNil(t, currencyConverter.GetInfo(), "GetInfo() should not return nil")
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
	currencyConverter := currencies.NewRateConverterWithNotifier(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(24)*time.Hour,
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

func TestRates_EmptyRates(t *testing.T) {

	// Setup:
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			rw.Write([]byte(""))
		}),
	)

	// Execute:
	// Will try to fetch directly on method call but will fail
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(24)*time.Hour,
	)
	defer currencyConverter.StopPeriodicFetching()

	// Verify:
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")
}

func TestSelectRatesBasedOnStaleness(t *testing.T) {
	calledURLs := []string{}
	callCnt := 0
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			calledURLs = append(calledURLs, req.RequestURI)
			if callCnt == 0 || callCnt >= 5 {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(getMockRates()))
			} else {
				rw.WriteHeader(http.StatusNotFound)
			}
			callCnt++
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

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(200)*time.Millisecond,
	)

	// Verify:
	// Rates are valid at t=0, then invalid for 500ms before being valid again
	assert.Equal(t, currencyConverter.Rates(), expectedRates, "Rates() should return expected rates")

	time.Sleep(100 * time.Millisecond)
	// Rates have been invalid for ~100ms, rates not stale yet
	assert.Equal(t, currencyConverter.Rates(), expectedRates, "Rates() should return expected rates")

	time.Sleep(200 * time.Millisecond)
	// Rates have been invalid for ~300ms, rates are stale
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")

	time.Sleep(300 * time.Millisecond)
	// Rates have been valid again for ~100ms
	assert.Equal(t, currencyConverter.Rates(), expectedRates, "Rates() should return expected rates")
}

func TestUseConstantRatesUntilFetchIsSuccessful(t *testing.T) {
	callCnt := 0
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			if callCnt >= 5 {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(getMockRates()))
			} else {
				rw.WriteHeader(http.StatusNotFound)
			}
			callCnt++
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

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(1)*time.Second,
	)

	// Verify:
	// Rates are invalid at t=0 and remain invalid until 500ms have elapsed
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")

	time.Sleep(400 * time.Millisecond)
	// Rates have been invalid for ~400ms
	assert.Equal(t, currencyConverter.Rates(), &currencies.ConstantRates{}, "Rates() should return constant rates")

	time.Sleep(200 * time.Millisecond)
	// Rates have been valid for ~100ms
	assert.Equal(t, currencyConverter.Rates(), expectedRates, "Rates() should return expected rates")
}

func TestRatesAreNeverStale(t *testing.T) {
	callCnt := 0
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			if callCnt == 0 {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(getMockRates()))
			} else {
				rw.WriteHeader(http.StatusNotFound)
			}
			callCnt++
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

	// Execute:
	currencyConverter := currencies.NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		time.Duration(100)*time.Millisecond,
		time.Duration(0)*time.Millisecond,
	)

	// Verify:
	// Rates are valid at t=0 and are then invalid at 100ms
	assert.Equal(t, currencyConverter.Rates(), expectedRates, "Rates() should return expected rates")

	time.Sleep(500 * time.Millisecond)
	// Rates have been invalid for ~400ms
	assert.Equal(t, currencyConverter.Rates(), expectedRates, "Rates() should return expected rates")
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
	currencyConverter := currencies.NewRateConverter(
		mockedHttpClient,
		"currency.fake.com",
		time.Duration(10)*time.Millisecond,
		time.Duration(24)*time.Hour,
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
