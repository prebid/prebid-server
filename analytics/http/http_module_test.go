package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/prebid/openrtb/v19/openrtb2"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/config"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var mockValidAuctionObject = analytics.AuctionObject{
	Status:    http.StatusOK,
	StartTime: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
	RequestWrapper: &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
			App: &openrtb2.App{
				ID: "com.app.test",
			},
			Device: &openrtb2.Device{
				UA: "ua",
			},
			User: &openrtb2.User{
				Consent: "Consent",
			},
		},
	},
}

var mockValidVideoObject = analytics.VideoObject{
	Status:    http.StatusOK,
	StartTime: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
	RequestWrapper: &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
			App: &openrtb2.App{
				ID: "com.app.test",
			},
			Device: &openrtb2.Device{
				UA: "ua",
			},
			User: &openrtb2.User{
				Consent: "Consent",
			},
		},
	},
}

var mockValidAmpObject = analytics.AmpObject{
	Status:    http.StatusOK,
	StartTime: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
	RequestWrapper: &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
			Site: &openrtb2.Site{
				ID: "my-site",
			},
			Device: &openrtb2.Device{
				UA: "ua",
			},
			User: &openrtb2.User{
				Consent: "Consent",
			},
		},
	},
}

var mockValidCookieSyncObject = analytics.CookieSyncObject{
	Status: http.StatusOK,
}

var mockValidNotificationEvent = analytics.NotificationEvent{
	Request: &analytics.EventRequest{
		Bidder: "bidder",
	},
	Account: &config.Account{
		ID: "id",
	},
}

var mockValidSetUIDObject = analytics.SetUIDObject{
	Status: http.StatusOK,
}

type MockedSender struct {
	mock.Mock
}

func (m *MockedSender) Send(payload []byte) error {
	args := m.Called(payload)
	return args.Error(0)
}

