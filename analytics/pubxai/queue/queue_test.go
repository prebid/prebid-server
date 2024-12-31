package queue

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func MockHTTPServer(statusCode int, responseBody string) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)
		w.Write([]byte(responseBody))
	}))
	return server
}

func TestEnqueue(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		bufferInterval string
		bufferSize     string
		actions        func(*GenericQueue[int], *clock.Mock)
		checkQueue     func(*testing.T, *GenericQueue[int])
	}{
		{
			name:           "Basic enqueue",
			statusCode:     http.StatusOK,
			bufferInterval: "1s",
			bufferSize:     "10",
			actions: func(q *GenericQueue[int], _ *clock.Mock) {
				q.Enqueue(1)
				q.Enqueue(2)
			},
			checkQueue: func(t *testing.T, q *GenericQueue[int]) {
				assert.Equal(t, 2, len(q.Queue), "Queue length should be 2")
			},
		},
		{
			name:           "Enqueue with flush",
			statusCode:     http.StatusOK,
			bufferInterval: "1s",
			bufferSize:     "10",
			actions: func(q *GenericQueue[int], clock *clock.Mock) {
				q.Enqueue(1)
				clock.Add(2 * time.Second)
				q.Enqueue(2)
			},
			checkQueue: func(t *testing.T, q *GenericQueue[int]) {
				assert.Equal(t, 0, len(q.Queue), "Queue should be empty after flush")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockServer := MockHTTPServer(tt.statusCode, "OK")
			defer mockServer.Close()

			mockClock := clock.NewMock()
			mockClock.Set(time.Now())

			q := NewGenericQueue[int]("test", mockServer.URL, mockServer.Client(),
				mockClock, tt.bufferInterval, tt.bufferSize)
			genericQueue := q.(*GenericQueue[int])

			tt.actions(genericQueue, mockClock)
			tt.checkQueue(t, genericQueue)
		})
	}
}

func TestUpdateConfig(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockClock.Set(time.Now())
	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	client := mockServer.Client()
	q := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")
	genericQueue := q.(*GenericQueue[int])
	genericQueue.UpdateConfig("2s", "20")
	assert.Equal(t, "2s", genericQueue.BufferInterval, "BufferInterval should be updated to 2s")
	assert.Equal(t, "20", genericQueue.BufferSize, "BufferSize should be updated to 20")
}

func TestFlushQueuedData(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		serverURL  string
		setup      func(*GenericQueue[int])
		checkQueue func(*testing.T, *GenericQueue[int])
	}{
		{
			name:       "Normal flush",
			statusCode: http.StatusOK,
			setup: func(q *GenericQueue[int]) {
				q.Enqueue(1)
				q.Enqueue(2)
			},
			checkQueue: func(t *testing.T, q *GenericQueue[int]) {
				assert.Equal(t, 0, len(q.Queue), "Queue should be empty after flushing")
			},
		},
		{
			name:       "Empty queue flush",
			statusCode: http.StatusOK,
			setup:      func(q *GenericQueue[int]) {},
			checkQueue: func(t *testing.T, q *GenericQueue[int]) {
				assert.Equal(t, 0, len(q.Queue), "Queue should be empty after flushing")
			},
		},
		{
			name:       "Bad request error",
			statusCode: http.StatusBadRequest,
			setup: func(q *GenericQueue[int]) {
				q.Enqueue(1)
				q.Enqueue(2)
			},
			checkQueue: func(t *testing.T, q *GenericQueue[int]) {
				assert.Nil(t, q.Queue, "Queue should be nil after error in flushing")
			},
		},
		{
			name:       "API error",
			statusCode: http.StatusNotFound,
			serverURL:  "testing.com",
			setup: func(q *GenericQueue[int]) {
				q.Enqueue(1)
				q.Enqueue(2)
			},
			checkQueue: func(t *testing.T, q *GenericQueue[int]) {
				assert.Nil(t, q.Queue, "Queue should be nil after error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClock := clock.NewMock()
			mockClock.Set(time.Now())

			mockServer := MockHTTPServer(tt.statusCode, "Fail")
			defer mockServer.Close()

			url := tt.serverURL
			if url == "" {
				url = mockServer.URL
			}

			genericQueue := NewGenericQueue[int]("test", url, mockServer.Client(),
				mockClock, "1s", "10")
			q := genericQueue.(*GenericQueue[int])

			tt.setup(q)
			q.flushQueuedData()
			tt.checkQueue(t, q)
		})
	}
}

func TestIsTimeToSend(t *testing.T) {
	tests := []struct {
		name           string
		bufferInterval string
		clockAdvance   time.Duration
		expected       bool
	}{
		{
			name:           "Not time to send initially",
			bufferInterval: "1s",
			clockAdvance:   0,
			expected:       false,
		},
		{
			name:           "Time to send after interval",
			bufferInterval: "1s",
			clockAdvance:   2 * time.Second,
			expected:       true,
		},
		{
			name:           "Invalid time interval",
			bufferInterval: "1i",
			clockAdvance:   0,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()

			mockClock := clock.NewMock()
			mockServer := MockHTTPServer(http.StatusOK, "OK")
			defer mockServer.Close()

			genericQueue := NewGenericQueue[int]("test", mockServer.URL,
				mockServer.Client(), mockClock, tt.bufferInterval, "10")
			q := genericQueue.(*GenericQueue[int])

			if tt.clockAdvance > 0 {
				mockClock.Add(tt.clockAdvance)
			}

			assert.Equal(t, tt.expected, q.isTimeToSend())
		})
	}
}

func TestNewBidQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	bufferInterval := "5s"
	bufferSize := "10"

	t.Run("Test WinningBidQueue creation", func(t *testing.T) {
		queue := NewBidQueue("win", "http://example.com", mockClient, mockClock, bufferInterval, bufferSize)
		assert.NotNil(t, queue)
		_, ok := queue.(*WinningBidQueue)
		assert.True(t, ok, "Expected type *WinningBidQueue")
	})

	t.Run("Test AuctionBidQueue creation", func(t *testing.T) {
		queue := NewBidQueue("auction", "http://example.com", mockClient, mockClock, bufferInterval, bufferSize)
		assert.NotNil(t, queue)
		_, ok := queue.(*AuctionBidsQueue)
		assert.True(t, ok, "Expected type *AuctionBidQueue")
	})

	t.Run("Test invalid queueType", func(t *testing.T) {
		queue := NewBidQueue("invalid", "http://example.com", mockClient, mockClock, bufferInterval, bufferSize)
		assert.Nil(t, queue)
	})
}
