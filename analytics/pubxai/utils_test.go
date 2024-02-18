package pubxai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/prebid/openrtb/v20/openrtb2"
	"github.com/prebid/prebid-server/v2/analytics"
	"github.com/prebid/prebid-server/v2/openrtb_ext"
)

func GetMockAuctionObject() *analytics.AuctionObject {

	requestData, err := os.ReadFile("./mocks/mock_openrtb_request.json")
	if err != nil {
		panic(err)
	}

	// Unmarshal the request data into the appropriate struct
	var bidRequest openrtb2.BidRequest
	if err := json.Unmarshal(requestData, &bidRequest); err != nil {
		panic(err)
	}

	// Read the response from mock_openrtb_response.json
	responseData, err := os.ReadFile("./mocks/mock_openrtb_response.json")
	if err != nil {
		panic(err)
	}

	var bidResponse openrtb2.BidResponse
	if err := json.Unmarshal(responseData, &bidResponse); err != nil {
		panic(err)
	}
	ao := &analytics.AuctionObject{
		StartTime: time.Now(),
		Status:    http.StatusOK,
		RequestWrapper: &openrtb_ext.RequestWrapper{
			BidRequest: &bidRequest,
		},
		Response: &bidResponse,
	}
	return ao
}

func TestNewBidQueue(t *testing.T) {
	// Mock HTTP client
	httpClient := &http.Client{}

	// Create a new clock for testing
	testClock := clock.NewMock()

	// Call NewBidQueue with valid parameters
	queue := NewBidQueue("testType", "http://example.com", httpClient, testClock, "10s", "10MB")

	// Assert queue and its internal fields
	if queue == nil {
		t.Error("Expected a non-nil queue, got nil")
	}
}

func TestProcessAuctionData(t *testing.T) {
	// Mock HTTP client
	httpClient := &http.Client{}

	// Create a new clock for testing
	testClock := clock.NewMock()

	ao := GetMockAuctionObject()
	// Create a mock PubxaiModule
	p := &PubxaiModule{
		// Initialize necessary fields for testing
		winBidsQueue:     NewBidQueue("win", "http://example.com/win", httpClient, testClock, "10m", "10MB"),
		auctionBidsQueue: NewBidQueue("auction", "http://example.com/auction", httpClient, testClock, "10m", "10MB"),
	}

	// Call processAuctionData
	p.processAuctionData(ao)
	//check if the queue is not empty
	if len(p.winBidsQueue.queue) == 0 {
		t.Error("Expected a non-empty queue, got empty")
	}
}

// MockHTTPServer returns a new mock HTTP server for testing purposes
func MockHTTPServer(statusCode int, responseBody string) *httptest.Server {
	// Create a new mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the status code
		w.WriteHeader(statusCode)
		// Write the response body
		w.Write([]byte(responseBody))
	}))
	return server
}

func TestIsTimeToSend(t *testing.T) {
	// Create a new clock for testing
	testClock := clock.NewMock()

	// Create a new BidQueue with test parameters
	bidQueue := &BidQueue{
		clock:          testClock,
		bufferInterval: "10s",
		lastSentTime:   testClock.Now().Add(-time.Second * 15), // Last sent time 15 seconds ago
	}

	// Call isTimeToSend
	result := bidQueue.isTimeToSend()

	// Expected result is true since the buffer interval has elapsed
	if !result {
		t.Error("Expected isTimeToSend to return true, got false")
	}

	// Update the last sent time to within the buffer interval
	bidQueue.lastSentTime = testClock.Now().Add(-time.Second * 5)

	// Call isTimeToSend again
	result = bidQueue.isTimeToSend()

	// Expected result is false since the buffer interval has not elapsed
	if result {
		t.Error("Expected isTimeToSend to return false, got true")
	}
}

func TestFlushQueuedData(t *testing.T) {
	// Create a mock HTTP server
	mockServer := MockHTTPServer(http.StatusOK, "OK")
	defer mockServer.Close()

	// Create a new HTTP client with the mock server URL
	httpClient := mockServer.Client()

	// Create a new clock for testing
	testClock := clock.NewMock()

	// Create a new BidQueue with test parameters
	bidQueue := &BidQueue{
		queue:          []Bid{{AdUnitCode: "test", BidId: "12345"}},
		endpoint:       mockServer.URL,
		httpClient:     httpClient,
		clock:          testClock,
		bufferSize:     "10MB",
		bufferInterval: "10s",
	}

	// Call flushQueuedData
	bidQueue.flushQueuedData()

	// Assert that the queue is cleared after sending the data
	if len(bidQueue.queue) != 0 {
		t.Error("Expected queue to be cleared after sending data, got non-empty queue")
	}
}

func TestEnqueue(t *testing.T) {
	httpClient := &http.Client{}

	testClock := clock.NewMock()
	bidQueue := NewBidQueue("test", "http://example.com/test", httpClient, testClock, "10s", "10MB")

	// Create some sample Bid objects
	bid1 := Bid{AdUnitCode: "ad1", BidId: "bid1"}
	bid2 := Bid{AdUnitCode: "ad2", BidId: "bid2"}

	// Enqueue the Bid objects
	bidQueue.Enqueue(bid1)
	bidQueue.Enqueue(bid2)

	// Check if the queue contains the enqueued Bid objects
	if len(bidQueue.queue) != 2 {
		t.Error("Expected 2 items in the queue, got", len(bidQueue.queue))
	}

}
