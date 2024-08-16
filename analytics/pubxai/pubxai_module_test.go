package pubxai

import (
	"net/http"
	"os"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/golang/mock/gomock"
	"github.com/prebid/prebid-server/v2/analytics"
	config "github.com/prebid/prebid-server/v2/analytics/pubxai/config"
	configMock "github.com/prebid/prebid-server/v2/analytics/pubxai/config/mocks"
	processorMock "github.com/prebid/prebid-server/v2/analytics/pubxai/processor/mocks"
	queue "github.com/prebid/prebid-server/v2/analytics/pubxai/queue"
	queueMock "github.com/prebid/prebid-server/v2/analytics/pubxai/queue/mocks"
	"github.com/prebid/prebid-server/v2/analytics/pubxai/utils"
	"github.com/stretchr/testify/assert"
)

func TestInitializePubxAIModule(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := &http.Client{}
	mockClock := clock.NewMock()

	// Test case: successful initialization
	t.Run("successful initialization", func(t *testing.T) {
		publisherId := "testPublisher"
		endpoint := "http://test-endpoint"
		bufferInterval := "10s"
		bufferSize := "1MB"
		SamplingPercentage := 50
		configRefresh := "1m"
		module, err := InitializePubxAIModule(mockClient, publisherId, endpoint, bufferInterval, bufferSize, SamplingPercentage, configRefresh, mockClock)
		assert.NoError(t, err)
		assert.NotNil(t, module)
	})

	// Test case: client is nil
	t.Run("client is nil", func(t *testing.T) {
		module, err := InitializePubxAIModule(nil, "testPublisher", "http://test-endpoint", "10s", "1MB", 50, "1m", mockClock)
		assert.Error(t, err)
		assert.Nil(t, module)
	})

	// Test case: empty publisherId
	t.Run("empty publisherId", func(t *testing.T) {
		module, err := InitializePubxAIModule(mockClient, "", "http://test-endpoint", "10s", "1MB", 50, "1m", mockClock)
		assert.Error(t, err)
		assert.Nil(t, module)
	})

	// Test case: empty endpoint
	t.Run("empty endpoint", func(t *testing.T) {
		module, err := InitializePubxAIModule(mockClient, "testPublisher", "", "10s", "1MB", 50, "1m", mockClock)
		assert.Error(t, err)
		assert.Nil(t, module)
	})

	// Test case: invalid bufferInterval
	t.Run("invalid bufferInterval", func(t *testing.T) {
		module, err := InitializePubxAIModule(mockClient, "testPublisher", "http://test-endpoint", "invalid", "1MB", 50, "1m", mockClock)
		assert.Error(t, err)
		assert.Nil(t, module)
	})

	// Test case: invalid bufferSize
	t.Run("invalid bufferSize", func(t *testing.T) {
		module, err := InitializePubxAIModule(mockClient, "testPublisher", "http://test-endpoint", "10s", "invalid", 50, "1m", mockClock)
		assert.Error(t, err)
		assert.Nil(t, module)
	})
}

func TestPubxaiModule_start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockConfigService := configMock.NewMockConfigService(ctrl)
	mockAuctionBidsQueue := queueMock.NewMockAuctionBidsQueueInterface(ctrl)
	mockWinBidsQueue := queueMock.NewMockWinningBidQueueInterface(ctrl)

	// Test case: update configuration
	t.Run("update configuration", func(t *testing.T) {
		// Create channels
		configChan := make(chan *config.Configuration)
		sigTermChan := make(chan os.Signal)
		stopChan := make(chan struct{})
		// Create a PubxaiModule instance
		p := &PubxaiModule{
			sigTermCh:     sigTermChan,
			stopCh:        stopChan,
			cfg:           &config.Configuration{},
			configService: mockConfigService,
			auctionBidsQueue: &queue.AuctionBidsQueue{
				QueueService: mockAuctionBidsQueue,
			},
			winBidsQueue: &queue.WinningBidQueue{
				QueueService: mockWinBidsQueue,
			},
		}
		go p.start(configChan)
		newConfig := &config.Configuration{
			PublisherId: "newPublisher",
		}
		mockConfigService.EXPECT().IsSameAs(newConfig, p.cfg).Return(false)
		mockAuctionBidsQueue.EXPECT().UpdateConfig(newConfig.BufferInterval, newConfig.BufferSize).Times(1)
		mockWinBidsQueue.EXPECT().UpdateConfig(newConfig.BufferInterval, newConfig.BufferSize).Times(1)
		configChan <- newConfig

		time.Sleep(100 * time.Millisecond) // give some time for the goroutine to process the update

		assert.Equal(t, newConfig, p.cfg)
		close(sigTermChan) // stop the goroutine
	})

	// Test case: terminate on signal
	t.Run("terminate on signal", func(t *testing.T) {
		// Create channels
		configChan := make(chan *config.Configuration)
		sigTermChan := make(chan os.Signal)
		stopChan := make(chan struct{})
		// Create a PubxaiModule instance
		p := &PubxaiModule{
			sigTermCh:     sigTermChan,
			stopCh:        stopChan,
			cfg:           &config.Configuration{},
			configService: mockConfigService,
			auctionBidsQueue: &queue.AuctionBidsQueue{
				QueueService: mockAuctionBidsQueue,
			},
			winBidsQueue: &queue.WinningBidQueue{
				QueueService: mockWinBidsQueue,
			},
		}
		go p.start(configChan)

		sigTermChan <- syscall.SIGTERM

		time.Sleep(100 * time.Millisecond) // give some time for the goroutine to terminate

		select {
		case <-stopChan:
			// success, the channel was closed
		default:
			t.Errorf("expected stopChan to be closed")
		}
	})
}