func TestConfigParsingError(t *testing.T) {
	testCases := []struct {
		name       string
		config     config.AnalyticsHttp
		shouldFail bool
	}{
		{
			name: "Test with invalid/empty URL",
			config: config.AnalyticsHttp{
				Enabled: true,
				Endpoint: config.AnalyticsHttpEndpoint{
					Url:     "%%2815197306101420000%29",
					Timeout: "1s",
					Gzip:    false,
				},
			},
			shouldFail: true,
		},
		{
			name: "Test with invalid timout",
			config: config.AnalyticsHttp{
				Enabled: true,
				Endpoint: config.AnalyticsHttpEndpoint{
					Url:     "http://localhost:8000/event",
					Timeout: "1x",
					Gzip:    false,
				},
			},
			shouldFail: true,
		},
		{
			name: "Test with invalid filter",
			config: config.AnalyticsHttp{
				Enabled: true,
				Endpoint: config.AnalyticsHttpEndpoint{
					Url:     "http://localhost:8000/event",
					Timeout: "1s",
					Gzip:    false,
				},
				Auction: config.AnalyticsFeature{
					SampleRate: 1,
					Filter:     "foo == bar",
				},
			},
			shouldFail: true,
		},
	}
	clockMock := clock.NewMock()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewModule(&http.Client{}, tc.config, clockMock)
			if tc.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestShouldNotTrack(t *testing.T) {
	cfg := config.AnalyticsHttp{
		Enabled: true,
		Endpoint: config.AnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AnalyticsBuffer{
			EventCount: 1,
			BufferSize: "1Kb",
			Timeout:    "1s",
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newHttpLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	go logger.start()
	assert.Zero(t, logger.eventCount)

	logger.LogAuctionObject(&mockValidAuctionObject)
	logger.LogVideoObject(&mockValidVideoObject)
	logger.LogAmpObject(&mockValidAmpObject)
	logger.LogCookieSyncObject(&mockValidCookieSyncObject)
	logger.LogNotificationEventObject(&mockValidNotificationEvent)
	logger.LogSetUIDObject(&mockValidSetUIDObject)

	clockMock.Add(2 * time.Minute)
	mockedSender.AssertNumberOfCalls(t, "Send", 0)
	assert.Zero(t, logger.eventCount)
}

func TestRaceAllEvents(t *testing.T) {
	sampleAll := config.AnalyticsFeature{
		SampleRate: 1,
	}
	cfg := config.AnalyticsHttp{
		Enabled: true,
		Endpoint: config.AnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AnalyticsBuffer{
			EventCount: 10000,
			BufferSize: "100Mb",
			Timeout:    "5m",
		},
		Auction:      sampleAll,
		AMP:          sampleAll,
		SetUID:       sampleAll,
		Notification: sampleAll,
		Video:        sampleAll,
		CookieSync:   sampleAll,
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newHttpLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	go logger.start()

	logger.LogAuctionObject(&mockValidAuctionObject)
	logger.LogVideoObject(&mockValidVideoObject)
	logger.LogAmpObject(&mockValidAmpObject)
	logger.LogCookieSyncObject(&mockValidCookieSyncObject)
	logger.LogNotificationEventObject(&mockValidNotificationEvent)
	logger.LogSetUIDObject(&mockValidSetUIDObject)

	clockMock.Add(10 * time.Millisecond)

	logger.mux.RLock()
	assert.Equal(t, int64(6), logger.eventCount)
	logger.mux.RUnlock()
}

func TestRaceBufferCount(t *testing.T) {
	cfg := config.AnalyticsHttp{
		Enabled: true,
		Endpoint: config.AnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AnalyticsBuffer{
			EventCount: 2,
			BufferSize: "2MB",
			Timeout:    "15m",
		},
		Auction: config.AnalyticsFeature{
			SampleRate: 1,
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newHttpLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	go logger.start()
	assert.Zero(t, logger.eventCount)

	// Test EventCount Buffer
	logger.LogAuctionObject(&mockValidAuctionObject)

	clockMock.Add(1 * time.Millisecond)

	logger.mux.RLock()
	assert.Equal(t, int64(1), logger.eventCount)
	logger.mux.RUnlock()

	assert.Equal(t, false, logger.isFull())

	// add 1 more
	logger.LogAuctionObject(&mockValidAuctionObject)
	clockMock.Add(1 * time.Millisecond)

	// should trigger send and flash the buffer
	mockedSender.AssertCalled(t, "Send", mock.Anything)

	logger.mux.RLock()
	assert.Equal(t, int64(0), logger.eventCount)
	logger.mux.RUnlock()
}

func TestBufferSize(t *testing.T) {
	cfg := config.AnalyticsHttp{
		Enabled: true,
		Endpoint: config.AnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AnalyticsBuffer{
			EventCount: 10000,
			BufferSize: "10Kb",
			Timeout:    "15m",
		},
		Auction: config.AnalyticsFeature{
			SampleRate: 1,
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newHttpLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	go logger.start()

	for i := 0; i < 50; i++ {
		logger.LogAuctionObject(&mockValidAuctionObject)
	}
	clockMock.Add(10 * time.Millisecond)
	mockedSender.AssertCalled(t, "Send", mock.Anything)
	mockedSender.AssertNumberOfCalls(t, "Send", 1)
}

func TestBufferTime(t *testing.T) {
	cfg := config.AnalyticsHttp{
		Enabled: true,
		Endpoint: config.AnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AnalyticsBuffer{
			EventCount: 10000,
			BufferSize: "100Mb",
			Timeout:    "5m",
		},
		Auction: config.AnalyticsFeature{
			SampleRate: 1,
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newHttpLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	go logger.start()

	for i := 0; i < 5; i++ {
		logger.LogAuctionObject(&mockValidAuctionObject)
	}
	clockMock.Add(10 * time.Minute)
	mockedSender.AssertCalled(t, "Send", mock.Anything)
	mockedSender.AssertNumberOfCalls(t, "Send", 1)
}

func TestRaceEnd2End(t *testing.T) {
	var mu sync.Mutex

	requestBodyAsString := ""

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for reponse
		requestBody, err := io.ReadAll(r.Body)
		mu.Lock()
		requestBodyAsString = string(requestBody)
		mu.Unlock()
		if err != nil {
			http.Error(w, "Error reading request body", 500)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))

	sampleAll := config.AnalyticsFeature{
		SampleRate: 1,
	}
	cfg := config.AnalyticsHttp{
		Enabled: true,
		Endpoint: config.AnalyticsHttpEndpoint{
			Url:     server.URL,
			Timeout: "5s",
		},
		Buffers: config.AnalyticsBuffer{
			EventCount: 2,
			BufferSize: "100Mb",
			Timeout:    "5m",
		},
		Auction:      sampleAll,
		AMP:          sampleAll,
		SetUID:       sampleAll,
		Notification: sampleAll,
		Video:        sampleAll,
		CookieSync:   sampleAll,
	}

	clockMock := clock.NewMock()
	clockMock.Set(time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC))

	logger, err := NewModule(&http.Client{}, cfg, clockMock)
	assert.NoError(t, err)

	logger.LogSetUIDObject(&mockValidSetUIDObject)
	logger.LogSetUIDObject(&mockValidSetUIDObject)

	time.Sleep(250 * time.Millisecond)

	expected := "[{\"type\":\"setuid\",\"createdAt\":\"2023-02-01T00:00:00Z\",\"status\":200},{\"type\":\"setuid\",\"createdAt\":\"2023-02-01T00:00:00Z\",\"status\":200}]"

	mu.Lock()
	actual := requestBodyAsString
	mu.Unlock()

	assert.Equal(t, expected, actual)
}
