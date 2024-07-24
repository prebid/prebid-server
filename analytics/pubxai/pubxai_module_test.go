package pubxai

import (
	"net/http"
	"os"
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/prebid/prebid-server/v2/analytics"
)

func GetMockAuctionObject() *analytics.AuctionObject {

	lo := GetMockLogObject()
	ao := &analytics.AuctionObject{
		StartTime:      lo.StartTime,
		Status:         lo.Status,
		RequestWrapper: lo.RequestWrapper,
		Response:       lo.Response,
	}
	return ao
}

func GetMockVideoObject() *analytics.VideoObject {
	lo := GetMockLogObject()
	vo := &analytics.VideoObject{
		StartTime:      lo.StartTime,
		Status:         lo.Status,
		RequestWrapper: lo.RequestWrapper,
		Response:       lo.Response,
	}
	return vo
}

func GetMockAMPObject() *analytics.AmpObject {
	lo := GetMockLogObject()
	ao := &analytics.AmpObject{
		StartTime:       lo.StartTime,
		Status:          lo.Status,
		RequestWrapper:  lo.RequestWrapper,
		AuctionResponse: lo.Response,
	}
	return ao
}

func TestInitializePubxAIModule(t *testing.T) {
	mockClient := &http.Client{}
	mockClock := clock.NewMock()

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
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})
	mockConfigChannel := make(chan *Configuration)

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
		winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
		auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

		pb := PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     winQueue.(*WinningBidQueue),
			auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
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
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})
	winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
	auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

	mockModule := &PubxaiModule{
		sigTermCh:        mockSigTermCh,
		stopCh:           mockStopCh,
		winBidsQueue:     winQueue.(*WinningBidQueue),
		auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
		cfg: &Configuration{
			PublisherId:        "oldPublisherID",
			BufferInterval:     "10s",
			BufferSize:         "1MB",
			SamplingPercentage: 50,
		},
	}

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
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})
	mockAuctionObject := GetMockAuctionObject()

	t.Run("NonNilAuctionObject", func(t *testing.T) {

		winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
		auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

		// Test with non-nil auction object
		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     winQueue.(*WinningBidQueue),
			auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
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
		winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
		auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     winQueue.(*WinningBidQueue),
			auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
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

func TestLogVideoObject(t *testing.T) {
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})
	mockVideoObject := GetMockVideoObject()

	t.Run("NonNilAuctionObject", func(t *testing.T) {
		// Test with non-nil auction object
		winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
		auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     winQueue.(*WinningBidQueue),
			auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
			cfg: &Configuration{
				PublisherId:        "oldPublisherID",
				BufferInterval:     "10m",
				BufferSize:         "10MB",
				SamplingPercentage: 100,
			},
		}
		pb.LogVideoObject(mockVideoObject)
		if len(pb.winBidsQueue.queue) == 0 {
			t.Error("Expected non-nil winBidsQueue, got nil")
		}
	})

	t.Run("NilAuctionObject", func(t *testing.T) {
		// Test with nil auction object
		winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
		auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     winQueue.(*WinningBidQueue),
			auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
			cfg: &Configuration{
				PublisherId:        "oldPublisherID",
				BufferInterval:     "10m",
				BufferSize:         "10MB",
				SamplingPercentage: 100,
			},
		}
		pb.LogVideoObject(nil)
		if len(pb.winBidsQueue.queue) != 0 {
			t.Error("Expected empty winBidsQueue, got not Empty")
		}

	})
}

func TestLogAmpObject(t *testing.T) {
	mockClient := &http.Client{}
	mockClock := clock.NewMock()
	mockSigTermCh := make(chan os.Signal)
	mockStopCh := make(chan struct{})
	mockAmpObject := GetMockAMPObject()

	t.Run("NonNilAuctionObject", func(t *testing.T) {
		// Test with non-nil auction object
		winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
		auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     winQueue.(*WinningBidQueue),
			auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
			cfg: &Configuration{
				PublisherId:        "oldPublisherID",
				BufferInterval:     "10m",
				BufferSize:         "10MB",
				SamplingPercentage: 100,
			},
		}
		pb.LogAmpObject(mockAmpObject)
		if len(pb.winBidsQueue.Queue) == 0 {
			t.Error("Expected non-nil winBidsQueue, got nil")
		}
	})

	t.Run("NilAuctionObject", func(t *testing.T) {
		// Test with nil auction object
		winQueue := NewBidQueue("win", "endpoint/win", mockClient, mockClock, "10s", "1MB")
		auctionQueue := NewBidQueue("auction", "endpoint/auction", mockClient, mockClock, "10s", "1MB")

		pb := &PubxaiModule{
			sigTermCh:        mockSigTermCh,
			stopCh:           mockStopCh,
			winBidsQueue:     winQueue.(*WinningBidQueue),
			auctionBidsQueue: auctionQueue.(*AuctionBidsQueue),
			cfg: &Configuration{
				PublisherId:        "oldPublisherID",
				BufferInterval:     "10m",
				BufferSize:         "10MB",
				SamplingPercentage: 100,
			},
		}
		pb.LogAmpObject(nil)
		if len(pb.winBidsQueue.queue) != 0 {
			t.Error("Expected empty winBidsQueue, got not Empty")
		}

	})
}