func TestPubxaiModule_updateConfig(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockConfigService := configMock.NewMockConfigService(ctrl)
	mockAuctionBidsQueue := queueMock.NewMockAuctionBidsQueueInterface(ctrl)
	mockWinBidsQueue := queueMock.NewMockWinningBidQueueInterface(ctrl)

	// Create a PubxaiModule instance
	p := &PubxaiModule{
		cfg:           &config.Configuration{},
		configService: mockConfigService,
		auctionBidsQueue: &queue.AuctionBidsQueue{
			QueueService: mockAuctionBidsQueue,
		},
		winBidsQueue: &queue.WinningBidQueue{
			QueueService: mockWinBidsQueue,
		},
		muxConfig: sync.RWMutex{},
	}

	// Test case: config is the same
	t.Run("config is the same", func(t *testing.T) {
		newConfig := &config.Configuration{
			PublisherId:    "samePublisher",
			BufferInterval: "10s",
			BufferSize:     "1MB",
		}
		mockConfigService.EXPECT().IsSameAs(newConfig, p.cfg).Return(true)
		mockAuctionBidsQueue.EXPECT().UpdateConfig(gomock.Any(), gomock.Any()).Times(0)
		mockWinBidsQueue.EXPECT().UpdateConfig(gomock.Any(), gomock.Any()).Times(0)
		p.updateConfig(newConfig)
	})

	// Test case: config is different
	t.Run("config is different", func(t *testing.T) {
		oldConfig := &config.Configuration{
			PublisherId:    "oldPublisher",
			BufferInterval: "10s",
			BufferSize:     "1MB",
		}
		p.cfg = oldConfig

		newConfig := &config.Configuration{
			PublisherId:    "newPublisher",
			BufferInterval: "20s",
			BufferSize:     "2MB",
		}

		mockConfigService.EXPECT().IsSameAs(newConfig, p.cfg).Return(false)
		mockAuctionBidsQueue.EXPECT().UpdateConfig(newConfig.BufferInterval, newConfig.BufferSize).Times(1)
		mockWinBidsQueue.EXPECT().UpdateConfig(newConfig.BufferInterval, newConfig.BufferSize).Times(1)
		p.updateConfig(newConfig)

		assert.Equal(t, newConfig, p.cfg)
	})
}

