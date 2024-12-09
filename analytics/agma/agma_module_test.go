package agma

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v3/analytics"
	"github.com/prebid/prebid-server/v3/config"
	"github.com/prebid/prebid-server/v3/openrtb_ext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var agmaConsent = "CP6-v9RP6-v9RNlAAAENCZCAAICAAAAAAAAAIxQAQIxAAAAA.II7Nd_X__bX9n-_7_6ft0eY1f9_r37uQzDhfNs-8F3L_W_LwX32E7NF36tq4KmR4ku1bBIQNtHMnUDUmxaolVrzHsak2cpyNKJ_JkknsZe2dYGF9Pn9lD-YKZ7_5_9_f52T_9_9_-39z3_9f___dv_-__-vjf_599n_v9fV_78_Kf9______-____________8A"

var mockValidAuctionObject = analytics.AuctionObject{
	Status:    http.StatusOK,
	StartTime: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
	RequestWrapper: &openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
			Site: &openrtb2.Site{
				ID: "track-me-site",
				Publisher: &openrtb2.Publisher{
					ID: "track-me",
				},
			},
			Device: &openrtb2.Device{
				UA: "ua",
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
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
				ID: "track-me-app",
				Publisher: &openrtb2.Publisher{
					ID: "track-me",
				},
			},
			Device: &openrtb2.Device{
				UA: "ua",
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
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
				ID: "track-me-site",
				Publisher: &openrtb2.Publisher{
					ID: "track-me",
				},
			},
			Device: &openrtb2.Device{
				UA: "ua",
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
			},
		},
	},
}

var mockValidAccounts = []config.AgmaAnalyticsAccount{
	{
		PublisherId: "track-me",
		Code:        "abc",
		SiteAppId:   "track-me-app",
	},
	{
		PublisherId: "track-me",
		Code:        "abcd",
		SiteAppId:   "track-me-site",
	},
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
		config     config.AgmaAnalytics
		shouldFail bool
	}{
		{
			name: "Test with invalid/empty URL",
			config: config.AgmaAnalytics{
				Enabled: true,
				Endpoint: config.AgmaAnalyticsHttpEndpoint{
					Url:     "%%2815197306101420000%29",
					Timeout: "1s",
					Gzip:    false,
				},
			},
			shouldFail: true,
		},
		{
			name: "Test with invalid timout",
			config: config.AgmaAnalytics{
				Enabled: true,
				Endpoint: config.AgmaAnalyticsHttpEndpoint{
					Url:     "http://localhost:8000/event",
					Timeout: "1x",
					Gzip:    false,
				},
			},
			shouldFail: true,
		},
		{
			name: "Test with no accounts",
			config: config.AgmaAnalytics{
				Enabled: true,
				Endpoint: config.AgmaAnalyticsHttpEndpoint{
					Url:     "http://localhost:8000/event",
					Timeout: "1s",
					Gzip:    false,
				},
				Buffers: config.AgmaAnalyticsBuffer{
					EventCount: 1,
					BufferSize: "1Kb",
					Timeout:    "1s",
				},
				Accounts: []config.AgmaAnalyticsAccount{},
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

func TestShouldTrackEvent(t *testing.T) {
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 1,
			BufferSize: "1Kb",
			Timeout:    "1s",
		},
		Accounts: []config.AgmaAnalyticsAccount{
			{
				PublisherId: "track-me",
				Code:        "abc",
			},
			{
				PublisherId: "",
				SiteAppId:   "track-me",
				Code:        "abc",
			},
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	// no userExt
	shouldTrack, code := logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
			App: &openrtb2.App{
				ID: "com.app.test",
				Publisher: &openrtb2.Publisher{
					ID: "track-me-not",
				},
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
			},
		},
	})

	assert.False(t, shouldTrack)
	assert.Equal(t, "", code)

	// no userExt
	shouldTrack, code = logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			App: &openrtb2.App{
				ID: "com.app.test",
				Publisher: &openrtb2.Publisher{
					ID: "track-me",
				},
			},
		},
	})

	assert.False(t, shouldTrack)
	assert.Equal(t, "", code)

	// Constent: No agma
	shouldTrack, code = logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			App: &openrtb2.App{
				ID: "com.app.test",
				Publisher: &openrtb2.Publisher{
					ID: "track-me",
				},
			},
			User: &openrtb2.User{
				Consent: "CP4LywcP4LywcLRAAAENCZCAAAIAAAIAAAAAIxQAQIwgAAAA.II7Nd_X__bX9n-_7_6ft0eY1f9_r37uQzDhfNs-8F3L_W_LwX32E7NF36tq4KmR4ku1bBIQNtHMnUDUmxaolVrzHsak2cpyNKJ_JkknsZe2dYGF9Pn9lD-YKZ7_5_9_f52T_9_9_-39z3_9f___dv_-__-vjf_599n_v9fV_78_Kf9______-____________8A",
			},
		},
	})

	assert.False(t, shouldTrack)
	assert.Equal(t, "", code)

	// Constent: No Purpose 9
	shouldTrack, code = logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			App: &openrtb2.App{
				ID: "com.app.test",
				Publisher: &openrtb2.Publisher{
					ID: "track-me",
				},
			},
			User: &openrtb2.User{
				Consent: "CP4LywcP4LywcLRAAAENCZCAAIAAAAAAAAAAIxQAQIxAAAAA.II7Nd_X__bX9n-_7_6ft0eY1f9_r37uQzDhfNs-8F3L_W_LwX32E7NF36tq4KmR4ku1bBIQNtHMnUDUmxaolVrzHsak2cpyNKJ_JkknsZe2dYGF9Pn9lD-YKZ7_5_9_f52T_9_9_-39z3_9f___dv_-__-vjf_599n_v9fV_78_Kf9______-____________8A",
			},
		},
	})

	assert.False(t, shouldTrack)
	assert.Equal(t, "", code)

	// No valid sites / apps / empty publisher app
	shouldTrack, code = logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			App: &openrtb2.App{
				ID: "",
				Publisher: &openrtb2.Publisher{
					ID: "",
				},
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
			},
		},
	})

	assert.False(t, shouldTrack)
	assert.Equal(t, "", code)

	// should allow empty accounts
	shouldTrack, code = logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			App: &openrtb2.App{
				ID: "track-me",
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
			},
		},
	})

	assert.True(t, shouldTrack)
	assert.Equal(t, "abc", code)

	// Bundle ID instead of app.id
	shouldTrack, code = logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			App: &openrtb2.App{
				Bundle: "track-me",
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
			},
		},
	})

	assert.True(t, shouldTrack)
	assert.Equal(t, "abc", code)
}

