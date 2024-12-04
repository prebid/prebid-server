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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	client := mockServer.Client()
	mockClock := clock.NewMock()
	mockClock.Set(time.Now())
	q := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")

	genericQueue := q.(*GenericQueue[int])

	genericQueue.Enqueue(1)
	genericQueue.Enqueue(2)

	assert.Equal(t, 2, len(genericQueue.Queue), "Queue length should be 2")
}

func TestEnqueue_Flush(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	client := mockServer.Client()
	mockClock := clock.NewMock()
	mockClock.Set(time.Now())
	q := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")

	genericQueue := q.(*GenericQueue[int])

	genericQueue.Enqueue(1)
	mockClock.Add(2 * time.Second) // Wait for flush to happen
	genericQueue.Enqueue(2)

	assert.Equal(t, 0, len(genericQueue.Queue), "Queue should be empty after flush")
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
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockClock.Set(time.Now())

	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	client := mockServer.Client()

	genericQueue := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")
	q := genericQueue.(*GenericQueue[int])
	q.Enqueue(1)
	q.Enqueue(2)

	q.flushQueuedData()

	assert.Equal(t, 0, len(q.Queue), "Queue should be empty after flushing")
}

func TestFlushQueuedData_EmptyQueue(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockClock.Set(time.Now())

	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	client := mockServer.Client()

	genericQueue := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")
	q := genericQueue.(*GenericQueue[int])

	q.flushQueuedData()

	assert.Equal(t, 0, len(q.Queue), "Queue should be empty after flushing")
}
func TestFlushQueuedData_Error(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockClock.Set(time.Now())

	mockServer := MockHTTPServer(http.StatusBadRequest, "Fail")
	defer mockServer.Close()

	client := mockServer.Client()
	// Create a queue with a mock server that will return an error
	genericQueue := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")
	q := genericQueue.(*GenericQueue[int])
	q.Enqueue(1)
	q.Enqueue(2)

	q.flushQueuedData()

	assert.Nil(t, q.Queue, "Queue should be nil after error in flushing")
}

func TestFlushQueuedData_ApiError(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockClock.Set(time.Now())

	mockServer := MockHTTPServer(http.StatusNotFound, "Fail")
	defer mockServer.Close()

	client := mockServer.Client()

	genericQueue := NewGenericQueue[int]("test", "testing.com", client, mockClock, "1s", "10")
	q := genericQueue.(*GenericQueue[int])
	q.Enqueue(1)
	q.Enqueue(2)

	q.flushQueuedData()

	assert.Nil(t, q.Queue, "Queue should be nil after error")
}

func TestFlushQueuedData_ApiNon200(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockClock.Set(time.Now())

	mockServer := MockHTTPServer(http.StatusBadRequest, "Fail")
	defer mockServer.Close()

	client := mockServer.Client()

	genericQueue := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")
	q := genericQueue.(*GenericQueue[int])
	q.Enqueue(1)
	q.Enqueue(2)

	q.flushQueuedData()

	assert.Nil(t, q.Queue, "Queue should be nil after error in flushing")
}

func TestIsTimeToSend(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	client := mockServer.Client()
	genericQueue := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1s", "10")
	q := genericQueue.(*GenericQueue[int])
	assert.False(t, q.isTimeToSend(), "Initially, it should not be time to send")

	mockClock.Add(2 * time.Second)
	assert.True(t, q.isTimeToSend(), "After 2 seconds, it should be time to send")
}

func TestIsTimeToSend_InvalidTime(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockClock := clock.NewMock()
	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	client := mockServer.Client()
	genericQueue := NewGenericQueue[int]("test", mockServer.URL, client, mockClock, "1i", "10")
	q := genericQueue.(*GenericQueue[int])
	assert.False(t, q.isTimeToSend(), "Invalid time should default to false")
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