func TestPubxaiModule_pushToQueue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockAuctionBidsQueue := queueMock.NewMockAuctionBidsQueueInterface(ctrl)
	mockWinBidsQueue := queueMock.NewMockWinningBidQueueInterface(ctrl)

	// Create a PubxaiModule instance
	p := &PubxaiModule{
		auctionBidsQueue: &queue.AuctionBidsQueue{
			QueueService: mockAuctionBidsQueue,
		},
		winBidsQueue: &queue.WinningBidQueue{
			QueueService: mockWinBidsQueue,
		},
	}

	// Test case: only winning bids
	t.Run("only winning bids", func(t *testing.T) {
		winningBids := []utils.WinningBid{
			{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}, WinningBid: utils.Bid{BidId: "bid1"}},
			{AuctionDetail: utils.AuctionDetail{AuctionId: "auction2"}, WinningBid: utils.Bid{BidId: "bid2"}},
		}

		mockWinBidsQueue.EXPECT().Enqueue(winningBids[0]).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[1]).Times(1)
		mockAuctionBidsQueue.EXPECT().Enqueue(gomock.Any()).Times(0)

		p.pushToQueue(nil, winningBids)
	})

	// Test case: only auction bids
	t.Run("only auction bids", func(t *testing.T) {
		auctionBids := &utils.AuctionBids{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}}

		mockAuctionBidsQueue.EXPECT().Enqueue(*auctionBids).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(gomock.Any()).Times(0)

		p.pushToQueue(auctionBids, nil)
	})

	// Test case: both auction bids and winning bids
	t.Run("both auction bids and winning bids", func(t *testing.T) {
		auctionBids := &utils.AuctionBids{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}}
		winningBids := []utils.WinningBid{
			{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}, WinningBid: utils.Bid{BidId: "bid1"}},
			{AuctionDetail: utils.AuctionDetail{AuctionId: "auction2"}, WinningBid: utils.Bid{BidId: "bid2"}},
		}

		mockAuctionBidsQueue.EXPECT().Enqueue(*auctionBids).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[0]).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[1]).Times(1)

		p.pushToQueue(auctionBids, winningBids)
	})

	// Test case: neither auction bids nor winning bids
	t.Run("neither auction bids nor winning bids", func(t *testing.T) {
		mockAuctionBidsQueue.EXPECT().Enqueue(gomock.Any()).Times(0)
		mockWinBidsQueue.EXPECT().Enqueue(gomock.Any()).Times(0)

		p.pushToQueue(nil, nil)
	})
}

func TestPubxaiModule_LogAuctionObject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockProcessorService := processorMock.NewMockProcessorService(ctrl)
	mockAuctionBidsQueue := queueMock.NewMockAuctionBidsQueueInterface(ctrl)
	mockWinBidsQueue := queueMock.NewMockWinningBidQueueInterface(ctrl)

	// Set up a PubxaiModule instance
	p := &PubxaiModule{
		processorService: mockProcessorService,
		auctionBidsQueue: &queue.AuctionBidsQueue{
			QueueService: mockAuctionBidsQueue,
		},
		winBidsQueue: &queue.WinningBidQueue{
			QueueService: mockWinBidsQueue,
		},
	}

	auctionBids := &utils.AuctionBids{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}}
	winningBids := []utils.WinningBid{
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}, WinningBid: utils.Bid{BidId: "bid1"}},
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction2"}, WinningBid: utils.Bid{BidId: "bid2"}},
	}

	// auctionObj is nil
	t.Run("Empty AuctionObject", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 0}

		mockProcessorService.EXPECT().ProcessLogData(gomock.Any()).Times(0)

		p.LogAuctionObject(nil)
	})
	// Test case: Sampling percentage too low
	t.Run("Sampling percentage too low", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 0}

		ao := &analytics.AuctionObject{
			Status: 0,
		}
		mockProcessorService.EXPECT().ProcessLogData(gomock.Any()).Times(0)

		p.LogAuctionObject(ao)
	})

	// Test case: Sampling percentage allows logging
	t.Run("Sampling percentage allows logging", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 100}

		ao := &analytics.AuctionObject{
			Status:         0,
			Errors:         nil,
			Response:       nil,
			StartTime:      time.Now(),
			SeatNonBid:     nil,
			RequestWrapper: nil,
		}

		logObject := &utils.LogObject{
			Status:         ao.Status,
			Errors:         ao.Errors,
			Response:       ao.Response,
			StartTime:      ao.StartTime,
			SeatNonBid:     ao.SeatNonBid,
			RequestWrapper: ao.RequestWrapper,
		}

		mockProcessorService.EXPECT().ProcessLogData(logObject).Return(auctionBids, winningBids).Times(1)
		mockAuctionBidsQueue.EXPECT().Enqueue(*auctionBids).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[0]).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[1]).Times(1)

		p.LogAuctionObject(ao)
	})
}

