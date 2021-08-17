package currency

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/prebid/prebid-server/util/task"
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

// FakeTime implements the Time interface
type FakeTime struct {
	time time.Time
}

func (mc *FakeTime) Now() time.Time {
	return mc.time
}

func TestReadWriteRates(t *testing.T) {
	// Setup
	mockServerHandler := func(mockResponse []byte, mockStatus int) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(mockStatus)
			rw.Write([]byte(mockResponse))
		})
	}

	tests := []struct {
		description       string
		giveFakeTime      time.Time
		giveMockUrl       string
		giveMockResponse  []byte
		giveMockStatus    int
		wantUpdateErr     bool
		wantConstantRates bool
		wantLastUpdated   time.Time
		wantConversions   map[string]map[string]float64
	}{
		{
			description:      "Fetching currency rates successfully",
			giveFakeTime:     time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse: getMockRates(),
			giveMockStatus:   200,
			wantLastUpdated:  time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			wantConversions:  map[string]map[string]float64{"USD": {"GBP": 0.77208}, "GBP": {"USD": 1.2952}},
		},
		{
			description:      "Currency rates endpoint returns empty response",
			giveFakeTime:     time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse: []byte("{}"),
			giveMockStatus:   200,
			wantLastUpdated:  time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			wantConversions:  nil,
		},
		{
			description:       "Currency rates endpoint returns nil response",
			giveFakeTime:      time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  nil,
			giveMockStatus:    200,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
		{
			description:       "Currency rates endpoint returns non-2xx status code",
			giveFakeTime:      time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  []byte(`{"message": "Not Found"}`),
			giveMockStatus:    404,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
		{
			description:       "Currency rates endpoint returns invalid json response",
			giveFakeTime:      time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockResponse:  []byte(`{"message": Invalid-JSON-No-Surrounding-Quotes}`),
			giveMockStatus:    200,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
		{
			description:       "Currency rates endpoint url is invalid",
			giveFakeTime:      time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC),
			giveMockUrl:       "invalidurl",
			giveMockResponse:  getMockRates(),
			giveMockStatus:    200,
			wantUpdateErr:     true,
			wantConstantRates: true,
			wantLastUpdated:   time.Time{},
		},
	}

	for _, tt := range tests {
		mockedHttpServer := httptest.NewServer(mockServerHandler(tt.giveMockResponse, tt.giveMockStatus))
		defer mockedHttpServer.Close()

		var url string
		if len(tt.giveMockUrl) > 0 {
			url = tt.giveMockUrl
		} else {
			url = mockedHttpServer.URL
		}
		currencyConverter := NewRateConverter(
			&http.Client{},
			url,
			24*time.Hour,
		)
		currencyConverter.time = &FakeTime{time: tt.giveFakeTime}
		err := currencyConverter.Run()

		if tt.wantUpdateErr {
			assert.NotNil(t, err)
		} else {
			assert.Nil(t, err)
		}

		if tt.wantConstantRates {
			assert.Equal(t, currencyConverter.Rates(), &ConstantRates{}, tt.description)
		} else {
			rates := currencyConverter.Rates().(*Rates)
			assert.Equal(t, tt.wantConversions, (*rates).Conversions, tt.description)
		}

		lastUpdated := currencyConverter.LastUpdated()
		assert.Equal(t, tt.wantLastUpdated, lastUpdated, tt.description)
	}
}

func TestRateStaleness(t *testing.T) {
	callCount := 0
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			callCount++
			if callCount == 2 || callCount >= 5 {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(getMockRates()))
			} else {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{"message": "Not Found"}`))
			}
		}),
	)

	defer mockedHttpServer.Close()

	expectedRates := &Rates{
		Conversions: map[string]map[string]float64{
			"USD": {
				"GBP": 0.77208,
			},
			"GBP": {
				"USD": 1.2952,
			},
		},
	}

	initialFakeTime := time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC)
	fakeTime := &FakeTime{time: initialFakeTime}

	// Execute:
	currencyConverter := NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		30*time.Second, // stale rates threshold
	)
	currencyConverter.time = fakeTime

	// First Update call results in error
	err1 := currencyConverter.Run()
	assert.NotNil(t, err1)

	// Verify constant rates are used and last update ts is not set
	assert.Equal(t, &ConstantRates{}, currencyConverter.Rates(), "Rates should return constant rates")
	assert.Equal(t, time.Time{}, currencyConverter.LastUpdated(), "LastUpdated return is incorrect")

	// Second Update call is successful and yields valid rates
	err2 := currencyConverter.Run()
	assert.Nil(t, err2)

	// Verify rates are valid and last update timestamp is set
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, initialFakeTime, currencyConverter.LastUpdated(), "LastUpdated should be set")

	// Advance time so the rates fall just short of being considered stale
	fakeTime.time = fakeTime.time.Add(29 * time.Second)

	// Third Update call results in error but stale rate threshold has not been exceeded
	err3 := currencyConverter.Run()
	assert.NotNil(t, err3)

	// Verify rates are valid and last update ts has not changed
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, initialFakeTime, currencyConverter.LastUpdated(), "LastUpdated should be set")

	// Advance time just past the threshold so the rates are considered stale
	fakeTime.time = fakeTime.time.Add(2 * time.Second)

	// Fourth Update call results in error and stale rate threshold has been exceeded
	err4 := currencyConverter.Run()
	assert.NotNil(t, err4)

	// Verify constant rates are used and last update ts has not changed
	assert.Equal(t, &ConstantRates{}, currencyConverter.Rates(), "Rates should return constant rates")
	assert.Equal(t, initialFakeTime, currencyConverter.LastUpdated(), "LastUpdated return is incorrect")

	// Fifth Update call is successful and yields valid rates
	err5 := currencyConverter.Run()
	assert.Nil(t, err5)

	// Verify rates are valid and last update ts has changed
	thirtyOneSec := 31 * time.Second
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, (initialFakeTime.Add(thirtyOneSec)), currencyConverter.LastUpdated(), "LastUpdated should be set")
}

func TestRatesAreNeverConsideredStale(t *testing.T) {
	callCount := 0
	mockedHttpServer := httptest.NewServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			callCount++
			if callCount == 1 {
				rw.WriteHeader(http.StatusOK)
				rw.Write([]byte(getMockRates()))
			} else {
				rw.WriteHeader(http.StatusNotFound)
				rw.Write([]byte(`{"message": "Not Found"}`))
			}
		}),
	)

	defer mockedHttpServer.Close()

	expectedRates := &Rates{
		Conversions: map[string]map[string]float64{
			"USD": {
				"GBP": 0.77208,
			},
			"GBP": {
				"USD": 1.2952,
			},
		},
	}

	initialFakeTime := time.Date(2018, time.September, 12, 30, 0, 0, 0, time.UTC)
	fakeTime := &FakeTime{time: initialFakeTime}

	// Execute:
	currencyConverter := NewRateConverter(
		&http.Client{},
		mockedHttpServer.URL,
		0*time.Millisecond, // stale rates threshold
	)
	currencyConverter.time = fakeTime

	// First Update call is successful and yields valid rates
	err1 := currencyConverter.Run()
	assert.Nil(t, err1)

	// Verify rates are valid and last update timestamp is correct
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, fakeTime.time, currencyConverter.LastUpdated(), "LastUpdated should be set")

	// Advance time so the current time is well past the the time the rates were last updated
	fakeTime.time = initialFakeTime.Add(24 * time.Hour)

	// Second Update call results in error but rates from a day ago are still valid
	err2 := currencyConverter.Run()
	assert.NotNil(t, err2)

	// Verify rates are valid and last update ts is correct
	assert.Equal(t, expectedRates, currencyConverter.Rates(), "Conversions.Rates weren't the expected ones")
	assert.Equal(t, initialFakeTime, currencyConverter.LastUpdated(), "LastUpdated should be set")
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
	// Create a rate converter which will be fetching new values every 1 ms
	interval := 1 * time.Millisecond
	currencyConverter := NewRateConverter(
		mockedHttpClient,
		"currency.fake.com",
		24*time.Hour,
	)
	ticker := task.NewTickerTask(interval, currencyConverter)
	ticker.Start()
	defer ticker.Stop()

	var wg sync.WaitGroup
	clientsCount := 10
	wg.Add(clientsCount)
	dones := make([]chan bool, clientsCount)

	for c := 0; c < clientsCount; c++ {
		dones[c] = make(chan bool)
		go func(done chan bool, clientNum int) {
			randomTickInterval := time.Duration(clientNum+1) * time.Millisecond
			clientTicker := time.NewTicker(randomTickInterval)
			for {
				select {
				case <-clientTicker.C:
					if clientNum < 5 {
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

	time.Sleep(100 * time.Millisecond)
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
