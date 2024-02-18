package pubxai

import (
	"net/http"
	"os"
	"testing"

	"github.com/benbjohnson/clock"
)

func TestInitializePubxAIModule(t *testing.T) {
	// Mocks
	mockClient := &http.Client{}
	mockClock := clock.NewMock()

	// Test cases
	t.Run("ValidInputs", func(t *testing.T) {
		// Test with valid inputs
		_, err := InitializePubxAIModule(mockClient, "publisherID", "endpoint", "10s", "1MB", 50, "1m", mockClock)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
	})

	t.Run("NilClient", func(t *testing.T) {
		// Test with nil client
		_, err := InitializePubxAIModule(nil, "publisherID", "endpoint", "10s", "1MB", 50, "1m", mockClock)
		if err == nil {
			t.Error("Expected error for nil client, but got nil")
		}
	})

	t.Run("EmptyPublisherIDAndEndpoint", func(t *testing.T) {
		// Test with empty publisher ID and endpoint
		_, err := InitializePubxAIModule(mockClient, "", "", "10s", "1MB", 50, "1m", mockClock)
		if err == nil {
			t.Error("Expected error for empty publisher ID and endpoint, but got nil")
		}
	})

	t.Run("InvalidBufferIntervalAndSize", func(t *testing.T) {
		// Test with invalid buffer interval and size
		_, err := InitializePubxAIModule(mockClient, "publisherID", "endpoint", "invalid", "invalid", 50, "1m", mockClock)
		if err == nil {
			t.Error("Expected error for invalid buffer interval and size, but got nil")
		}
	})
}

func TestStart(t *testing.T) {
	// Mocks
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})
	mockConfigChannel := make(chan *Configuration)

	// Test cases
	t.Run("SignalReceived", func(t *testing.T) {
		// Test for proper termination when receiving a signal
		pb := PubxaiModule{
			sigTermCh: mockSigTermCh,
			stopCh:    mockStopCh,
		}
		go pb.start(mockConfigChannel)
		mockSigTermCh <- os.Interrupt
		_, open := <-mockStopCh
		if open {
			t.Error("Stop channel not closed after receiving signal")
		}
	})

	t.Run("ConfigUpdateReceived", func(t *testing.T) {
		// Test for proper update of configuration when receiving a configuration update
		pb := PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB"),
			auctionBidsQueue: NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB"),
			cfg: &Configuration{
				PublisherId:        "oldPublisherID",
				BufferInterval:     "10s",
				BufferSize:         "1MB",
				SamplingPercentage: 50,
			},
		}
		go pb.start(mockConfigChannel)
		mockConfigChannel <- &Configuration{
			PublisherId:        "newPublisherID",
			BufferInterval:     "20s",
			BufferSize:         "2MB",
			SamplingPercentage: 100,
		}
		if pb.cfg.PublisherId != "newPublisherID" && pb.cfg.BufferInterval != "20s" && pb.cfg.BufferSize != "2MB" && pb.cfg.SamplingPercentage != 100 {
			t.Errorf("Expected PublisherId to be 'newPublisherID', got %s", pb.cfg.PublisherId)
		}
	})
}

func TestUpdateConfig(t *testing.T) {
	// Mocks
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})

	mockModule := &PubxaiModule{
		sigTermCh:        mockSigTermCh,
		stopCh:           mockStopCh,
		winBidsQueue:     NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB"),
		auctionBidsQueue: NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB"),
		cfg: &Configuration{
			PublisherId:        "oldPublisherID",
			BufferInterval:     "10s",
			BufferSize:         "1MB",
			SamplingPercentage: 50,
		},
	}

	// Test cases
	t.Run("SameConfig", func(t *testing.T) {
		// Test with same configuration
		newConfig := &Configuration{
			PublisherId:        "oldPublisherID",
			BufferInterval:     "10s",
			BufferSize:         "1MB",
			SamplingPercentage: 50,
		}
		mockModule.updateConfig(newConfig)
		if mockModule.cfg.PublisherId != "oldPublisherID" && mockModule.cfg.BufferInterval != "10s" && mockModule.cfg.BufferSize != "1MB" && mockModule.cfg.SamplingPercentage != 50 {
			t.Errorf("Expected Old Configuration, got %v", mockModule.cfg)
		}
	})

	t.Run("DifferentConfig", func(t *testing.T) {
		// Test with different configuration
		newConfig := &Configuration{
			PublisherId:        "newPublisherID",
			BufferInterval:     "20s",
			BufferSize:         "2MB",
			SamplingPercentage: 100,
		}
		mockModule.updateConfig(newConfig)
		if mockModule.cfg.PublisherId != "newPublisherID" && mockModule.cfg.BufferInterval != "20s" && mockModule.cfg.BufferSize != "2MB" && mockModule.cfg.SamplingPercentage != 100 {
			t.Errorf("Expected New Configuration, got %v", mockModule.cfg)
		}
	})
}

func TestLogAuctionObject(t *testing.T) {
	// Mocks
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})
	mockAuctionObject := GetMockAuctionObject()

	// Test cases
	t.Run("NonNilAuctionObject", func(t *testing.T) {
		// Test with non-nil auction object
		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB"),
			auctionBidsQueue: NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB"),
			cfg: &Configuration{
				PublisherId:        "oldPublisherID",
				BufferInterval:     "10m",
				BufferSize:         "10MB",
				SamplingPercentage: 100,
			},
		}
		pb.LogAuctionObject(mockAuctionObject)
		if len(pb.winBidsQueue.queue) == 0 {
			t.Error("Expected non-nil winBidsQueue, got nil")
		}
	})

	t.Run("NilAuctionObject", func(t *testing.T) {
		// Test with nil auction object
		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB"),
			auctionBidsQueue: NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB"),
			cfg: &Configuration{
				PublisherId:        "oldPublisherID",
				BufferInterval:     "10m",
				BufferSize:         "10MB",
				SamplingPercentage: 100,
			},
		}
		pb.LogAuctionObject(nil)
		if len(pb.winBidsQueue.queue) != 0 {
			t.Error("Expected empty winBidsQueue, got not Empty")
		}

	})
}