func TestPubxaiModule_VideoObject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockProcessorService := processorMock.NewMockProcessorService(ctrl)
	mockAuctionBidsQueue := queueMock.NewMockAuctionBidsQueueInterface(ctrl)
	mockWinBidsQueue := queueMock.NewMockWinningBidQueueInterface(ctrl)

	// Set up a PubxaiModule instance
	p := &PubxaiModule{
		processorService: mockProcessorService,
		auctionBidsQueue: &queue.AuctionBidsQueue{
			QueueService: mockAuctionBidsQueue,
		},
		winBidsQueue: &queue.WinningBidQueue{
			QueueService: mockWinBidsQueue,
		},
	}

	auctionBids := &utils.AuctionBids{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}}
	winningBids := []utils.WinningBid{
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}, WinningBid: utils.Bid{BidId: "bid1"}},
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction2"}, WinningBid: utils.Bid{BidId: "bid2"}},
	}
	// Test case: Sampling percentage too low
	t.Run("Empty AuctionObject", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 0}

		mockProcessorService.EXPECT().ProcessLogData(gomock.Any()).Times(0)

		p.LogVideoObject(nil)
	})
	t.Run("Sampling percentage too low", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 0}

		vo := &analytics.VideoObject{
			Status: 0,
		}
		mockProcessorService.EXPECT().ProcessLogData(gomock.Any()).Times(0)

		p.LogVideoObject(vo)
	})

	// Test case: Sampling percentage allows logging
	t.Run("Sampling percentage allows logging", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 100}

		vo := &analytics.VideoObject{
			Status:         0,
			Errors:         nil,
			Response:       nil,
			StartTime:      time.Now(),
			SeatNonBid:     nil,
			RequestWrapper: nil,
		}

		logObject := &utils.LogObject{
			Status:         vo.Status,
			Errors:         vo.Errors,
			Response:       vo.Response,
			StartTime:      vo.StartTime,
			SeatNonBid:     vo.SeatNonBid,
			RequestWrapper: vo.RequestWrapper,
		}

		mockProcessorService.EXPECT().ProcessLogData(logObject).Return(auctionBids, winningBids).Times(1)
		mockAuctionBidsQueue.EXPECT().Enqueue(*auctionBids).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[0]).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[1]).Times(1)

		p.LogVideoObject(vo)
	})
}

func TestPubxaiModule_LogAmpObject(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Mock dependencies
	mockProcessorService := processorMock.NewMockProcessorService(ctrl)
	mockAuctionBidsQueue := queueMock.NewMockAuctionBidsQueueInterface(ctrl)
	mockWinBidsQueue := queueMock.NewMockWinningBidQueueInterface(ctrl)

	// Set up a PubxaiModule instance
	p := &PubxaiModule{
		processorService: mockProcessorService,
		auctionBidsQueue: &queue.AuctionBidsQueue{
			QueueService: mockAuctionBidsQueue,
		},
		winBidsQueue: &queue.WinningBidQueue{
			QueueService: mockWinBidsQueue,
		},
	}

	auctionBids := &utils.AuctionBids{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}}
	winningBids := []utils.WinningBid{
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction1"}, WinningBid: utils.Bid{BidId: "bid1"}},
		{AuctionDetail: utils.AuctionDetail{AuctionId: "auction2"}, WinningBid: utils.Bid{BidId: "bid2"}},
	}
	t.Run("Empty AuctionObject", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 0}

		mockProcessorService.EXPECT().ProcessLogData(gomock.Any()).Times(0)

		p.LogAmpObject(nil)
	})
	// Test case: Sampling percentage too low
	t.Run("Sampling percentage too low", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 0}

		ampo := &analytics.AmpObject{
			Status: 0,
		}
		mockProcessorService.EXPECT().ProcessLogData(gomock.Any()).Times(0)

		p.LogAmpObject(ampo)
	})

	// Test case: Sampling percentage allows logging
	t.Run("Sampling percentage allows logging", func(t *testing.T) {
		p.cfg = &config.Configuration{SamplingPercentage: 100}

		ampo := &analytics.AmpObject{
			Status:          0,
			Errors:          nil,
			AuctionResponse: nil,
			StartTime:       time.Now(),
			SeatNonBid:      nil,
			RequestWrapper:  nil,
		}

		logObject := &utils.LogObject{
			Status:         ampo.Status,
			Errors:         ampo.Errors,
			Response:       ampo.AuctionResponse,
			StartTime:      ampo.StartTime,
			SeatNonBid:     ampo.SeatNonBid,
			RequestWrapper: ampo.RequestWrapper,
		}

		mockProcessorService.EXPECT().ProcessLogData(logObject).Return(auctionBids, winningBids).Times(1)
		mockAuctionBidsQueue.EXPECT().Enqueue(*auctionBids).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[0]).Times(1)
		mockWinBidsQueue.EXPECT().Enqueue(winningBids[1]).Times(1)

		p.LogAmpObject(ampo)
	})
}