func TestShouldTrackMultipleAccounts(t *testing.T) {
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 1,
			BufferSize: "1Kb",
			Timeout:    "1s",
		},
		Accounts: []config.AgmaAnalyticsAccount{
			{
				PublisherId: "track-me-a",
				Code:        "abc",
			},
			{
				PublisherId: "track-me-b",
				Code:        "123",
			},
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	shouldTrack, code := logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
			App: &openrtb2.App{
				ID: "com.app.test",
				Publisher: &openrtb2.Publisher{
					ID: "track-me-a",
				},
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
			},
		},
	})

	assert.True(t, shouldTrack)
	assert.Equal(t, "abc", code)

	shouldTrack, code = logger.shouldTrackEvent(&openrtb_ext.RequestWrapper{
		BidRequest: &openrtb2.BidRequest{
			ID: "some-id",
			Site: &openrtb2.Site{
				ID: "site-test",
				Publisher: &openrtb2.Publisher{
					ID: "track-me-b",
				},
			},
			User: &openrtb2.User{
				Consent: agmaConsent,
			},
		},
	})

	assert.True(t, shouldTrack)
	assert.Equal(t, "123", code)
}

func TestShouldNotTrackLog(t *testing.T) {
	testCases := []struct {
		name   string
		config config.AgmaAnalytics
	}{
		{
			name: "Test with do-not-track PublisherId",
			config: config.AgmaAnalytics{
				Enabled: true,
				Endpoint: config.AgmaAnalyticsHttpEndpoint{
					Url:     "http://localhost:8000/event",
					Timeout: "5s",
				},
				Buffers: config.AgmaAnalyticsBuffer{
					EventCount: 1,
					BufferSize: "1Kb",
					Timeout:    "1s",
				},
				Accounts: []config.AgmaAnalyticsAccount{
					{
						PublisherId: "do-not-track",
						Code:        "abc",
					},
				},
			},
		},
		{
			name: "Test with do-not-track PublisherId",
			config: config.AgmaAnalytics{
				Enabled: true,
				Endpoint: config.AgmaAnalyticsHttpEndpoint{
					Url:     "http://localhost:8000/event",
					Timeout: "5s",
				},
				Buffers: config.AgmaAnalyticsBuffer{
					EventCount: 1,
					BufferSize: "1Kb",
					Timeout:    "1s",
				},
				Accounts: []config.AgmaAnalyticsAccount{
					{
						PublisherId: "track-me",
						Code:        "abc",
						SiteAppId:   "do-not-track",
					},
				},
			},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockedSender := new(MockedSender)
			mockedSender.On("Send", mock.Anything).Return(nil)
			clockMock := clock.NewMock()
			logger, err := newAgmaLogger(tc.config, mockedSender.Send, clockMock)
			assert.NoError(t, err)

			go logger.start()
			assert.Zero(t, logger.eventCount)

			logger.LogAuctionObject(&mockValidAuctionObject)
			logger.LogVideoObject(&mockValidVideoObject)
			logger.LogAmpObject(&mockValidAmpObject)

			clockMock.Add(2 * time.Minute)
			mockedSender.AssertNumberOfCalls(t, "Send", 0)
			assert.Zero(t, logger.eventCount)
		})
	}
}

func TestRaceAllEvents(t *testing.T) {
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 10000,
			BufferSize: "100Mb",
			Timeout:    "5m",
		},
		Accounts: mockValidAccounts,
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	go logger.start()

	logger.LogAuctionObject(&mockValidAuctionObject)
	logger.LogVideoObject(&mockValidVideoObject)
	logger.LogAmpObject(&mockValidAmpObject)
	clockMock.Add(10 * time.Millisecond)

	logger.mux.RLock()
	assert.Equal(t, int64(3), logger.eventCount)
	logger.mux.RUnlock()
}

func TestFlushOnSigterm(t *testing.T) {
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 10000,
			BufferSize: "100Mb",
			Timeout:    "5m",
		},
		Accounts: mockValidAccounts,
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	done := make(chan struct{})
	go func() {
		logger.start()
		close(done)
	}()

	logger.LogAuctionObject(&mockValidAuctionObject)
	logger.LogVideoObject(&mockValidVideoObject)
	logger.LogAmpObject(&mockValidAmpObject)

	logger.sigTermCh <- syscall.SIGTERM
	<-done

	time.Sleep(100 * time.Millisecond)

	mockedSender.AssertCalled(t, "Send", mock.Anything)
}

func TestRaceBufferCount(t *testing.T) {
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 2,
			BufferSize: "100Mb",
			Timeout:    "5m",
		},
		Accounts: []config.AgmaAnalyticsAccount{
			{
				PublisherId: "track-me",
				Code:        "abc",
			},
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
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
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 1000,
			BufferSize: "20Kb",
			Timeout:    "5m",
		},
		Accounts: []config.AgmaAnalyticsAccount{
			{
				PublisherId: "track-me",
				Code:        "abc",
			},
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
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
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 1000,
			BufferSize: "100mb",
			Timeout:    "5m",
		},
		Accounts: []config.AgmaAnalyticsAccount{
			{
				PublisherId: "track-me",
				Code:        "abc",
			},
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
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
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     server.URL,
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 2,
			BufferSize: "100mb",
			Timeout:    "5m",
		},
		Accounts: mockValidAccounts,
	}

	clockMock := clock.NewMock()
	clockMock.Set(time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC))

	logger, err := NewModule(&http.Client{}, cfg, clockMock)
	assert.NoError(t, err)

	logger.LogAmpObject(&mockValidAmpObject)
	logger.LogAmpObject(&mockValidAmpObject)

	time.Sleep(250 * time.Millisecond)

	expected := "[{\"type\":\"amp\",\"id\":\"some-id\",\"code\":\"abcd\",\"site\":{\"id\":\"track-me-site\",\"publisher\":{\"id\":\"track-me\"}},\"device\":{\"ua\":\"ua\"},\"user\":{\"consent\":\"" + agmaConsent + "\"},\"created_at\":\"2023-02-01T00:00:00Z\"},{\"type\":\"amp\",\"id\":\"some-id\",\"code\":\"abcd\",\"site\":{\"id\":\"track-me-site\",\"publisher\":{\"id\":\"track-me\"}},\"device\":{\"ua\":\"ua\"},\"user\":{\"consent\":\"" + agmaConsent + "\"},\"created_at\":\"2023-02-01T00:00:00Z\"}]"

	mu.Lock()
	actual := requestBodyAsString
	mu.Unlock()

	assert.Equal(t, expected, actual)
}

func TestShutdownFlush(t *testing.T) {
	cfg := config.AgmaAnalytics{
		Enabled: true,
		Endpoint: config.AgmaAnalyticsHttpEndpoint{
			Url:     "http://localhost:8000/event",
			Timeout: "5s",
		},
		Buffers: config.AgmaAnalyticsBuffer{
			EventCount: 1000,
			BufferSize: "100mb",
			Timeout:    "5m",
		},
		Accounts: []config.AgmaAnalyticsAccount{
			{
				PublisherId: "track-me",
				Code:        "abc",
			},
		},
	}
	mockedSender := new(MockedSender)
	mockedSender.On("Send", mock.Anything).Return(nil)
	clockMock := clock.NewMock()
	logger, err := newAgmaLogger(cfg, mockedSender.Send, clockMock)
	assert.NoError(t, err)

	go logger.start()
	logger.LogAuctionObject(&mockValidAuctionObject)
	logger.Shutdown()

	time.Sleep(10 * time.Millisecond)

	mockedSender.AssertCalled(t, "Send", mock.Anything)
	mockedSender.AssertNumberOfCalls(t, "Send", 1)
}
